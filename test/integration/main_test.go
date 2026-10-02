//go:build integration

// Package integration runs the dtiam binary against a live Dynatrace account.
//
// It is opt-in twice over: the integration build tag, and
//
//	DTIAM_INTEGRATION=1 DTIAM_INTEGRATION_CONTEXT=<context> make test-integration
//
// The context must exist and be at safety level readwrite. Every object the
// suite creates is named dtiam-it-<run>-..., deleted when its test ends, and
// swept on the next run if a crash left it behind. Nothing else on the
// account is modified.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const prefix = "dtiam-it-"

var (
	binary  string // path to the dtiam binary built for this run
	ctxName string // context under test
	runID   string // unique per run, part of every object name
)

func TestMain(m *testing.M) {
	if os.Getenv("DTIAM_INTEGRATION") != "1" {
		fmt.Println("integration: skipped (set DTIAM_INTEGRATION=1 and DTIAM_INTEGRATION_CONTEXT to run)")
		os.Exit(0)
	}
	ctxName = os.Getenv("DTIAM_INTEGRATION_CONTEXT")
	if ctxName == "" {
		fmt.Fprintln(os.Stderr, "integration: DTIAM_INTEGRATION_CONTEXT must name the context to test against")
		os.Exit(1)
	}
	runID = strconv.FormatInt(time.Now().Unix(), 36)

	dir, err := os.MkdirTemp("", "dtiam-it-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration:", err)
		os.Exit(1)
	}
	binary = filepath.Join(dir, "dtiam")
	build := exec.Command("go", "build", "-o", binary, "../../cmd/dtiam")
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "integration: build failed:", err)
		os.Exit(1)
	}

	if err := checkContext(); err != nil {
		fmt.Fprintln(os.Stderr, "integration:", err)
		os.Exit(1)
	}
	sweep()

	code := m.Run()
	sweep() // anything a failed cleanup left behind
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// envelope is the agent-mode output every test reads.
type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Context struct {
		Operation string   `json:"operation"`
		ExitCode  int      `json:"exit_code"`
		Total     *int     `json:"total"`
		Messages  []string `json:"messages"`
		Warnings  []string `json:"warnings"`
	} `json:"context"`
}

// dtiam runs the binary in agent mode against the context under test.
func dtiam(t *testing.T, args ...string) envelope {
	t.Helper()
	env, _ := dtiamWithEnv(t, nil, args...)
	return env
}

func dtiamWithEnv(t *testing.T, extraEnv []string, args ...string) (envelope, int) {
	t.Helper()
	cmd := exec.Command(binary, append([]string{"-A", "--context", ctxName}, args...)...)
	cmd.Env = append(os.Environ(), "DTIAM_NO_AGENT_DETECT=1")
	cmd.Env = append(cmd.Env, extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	exit := 0
	if ee, ok := err.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("dtiam %s: %v", strings.Join(args, " "), err)
	}

	var env envelope
	if jerr := json.Unmarshal(stdout.Bytes(), &env); jerr != nil {
		t.Fatalf("dtiam %s: stdout is not one JSON envelope (%v)\nstdout: %s\nstderr: %s",
			strings.Join(args, " "), jerr, stdout.String(), stderr.String())
	}
	if env.Context.ExitCode != exit {
		t.Errorf("dtiam %s: envelope exit_code %d, process exit %d", strings.Join(args, " "), env.Context.ExitCode, exit)
	}
	return env, exit
}

// mustOK runs a command and fails the test unless it succeeded.
func mustOK(t *testing.T, args ...string) envelope {
	t.Helper()
	env := dtiam(t, args...)
	if !env.OK {
		t.Fatalf("dtiam %s failed: %+v", strings.Join(args, " "), env.Error)
	}
	return env
}

// object decodes a single-object result.
func object(t *testing.T, env envelope) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(env.Result, &m); err != nil {
		var list []map[string]any
		if lerr := json.Unmarshal(env.Result, &list); lerr == nil && len(list) == 1 {
			return list[0]
		}
		t.Fatalf("result is not an object: %s", env.Result)
	}
	return m
}

// list decodes a list result.
func list(t *testing.T, env envelope) []map[string]any {
	t.Helper()
	var l []map[string]any
	if err := json.Unmarshal(env.Result, &l); err != nil {
		t.Fatalf("result is not a list: %s", env.Result)
	}
	return l
}

// name returns a unique object name for this run.
func name(kind string) string { return prefix + runID + "-" + kind }

func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }

// checkContext refuses to run against a context that would block writes or
// that does not exist, so a misconfigured run fails before creating anything.
func checkContext() error {
	out, err := exec.Command(binary, "-A", "config", "get-contexts").Output()
	if err != nil {
		return fmt.Errorf("config get-contexts: %w", err)
	}
	var env struct {
		Result []map[string]any `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return fmt.Errorf("config get-contexts: %w", err)
	}
	for _, c := range env.Result {
		if c["name"] == ctxName {
			if level := c["safety_level"]; level != "readwrite" {
				return fmt.Errorf("context %q is at safety level %v; the suite needs readwrite", ctxName, level)
			}
			return nil
		}
	}
	return fmt.Errorf("context %q not found", ctxName)
}

// sweep deletes leftover dtiam-it-* objects from runs more than an hour old,
// so a crashed run does not accumulate garbage. Recent ones are left alone in
// case another run is in progress.
func sweep() {
	cutoff := time.Now().Add(-time.Hour).Unix()
	stale := func(n string) bool {
		if !strings.HasPrefix(n, prefix) {
			return false
		}
		id := strings.SplitN(strings.TrimPrefix(n, prefix), "-", 2)[0]
		ts, err := strconv.ParseInt(id, 36, 64)
		return err == nil && (ts < cutoff || id == runID)
	}
	run := func(args ...string) []byte {
		out, _ := exec.Command(binary, append([]string{"-A", "--context", ctxName}, args...)...).Output()
		return out
	}
	names := func(args []string, field, idField string) []string {
		var env struct {
			Result []map[string]any `json:"result"`
		}
		_ = json.Unmarshal(run(args...), &env)
		var ids []string
		for _, item := range env.Result {
			if n, _ := item[field].(string); stale(n) {
				id, _ := item[idField].(string)
				ids = append(ids, id)
			}
		}
		return ids
	}
	for _, id := range names([]string{"get", "tokens"}, "name", "tokenId") {
		run("delete", "token", id, "--force")
	}
	for _, id := range names([]string{"get", "groups"}, "name", "uuid") {
		run("delete", "group", id, "--force")
	}
	for _, id := range names([]string{"get", "policies"}, "name", "uuid") {
		run("delete", "policy", id, "--force")
	}
	for _, id := range names([]string{"get", "boundaries"}, "name", "uuid") {
		run("delete", "boundary", id, "--force")
	}
	for _, id := range names([]string{"service-user", "list"}, "name", "uid") {
		run("delete", "service-user", id, "--force")
	}
}

// jsonResult decodes env.Result into v.
func jsonResult(env envelope, v any) error { return json.Unmarshal(env.Result, v) }
