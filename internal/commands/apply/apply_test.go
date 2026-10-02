package apply

import (
	"testing"
)

func TestApplyCmd_HasFileFlag(t *testing.T) {
	f := Cmd.Flags().Lookup("file")
	if f == nil {
		t.Fatal("apply command should have --file flag")
	}
	if f.Shorthand != "f" {
		t.Errorf("--file should have shorthand -f, got -%s", f.Shorthand)
	}
}

func TestApplyCmd_HasSetFlag(t *testing.T) {
	f := Cmd.Flags().Lookup("set")
	if f == nil {
		t.Error("apply command should have --set flag")
	}
}

func TestApplyCmd_HasExample(t *testing.T) {
	if Cmd.Example == "" {
		t.Error("apply command should have example text")
	}
}

func TestSplitYAMLDocuments_Single(t *testing.T) {
	content := []byte("kind: Group\nspec:\n  name: Test\n")
	docs := splitYAMLDocuments(content)
	if len(docs) != 1 {
		t.Errorf("splitYAMLDocuments() returned %d docs, want 1", len(docs))
	}
}

func TestSplitYAMLDocuments_Multiple(t *testing.T) {
	content := []byte("kind: Group\nspec:\n  name: G1\n---\nkind: Policy\nspec:\n  name: P1\n")
	docs := splitYAMLDocuments(content)
	if len(docs) != 2 {
		t.Errorf("splitYAMLDocuments() returned %d docs, want 2", len(docs))
	}
}

func TestSplitYAMLDocuments_LeadingSeparator(t *testing.T) {
	content := []byte("---\nkind: Group\nspec:\n  name: Test\n")
	docs := splitYAMLDocuments(content)
	if len(docs) != 1 {
		t.Errorf("splitYAMLDocuments() returned %d docs, want 1", len(docs))
	}
}

func TestSplitYAMLDocuments_Empty(t *testing.T) {
	docs := splitYAMLDocuments([]byte(""))
	if len(docs) != 0 {
		t.Errorf("splitYAMLDocuments() returned %d docs, want 0", len(docs))
	}
}

func TestMergePolicy(t *testing.T) {
	live := map[string]any{
		"uuid":           "p1",
		"name":           "Read Only",
		"description":    "old",
		"statementQuery": "ALLOW iam:policies:read;",
		"category":       "CUSTOM",
	}

	tests := []struct {
		name string
		spec map[string]any
		want map[string]any
	}{
		{
			name: "should keep the live statement when the spec changes only the description",
			spec: map[string]any{"name": "Read Only", "description": "new"},
			want: map[string]any{"name": "Read Only", "description": "new", "statementQuery": "ALLOW iam:policies:read;"},
		},
		{
			name: "should take the statement from the spec when given",
			spec: map[string]any{"name": "Read Only", "statementQuery": "ALLOW iam:bindings:read;"},
			want: map[string]any{"name": "Read Only", "description": "old", "statementQuery": "ALLOW iam:bindings:read;"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergePolicy(live, tt.spec)
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("mergePolicy()[%s] = %v, want %v", k, got[k], v)
				}
			}
			// PUT accepts only these fields; read-only ones must not be sent.
			for _, k := range []string{"uuid", "category"} {
				if _, ok := got[k]; ok {
					t.Errorf("mergePolicy() included read-only field %q", k)
				}
			}
		})
	}
}

func TestMergePolicy_DefaultsMissingDescription(t *testing.T) {
	got := mergePolicy(map[string]any{"name": "x", "statementQuery": "ALLOW a:b:c;"}, map[string]any{"name": "x"})
	if got["description"] != "" {
		t.Errorf("description = %v, want empty string (PUT requires the field)", got["description"])
	}
}
