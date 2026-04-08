package create

import (
	"bytes"
	"strings"
	"testing"
)

func TestCreateCmd_HasSubcommands(t *testing.T) {
	expected := []string{"group", "policy", "binding", "boundary", "token"}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in create command", name)
		}
	}
}

func TestCreateCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "create") {
		t.Error("help output should mention 'create'")
	}
}

func TestCreateGroupCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"name":        "n",
		"description": "d",
	}

	for name, shorthand := range flags {
		f := groupCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("group command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}
}

func TestCreatePolicyCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"name":        "n",
		"statement":   "s",
		"description": "d",
	}

	for name, shorthand := range flags {
		f := policyCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("policy command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}
}

func TestCreateBindingCmd_Flags(t *testing.T) {
	flags := []string{"group", "policy", "boundary", "param"}
	for _, name := range flags {
		f := bindingCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("binding command should have --%s flag", name)
		}
	}
}

func TestCreateBoundaryCmd_Flags(t *testing.T) {
	flags := []string{"name", "zone", "query", "description"}
	for _, name := range flags {
		f := boundaryCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("boundary command should have --%s flag", name)
		}
	}
}

func TestCreateTokenCmd_Flags(t *testing.T) {
	flags := []string{"name", "scopes", "expires-in"}
	for _, name := range flags {
		f := tokenCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("token command should have --%s flag", name)
		}
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"single", "foo", []string{"foo"}},
		{"multiple", "foo,bar,baz", []string{"foo", "bar", "baz"}},
		{"whitespace", " foo , bar , baz ", []string{"foo", "bar", "baz"}},
		{"empty parts", "foo,,bar", []string{"foo", "bar"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitAndTrim(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitAndTrim(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}
