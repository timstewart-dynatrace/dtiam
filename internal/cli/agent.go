package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/safety"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/version"
)

// EnvAgent enables the agent envelope when set to a non-empty value.
const EnvAgent = "DTIAM_AGENT"

// agentRun holds the state of one agent-mode invocation.
type agentRun struct {
	started    time.Time
	realStdout *os.File
	pipeWriter *os.File
	captured   bytes.Buffer
	done       chan struct{}
	command    *cobra.Command
}

var currentAgentRun *agentRun

// startAgentMode switches the process into agent mode for this invocation.
//
// The printer records into the session, and os.Stdout is redirected into a
// buffer so that any command writing to stdout directly -- several older ones
// still use fmt.Printf -- cannot break the one-document contract. Captured text
// is reported as messages in the envelope.
func startAgentMode(cmd *cobra.Command) {
	run := &agentRun{started: time.Now(), command: cmd, done: make(chan struct{})}
	GlobalState.Agent = true
	GlobalState.Plain = true
	GlobalState.AgentSession = output.NewAgentSession()

	if r, w, err := os.Pipe(); err == nil {
		run.realStdout = os.Stdout
		run.pipeWriter = w
		os.Stdout = w
		go func() {
			_, _ = io.Copy(&run.captured, r)
			close(run.done)
		}()
	}
	currentAgentRun = run
}

// finishAgentMode restores stdout and writes the envelope for err.
// It returns the process exit code.
func finishAgentMode(err error) int {
	run := currentAgentRun
	if run == nil {
		return exitCodeFor(err)
	}
	out := os.Stdout
	if run.pipeWriter != nil {
		_ = run.pipeWriter.Close()
		<-run.done
		os.Stdout = run.realStdout
		out = run.realStdout
	}

	session := GlobalState.AgentSession
	if session == nil {
		session = output.NewAgentSession()
	}
	for _, line := range strings.Split(run.captured.String(), "\n") {
		session.AddMessage(line)
	}

	exit := exitCodeFor(err)
	env := output.AgentEnvelope{
		OK:     err == nil || errors.Is(err, ErrSilentExit),
		Result: session.Result(),
		Context: &output.AgentContext{
			Total:    session.Total(),
			ExitCode: exit,
			Duration: time.Since(run.started).Round(time.Millisecond).String(),
			Messages: session.Messages(),
			Warnings: session.Warnings(),
			DryRun:   GlobalState.IsDryRun(),
			Version:  version.Version,
		},
	}
	if run.command != nil {
		env.Context.Command = run.command.CommandPath()
		if op, ok := EffectiveOperation(run.command); ok {
			env.Context.Operation = string(op)
		}
	}
	if err != nil && !errors.Is(err, ErrSilentExit) {
		env.Error = classifyError(err)
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(env)
	return exit
}

// exitCodeFor maps a command error to the process exit code.
func exitCodeFor(err error) int {
	if err == nil {
		return 0
	}
	return 1
}

// classifyError turns an error into a machine-readable AgentError.
func classifyError(err error) *output.AgentError {
	e := &output.AgentError{Code: "error", Message: err.Error()}

	var blocked *safety.BlockedError
	if errors.As(err, &blocked) {
		e.Code = "safety_blocked"
		e.Suggestions = []string{
			"Use a context whose safety level allows " + string(blocked.Operation) + " operations",
			"Preview the change with --dry-run, which every safety level allows",
		}
		return e
	}

	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		e.StatusCode = apiErr.StatusCode
		e.Code = classifyStatus(apiErr.StatusCode)
		if e.Code == "permission_denied" {
			e.Suggestions = []string{"Check the OAuth client's scopes and the identity's IAM policies ('dtiam auth can-i PERMISSION')"}
		}
		return e
	}

	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "unknown command"), strings.HasPrefix(msg, "unknown flag"),
		strings.HasPrefix(msg, "unknown shorthand flag"), strings.Contains(msg, "arg(s)"),
		strings.HasPrefix(msg, "required flag"), strings.HasPrefix(msg, "invalid argument"):
		e.Code = "usage"
		e.Suggestions = []string{"Run 'dtiam commands -o json' for the full command catalog"}
	case strings.Contains(msg, "not found"):
		e.Code = "not_found"
	case strings.HasPrefix(msg, "permission denied"):
		e.Code = "permission_denied"
	}
	return e
}

// classifyStatus maps an HTTP status to an AgentError code.
func classifyStatus(status int) string {
	switch status {
	case 400:
		return "bad_request"
	case 401:
		return "auth_required"
	case 403:
		return "permission_denied"
	case 404:
		return "not_found"
	case 409:
		return "conflict"
	case 429:
		return "rate_limited"
	}
	if status >= 500 {
		return "server_error"
	}
	return "error"
}
