package template

import (
	"strings"
	"testing"
)

func TestRenderTemplate_SimpleVariable(t *testing.T) {
	content := `name: "{{.name}}"`
	result, err := RenderTemplate(content, map[string]string{"name": "TestGroup"})
	if err != nil {
		t.Fatalf("RenderTemplate() error: %v", err)
	}
	if !strings.Contains(result, "TestGroup") {
		t.Errorf("RenderTemplate() = %q, want to contain 'TestGroup'", result)
	}
}

func TestRenderTemplate_DefaultFunction(t *testing.T) {
	content := `desc: "{{.description | default "fallback"}}"`

	// With value provided
	result, err := RenderTemplate(content, map[string]string{"description": "custom"})
	if err != nil {
		t.Fatalf("RenderTemplate() error: %v", err)
	}
	if !strings.Contains(result, "custom") {
		t.Errorf("RenderTemplate() = %q, want 'custom'", result)
	}

	// With empty value (should use default)
	result, err = RenderTemplate(content, map[string]string{"description": ""})
	if err != nil {
		t.Fatalf("RenderTemplate() error: %v", err)
	}
	if !strings.Contains(result, "fallback") {
		t.Errorf("RenderTemplate() = %q, want 'fallback'", result)
	}
}

func TestRenderTemplate_MultipleVariables(t *testing.T) {
	content := `name: "{{.name}}"
description: "{{.description}}"`
	vars := map[string]string{"name": "Group1", "description": "My group"}
	result, err := RenderTemplate(content, vars)
	if err != nil {
		t.Fatalf("RenderTemplate() error: %v", err)
	}
	if !strings.Contains(result, "Group1") || !strings.Contains(result, "My group") {
		t.Errorf("RenderTemplate() = %q, missing variables", result)
	}
}

func TestRenderTemplate_InvalidTemplate(t *testing.T) {
	_, err := RenderTemplate("{{.unclosed", map[string]string{})
	if err == nil {
		t.Error("RenderTemplate() expected error for invalid template")
	}
}

func TestParseSetFlags_Valid(t *testing.T) {
	flags := []string{"name=MyGroup", "desc=A description with = sign"}
	vars, err := ParseSetFlags(flags)
	if err != nil {
		t.Fatalf("ParseSetFlags() error: %v", err)
	}
	if vars["name"] != "MyGroup" {
		t.Errorf("ParseSetFlags() name = %q, want 'MyGroup'", vars["name"])
	}
	if vars["desc"] != "A description with = sign" {
		t.Errorf("ParseSetFlags() desc = %q, want 'A description with = sign'", vars["desc"])
	}
}

func TestParseSetFlags_Invalid(t *testing.T) {
	_, err := ParseSetFlags([]string{"no-equals"})
	if err == nil {
		t.Error("ParseSetFlags() expected error for missing =")
	}
}

func TestParseSetFlags_Empty(t *testing.T) {
	vars, err := ParseSetFlags(nil)
	if err != nil {
		t.Fatalf("ParseSetFlags() error: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("ParseSetFlags(nil) returned %d vars, want 0", len(vars))
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{"simple", `{{.name}}`, []string{"name"}},
		{"with default", `{{.name | default "x"}}`, []string{"name"}},
		{"multiple", `{{.name}} and {{.description}}`, []string{"name", "description"}},
		{"deduped", `{{.name}} {{.name}}`, []string{"name"}},
		{"no variables", `static content`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVariables(tt.content)
			if len(got) != len(tt.expected) {
				t.Errorf("ExtractVariables() = %v, want %v", got, tt.expected)
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("ExtractVariables()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestListBuiltin(t *testing.T) {
	templates := ListBuiltin()
	if len(templates) < 5 {
		t.Errorf("ListBuiltin() returned %d templates, want at least 5", len(templates))
	}

	names := make(map[string]bool)
	for _, tmpl := range templates {
		names[tmpl.Name] = true
		if tmpl.Source != "builtin" {
			t.Errorf("ListBuiltin() template %q source = %q, want 'builtin'", tmpl.Name, tmpl.Source)
		}
	}

	expected := []string{"group-team", "policy-readonly", "policy-admin", "binding-simple", "boundary-mz"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("ListBuiltin() missing expected template %q", name)
		}
	}
}

func TestGetBuiltin_Exists(t *testing.T) {
	content, err := GetBuiltin("group-team")
	if err != nil {
		t.Fatalf("GetBuiltin() error: %v", err)
	}
	if len(content) == 0 {
		t.Error("GetBuiltin() returned empty content")
	}
	if !strings.Contains(string(content), "kind: Group") {
		t.Error("GetBuiltin() content should contain 'kind: Group'")
	}
}

func TestGetBuiltin_NotFound(t *testing.T) {
	_, err := GetBuiltin("nonexistent")
	if err == nil {
		t.Error("GetBuiltin() expected error for missing template")
	}
}
