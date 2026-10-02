//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadCommands runs every account-wide read command and checks it returns
// a successful envelope. Before 3.0.1 six of these silently returned empty
// lists; a non-empty check guards the ones every account has data for.
func TestReadCommands(t *testing.T) {
	tests := []struct {
		args     []string
		nonEmpty bool
	}{
		{[]string{"get", "groups"}, true},
		{[]string{"get", "users"}, true},
		{[]string{"get", "policies"}, true},
		{[]string{"get", "bindings"}, false},
		{[]string{"get", "boundaries"}, false},
		{[]string{"get", "environments"}, true},
		{[]string{"get", "available-permissions"}, true},
		{[]string{"service-user", "list"}, false},
		{[]string{"account", "limits"}, true},
		{[]string{"account", "subscriptions"}, true},
		{[]string{"account", "notifications"}, false},
		{[]string{"analyze", "permissions-matrix"}, false},
		{[]string{"commands", "--brief"}, false},
		{[]string{"doctor"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.args[0]+" "+tt.args[len(tt.args)-1], func(t *testing.T) {
			env := mustOK(t, tt.args...)
			if env.Context.Operation != "read" {
				t.Errorf("operation = %q, want read", env.Context.Operation)
			}
			if tt.nonEmpty && (env.Context.Total == nil || *env.Context.Total == 0) {
				t.Errorf("expected a non-empty list, got total %v", env.Context.Total)
			}
		})
	}
}

func TestWhoAmIAndCanI(t *testing.T) {
	me := object(t, mustOK(t, "auth", "whoami"))
	if str(me, "uid") == "" {
		t.Fatalf("whoami returned no uid: %v", me)
	}

	// A permission nobody has: a clean "no" is ok=true, exit 1.
	env, exit := dtiamWithEnv(t, nil, "auth", "can-i", "dtiam-it:never:granted")
	if !env.OK || exit != 1 || str(object(t, env), "decision") != "no" {
		t.Errorf("can-i of an unknown permission = ok %v exit %d %s", env.OK, exit, env.Result)
	}
}

func TestErrorEnvelopes(t *testing.T) {
	tests := []struct {
		args []string
		code string
	}{
		{[]string{"get", "groupz"}, "usage"},
		{[]string{"get", "groups", "--no-such-flag"}, "usage"},
		{[]string{"get", "policies", name("does-not-exist")}, "not_found"},
	}
	for _, tt := range tests {
		env, exit := dtiamWithEnv(t, nil, tt.args...)
		if env.OK || exit != 1 || env.Error == nil || env.Error.Code != tt.code {
			t.Errorf("dtiam %v: ok %v exit %d error %+v, want code %s", tt.args, env.OK, exit, env.Error, tt.code)
		}
	}
}

// TestSafetyLevels runs against a copy of the real config with the context
// set to readonly, so the real config is never changed.
func TestSafetyLevels(t *testing.T) {
	src := realConfigPath(t)
	home := t.TempDir()
	dst := filepath.Join(home, "dtiam", "config")
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		t.Fatal(err)
	}
	isolated := []string{"XDG_CONFIG_HOME=" + home}

	if env, _ := dtiamWithEnv(t, isolated, "config", "set-context", ctxName, "--safety-level", "readonly"); !env.OK {
		t.Fatalf("set-context: %+v", env.Error)
	}

	if env, _ := dtiamWithEnv(t, isolated, "get", "groups"); !env.OK {
		t.Errorf("reads must work in readonly: %+v", env.Error)
	}
	env, exit := dtiamWithEnv(t, isolated, "create", "group", "--name", name("blocked"))
	if env.OK || exit != 1 || env.Error == nil || env.Error.Code != "safety_blocked" {
		t.Errorf("create in readonly: ok %v exit %d %+v, want safety_blocked", env.OK, exit, env.Error)
	}
	if env, _ := dtiamWithEnv(t, isolated, "delete", "group", name("blocked"), "--dry-run"); !env.OK {
		t.Errorf("dry runs must work in readonly: %+v", env.Error)
	}

	if env, _ := dtiamWithEnv(t, isolated, "config", "set-context", ctxName, "--safety-level", "no-delete"); !env.OK {
		t.Fatalf("set-context: %+v", env.Error)
	}
	env, _ = dtiamWithEnv(t, isolated, "delete", "group", name("blocked"), "--force")
	if env.Error == nil || env.Error.Code != "safety_blocked" {
		t.Errorf("delete in no-delete: %+v, want safety_blocked", env.Error)
	}
}

// realConfigPath asks the binary where its config lives.
func realConfigPath(t *testing.T) string {
	t.Helper()
	var path string
	if err := jsonResult(mustOK(t, "config", "path"), &path); err != nil || path == "" {
		// config path prints text; the envelope carries it as a message.
		env := mustOK(t, "config", "path")
		for _, m := range env.Context.Messages {
			if filepath.IsAbs(m) {
				return m
			}
		}
		t.Fatalf("could not determine the config path")
	}
	return path
}
