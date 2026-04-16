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
