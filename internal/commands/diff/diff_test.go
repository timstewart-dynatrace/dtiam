package diff

import (
	"strings"
	"testing"
)

func TestParseDocuments_SingleResource(t *testing.T) {
	specs, err := parseDocuments([]byte("kind: Group\nspec:\n  name: A\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want 1", len(specs))
	}
	if specs[0].kind != "Group" || specs[0].name() != "A" {
		t.Errorf("got kind=%q name=%q", specs[0].kind, specs[0].name())
	}
}

func TestParseDocuments_MultipleResources(t *testing.T) {
	content := []byte("kind: Group\nspec:\n  name: A\n---\nkind: Policy\nspec:\n  name: P\n")
	specs, err := parseDocuments(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("got %d specs, want 2", len(specs))
	}
	if specs[1].kind != "Policy" {
		t.Errorf("specs[1].kind = %q, want Policy", specs[1].kind)
	}
}

func TestParseDocuments_AcceptsJSON(t *testing.T) {
	specs, err := parseDocuments([]byte(`{"kind":"Group","spec":{"name":"A"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 || specs[0].kind != "Group" {
		t.Errorf("got %+v", specs)
	}
}

func TestParseDocuments_SkipsBlankDocuments(t *testing.T) {
	specs, err := parseDocuments([]byte("kind: Group\nspec:\n  name: A\n---\n\n---\nkind: Policy\nspec:\n  name: P\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 2 {
		t.Errorf("got %d specs, want 2 with the blank document skipped", len(specs))
	}
}

func TestParseDocuments_Errors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "should reject a missing kind",
			content: "spec:\n  name: A\n",
			wantErr: "missing 'kind'",
		},
		{
			name:    "should reject a missing spec",
			content: "kind: Group\n",
			wantErr: "missing 'spec'",
		},
		{
			name:    "should reject unparseable content",
			content: "\tnot: [valid: yaml",
			wantErr: "failed to parse",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDocuments([]byte(tt.content))
			if err == nil {
				t.Fatalf("expected an error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseDocuments_ReportsTheOffendingDocumentNumber(t *testing.T) {
	// With several documents in a file, the number is the only way to find the
	// bad one.
	_, err := parseDocuments([]byte("kind: Group\nspec:\n  name: A\n---\nkind: Policy\n"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "document 2") {
		t.Errorf("error = %q, want it to name document 2", err)
	}
}

func TestColumns(t *testing.T) {
	cols := Columns()
	want := []string{"kind", "name", "change", "details"}
	if len(cols) != len(want) {
		t.Fatalf("got %d columns, want %d", len(cols), len(want))
	}
	for i, w := range want {
		if cols[i].Key != w {
			t.Errorf("column %d key = %q, want %q", i, cols[i].Key, w)
		}
	}
}

func TestCmdMetadata(t *testing.T) {
	if Cmd.Short == "" || Cmd.Long == "" || Cmd.Example == "" {
		t.Error("Short, Long and Example are all required")
	}
	if Cmd.RunE == nil {
		t.Error("must use RunE")
	}
	for _, flag := range []string{"file", "set", "exit-zero"} {
		if Cmd.Flags().Lookup(flag) == nil {
			t.Errorf("flag --%s is missing", flag)
		}
	}
	// diff is read-only, so --dry-run would be meaningless on it.
	if Cmd.Flags().Lookup("dry-run") != nil {
		t.Error("diff is read-only and should not define its own --dry-run")
	}
}
