//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// createGroup creates a throwaway group and deletes it when the test ends.
func createGroup(t *testing.T, suffix string) map[string]any {
	t.Helper()
	n := name(suffix)
	mustOK(t, "create", "group", "--name", n, "--description", "dtiam integration test object")
	g := object(t, mustOK(t, "get", "groups", n))
	t.Cleanup(func() { dtiam(t, "delete", "group", str(g, "uuid"), "--force") })
	return g
}

// createServiceUser creates a throwaway service user to act as a group member.
func createServiceUser(t *testing.T) map[string]any {
	t.Helper()
	n := name("su")
	mustOK(t, "service-user", "create", "--name", n, "--description", "dtiam integration test object")
	for _, u := range list(t, mustOK(t, "service-user", "list")) {
		if str(u, "name") == n {
			t.Cleanup(func() { dtiam(t, "delete", "service-user", str(u, "uid"), "--force") })
			return u
		}
	}
	t.Fatalf("service user %s not found after create", n)
	return nil
}

func TestGroupLifecycle(t *testing.T) {
	g := createGroup(t, "group")
	uuid := str(g, "uuid")

	mustOK(t, "group", "update", uuid, "--description", "updated")
	if got := str(object(t, mustOK(t, "get", "groups", uuid)), "description"); got != "updated" {
		t.Errorf("description = %q after group update", got)
	}

	// Membership, through every path, using a throwaway service user.
	su := createServiceUser(t)
	email := str(su, "email")
	members := func() int { return len(list(t, mustOK(t, "group", "members", uuid))) }

	mustOK(t, "group", "add-member", uuid, "--email", email)
	if members() != 1 {
		t.Fatal("group add-member did not add the member")
	}
	mustOK(t, "group", "remove-member", uuid, "--user", str(su, "uid")) // by UID
	if members() != 0 {
		t.Fatal("group remove-member did not remove the member")
	}
	mustOK(t, "service-user", "add-to-group", str(su, "uid"), "--group", uuid)
	groups := list(t, mustOK(t, "service-user", "list-groups", str(su, "uid")))
	found := false
	for _, gr := range groups {
		found = found || str(gr, "uuid") == uuid
	}
	if !found {
		t.Error("service-user list-groups does not show the group it was added to")
	}
	mustOK(t, "user", "remove-from-groups", email, "--groups", uuid)
	if members() != 0 {
		t.Error("user remove-from-groups did not remove the member")
	}
}

func TestPolicyBoundaryBindingLifecycle(t *testing.T) {
	g := createGroup(t, "bind")
	group := str(g, "uuid")

	pname := name("policy")
	mustOK(t, "create", "policy", "--name", pname, "--description", "dtiam integration test object",
		"--statement", `ALLOW settings:objects:read WHERE settings:schemaId = "builtin:dtiam-it-nonexistent";`)
	policy := str(object(t, mustOK(t, "get", "policies", pname)), "uuid")
	t.Cleanup(func() { dtiam(t, "delete", "policy", policy, "--force") })

	var bounds []string
	for _, suffix := range []string{"bound-a", "bound-b"} {
		bname := name(suffix)
		mustOK(t, "create", "boundary", "--name", bname, "--query", `shared:app-id IN ("dtiam.it.nonexistent");`)
		id := str(object(t, mustOK(t, "get", "boundaries", bname)), "uuid")
		bounds = append(bounds, id)
		t.Cleanup(func() { dtiam(t, "delete", "boundary", id, "--force") })
	}

	boundaries := func() []any {
		for _, b := range list(t, mustOK(t, "get", "bindings", "--group", group)) {
			if str(b, "policyUuid") == policy {
				l, _ := b["boundaries"].([]any)
				return l
			}
		}
		t.Fatal("binding not found")
		return nil
	}

	mustOK(t, "create", "binding", "--group", group, "--policy", policy, "--boundary", bounds[0])
	t.Cleanup(func() { dtiam(t, "delete", "binding", "--group", group, "--policy", policy, "--force") })

	// Attach keeps existing boundaries; detach removes only the named one.
	mustOK(t, "boundary", "attach", "-g", group, "-p", policy, "-b", bounds[1])
	if n := len(boundaries()); n != 2 {
		t.Fatalf("after attach: %d boundaries, want 2", n)
	}
	mustOK(t, "boundary", "detach", "-g", group, "-p", policy, "-b", bounds[0])
	if b := boundaries(); len(b) != 1 || b[0] != bounds[1] {
		t.Fatalf("after detach: %v, want only %s", b, bounds[1])
	}

	if desc := object(t, mustOK(t, "describe", "group", group)); desc["policy_count"] != float64(1) {
		t.Errorf("describe group policy_count = %v, want 1", desc["policy_count"])
	}

	mustOK(t, "delete", "binding", "--group", group, "--policy", policy, "--force")
	if len(list(t, mustOK(t, "get", "bindings", "--group", group))) != 0 {
		t.Error("delete binding left the binding in place")
	}
}

func TestApplyAndDiffAreIdempotent(t *testing.T) {
	g := createGroup(t, "apply")
	file := filepath.Join(t.TempDir(), "group.yaml")
	spec := "kind: Group\nspec:\n  name: " + str(g, "name") + "\n  description: set by apply\n"
	if err := os.WriteFile(file, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	if env, exit := dtiamWithEnv(t, nil, "diff", "-f", file); !env.OK || exit != 1 {
		t.Errorf("diff before apply: ok %v exit %d, want drift (exit 1)", env.OK, exit)
	}
	mustOK(t, "apply", "-f", file)
	if env, exit := dtiamWithEnv(t, nil, "diff", "-f", file); !env.OK || exit != 0 {
		t.Errorf("diff after apply: ok %v exit %d, want no drift", env.OK, exit)
	}
	env := mustOK(t, "apply", "-f", file)
	if !strings.Contains(strings.Join(env.Context.Messages, " "), "unchanged") {
		t.Errorf("second apply should report unchanged: %v", env.Context.Messages)
	}
}

func TestTokenLifecycle(t *testing.T) {
	me := str(object(t, mustOK(t, "auth", "whoami")), "uid")
	tname := name("token")
	// The API only mints tokens owned by the caller. One read scope, one hour.
	mustOK(t, "create", "token", "--name", tname, "--user", me,
		"--scopes", "app-settings:objects:read", "--expires-in", "1h")

	var id string
	status := func() map[string]any {
		for _, tok := range list(t, mustOK(t, "get", "tokens")) {
			if str(tok, "name") == tname {
				return tok
			}
		}
		t.Fatal("token not found")
		return nil
	}
	id = str(status(), "tokenId")
	t.Cleanup(func() { dtiam(t, "delete", "token", id, "--force") })

	mustOK(t, "token", "deactivate", id, "--force")
	if s := str(status(), "status"); s != "INACTIVE" {
		t.Errorf("after deactivate: %s", s)
	}
	mustOK(t, "token", "activate", id)
	if s := str(status(), "status"); s != "ACTIVE" {
		t.Errorf("after activate: %s", s)
	}
	expiry := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second).Format(time.RFC3339)
	mustOK(t, "token", "set-expiration", id, "--date", expiry)
	if got := str(status(), "expirationDate"); got != expiry {
		t.Errorf("expirationDate = %s, want %s", got, expiry)
	}
}

// TestEditPolicy drives "dtiam edit" for real: script(1) provides the
// terminal edit requires, and EDITOR is a sed one-liner.
func TestEditPolicy(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script(1) not available to provide a terminal")
	}
	pname := name("edit")
	mustOK(t, "create", "policy", "--name", pname, "--description", "before",
		"--statement", `ALLOW settings:objects:read WHERE settings:schemaId = "builtin:dtiam-it-nonexistent";`)
	policy := str(object(t, mustOK(t, "get", "policies", pname)), "uuid")
	t.Cleanup(func() { dtiam(t, "delete", "policy", policy, "--force") })

	editor := `sed -i.bak 's/description: before/description: after/'`
	args := []string{binary, "--context", ctxName, "edit", "policy", policy, "--force"}
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("script", append([]string{"-q", "/dev/null"}, args...)...)
	} else {
		cmd = exec.Command("script", "-qec", strings.Join(quoteAll(args), " "), "/dev/null")
	}
	cmd.Env = append(os.Environ(), "EDITOR="+editor, "VISUAL=", "DTIAM_NO_AGENT_DETECT=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dtiam edit: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "updated") {
		t.Errorf("edit output: %s", out)
	}
	if got := str(object(t, mustOK(t, "get", "policies", policy)), "description"); got != "after" {
		t.Errorf("description = %q after edit, want after", got)
	}
}

func quoteAll(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return out
}
