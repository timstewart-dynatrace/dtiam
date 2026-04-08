package export

import (
	"bytes"
	"strings"
	"testing"
)

func TestExportCmd_HasSubcommands(t *testing.T) {
	expected := []string{"all", "group", "policy"}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in export command", name)
		}
	}
}

func TestExportCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "export") || !strings.Contains(output, "Export") {
		t.Error("help output should mention 'export'")
	}
}

func TestExportAllCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"output":  "o",
		"format":  "f",
		"prefix":  "p",
		"include": "i",
		"detailed": "d",
	}

	for name, shorthand := range flags {
		f := allCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("export all command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}

	f := allCmd.Flags().Lookup("timestamp-dir")
	if f == nil {
		t.Error("export all command should have --timestamp-dir flag")
	}
}

func TestExportGroupCmd_Flags(t *testing.T) {
	flags := []string{"output", "format", "include-members", "include-policies"}
	for _, name := range flags {
		f := groupCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("export group command should have --%s flag", name)
		}
	}
}

func TestExportGroupCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"group"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("export group should require exactly 1 argument")
	}
}

func TestExportPolicyCmd_Flags(t *testing.T) {
	flags := []string{"output", "format", "as-template"}
	for _, name := range flags {
		f := policyCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("export policy command should have --%s flag", name)
		}
	}
}

func TestExportPolicyCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"policy"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("export policy should require exactly 1 argument")
	}
}

func TestSplitAndTrim_Export(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{"empty", "", ",", nil},
		{"single", "groups", ",", []string{"groups"}},
		{"multiple", "groups,policies,users", ",", []string{"groups", "policies", "users"}},
		{"with spaces", " groups , policies ", ",", []string{"groups", "policies"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input, tt.sep)
			if len(result) != len(tt.expected) {
				t.Errorf("splitAndTrim(%q, %q) = %v, want %v", tt.input, tt.sep, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitAndTrim(%q, %q)[%d] = %q, want %q", tt.input, tt.sep, i, v, tt.expected[i])
				}
			}
		})
	}
}
