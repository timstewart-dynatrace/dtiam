package edit

import (
	"errors"
	"os"
	"strings"
	"testing"

	diffpkg "github.com/timstewart-dynatrace/dtiam/v3/pkg/diff"
)

// fakePolicy is an in-memory policy resource recording what was saved.
func fakePolicy(live map[string]any, saveErr error) (*resource, *map[string]any) {
	var saved map[string]any
	return &resource{
		kind:     "Policy",
		fields:   []string{"name", "description", "statementQuery", "tags"},
		readOnly: []string{"uuid", "category"},
		defaults: map[string]any{"tags": []any{}},
		load: func(string) (map[string]any, string, error) {
			return live, live["uuid"].(string), nil
		},
		save: func(uuid string, _ map[string]any, spec map[string]any) error {
			saved = spec
			return saveErr
		},
	}, &saved
}

func livePolicy() map[string]any {
	// No description and no tags, as the API returns for a bare policy: the
	// untouched empty fields must not count as changes.
	return map[string]any{
		"uuid":           "p1",
		"name":           "Read Only",
		"statementQuery": "ALLOW iam:policies:read;",
		"category":       "CUSTOM",
	}
}

// replaceIn returns an editor that rewrites old to new in the file.
func replaceIn(old, new string) func(string) error {
	return func(path string) error {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(b), old) {
			return errors.New("test editor: text not found: " + old)
		}
		return os.WriteFile(path, []byte(strings.Replace(string(b), old, new, 1)), 0o600)
	}
}

func opts(editor func(string) error, confirm bool) options {
	return options{
		identifier: "Read Only",
		editor:     editor,
		confirm:    func(diffpkg.ResourceDiff) bool { return confirm },
		report:     func(string, ...any) {},
	}
}

func TestRun_AppliesConfirmedEdit(t *testing.T) {
	r, saved := fakePolicy(livePolicy(), nil)
	d, err := run(r, opts(replaceIn("ALLOW iam:policies:read;", "ALLOW iam:policies:read, iam:bindings:read;"), true))
	if err != nil {
		t.Fatalf("run() error: %v", err)
	}
	if len(d.Fields) != 1 || d.Fields[0].Field != "statementQuery" {
		t.Errorf("diff = %+v, want only statementQuery", d.Fields)
	}
	if (*saved)["statementQuery"] != "ALLOW iam:policies:read, iam:bindings:read;" {
		t.Errorf("saved = %v", *saved)
	}
	for _, f := range []string{"category", "uuid"} {
		if _, ok := (*saved)[f]; ok {
			t.Errorf("read-only field %q was sent", f)
		}
	}
}

func TestRun_NoChangesDoesNotSave(t *testing.T) {
	r, saved := fakePolicy(livePolicy(), nil)
	_, err := run(r, opts(func(string) error { return nil }, true))
	if !errors.Is(err, errNoChanges) {
		t.Fatalf("run() error = %v, want errNoChanges", err)
	}
	if *saved != nil {
		t.Error("an unchanged edit must not be saved")
	}
}

func TestRun_DeclinedKeepsTheFile(t *testing.T) {
	r, saved := fakePolicy(livePolicy(), nil)
	_, err := run(r, opts(replaceIn("name: Read Only", "name: Read Only v2"), false))
	if err == nil || !strings.Contains(err.Error(), "--from-file") {
		t.Fatalf("run() error = %v, want a resume hint", err)
	}
	if *saved != nil {
		t.Error("a declined edit must not be saved")
	}
	assertKeptFile(t, err)
}

func TestRun_SaveFailureKeepsTheFile(t *testing.T) {
	r, _ := fakePolicy(livePolicy(), errors.New("Empty policy statement"))
	_, err := run(r, opts(replaceIn("name: Read Only", "name: Read Only v2"), true))
	if err == nil || !strings.Contains(err.Error(), "Empty policy statement") {
		t.Fatalf("run() error = %v", err)
	}
	assertKeptFile(t, err)
}

func TestRun_RejectsInvalidEdits(t *testing.T) {
	tests := []struct {
		name    string
		editor  func(string) error
		wantErr string
	}{
		{"should reject broken YAML", replaceIn("kind: Policy", "kind: [Policy"), "not valid YAML"},
		{"should reject a changed kind", replaceIn("kind: Policy", "kind: Group"), "kind changed"},
		{"should reject a changed uuid", replaceIn("    uuid: p1", "    uuid: p2"), "metadata.uuid changed"},
		{"should reject a non-editable field", replaceIn("spec:", "spec:\n    category: BUILTIN"), "cannot be edited"},
		{"should reject an empty name", replaceIn("name: Read Only", "name: \"\""), "cannot be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, saved := fakePolicy(livePolicy(), nil)
			_, err := run(r, opts(tt.editor, true))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("run() error = %v, want %q", err, tt.wantErr)
			}
			if *saved != nil {
				t.Error("an invalid edit must not be saved")
			}
			assertKeptFile(t, err)
		})
	}
}

func TestRun_DryRunDoesNotSave(t *testing.T) {
	r, saved := fakePolicy(livePolicy(), nil)
	o := opts(replaceIn("name: Read Only", "name: Read Only v2"), true)
	o.dryRun = true
	d, err := run(r, o)
	if err != nil || !d.HasChanges() {
		t.Fatalf("run() = %+v, %v", d, err)
	}
	if *saved != nil {
		t.Error("dry run must not save")
	}
}

func TestRun_FromFileResumesEdits(t *testing.T) {
	r, saved := fakePolicy(livePolicy(), nil)
	content, err := r.render(livePolicy(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	previous := t.TempDir() + "/prev.yaml"
	resumed := strings.Replace(string(content), "name: Read Only", "name: Resumed Name", 1)
	if err := os.WriteFile(previous, []byte(resumed), 0o600); err != nil {
		t.Fatal(err)
	}

	o := opts(func(string) error { return nil }, true) // reopen, save as-is
	o.fromFile = previous
	if _, err := run(r, o); err != nil {
		t.Fatalf("run() error: %v", err)
	}
	if (*saved)["name"] != "Resumed Name" {
		t.Errorf("saved = %v, want the resumed edit", *saved)
	}
}

func TestRender_ShowsReadOnlyAsComments(t *testing.T) {
	r, _ := fakePolicy(livePolicy(), nil)
	out, err := r.render(livePolicy(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"#   category: CUSTOM (read-only)", "kind: Policy", "uuid: p1", "statementQuery: ALLOW iam:policies:read;"} {
		if !strings.Contains(s, want) {
			t.Errorf("rendered document is missing %q:\n%s", want, s)
		}
	}
}

func TestEditorCommandPrecedence(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "nano")
	if got := editorCommand(); got != "nano" {
		t.Errorf("got %q, want nano", got)
	}
	t.Setenv("VISUAL", "code --wait")
	if got := editorCommand(); got != "code --wait" {
		t.Errorf("got %q, want VISUAL to win", got)
	}
}

// assertKeptFile checks the error names a file that still exists, then removes it.
func assertKeptFile(t *testing.T, err error) {
	t.Helper()
	i := strings.Index(err.Error(), "--from-file ")
	if i < 0 {
		t.Fatalf("error has no --from-file hint: %v", err)
	}
	path := strings.TrimSpace(err.Error()[i+len("--from-file "):])
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("the edits file %s was not kept: %v", path, statErr)
	}
	_ = os.Remove(path)
}
