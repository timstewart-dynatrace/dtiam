package analyze

import (
	"bytes"
	"strings"
	"testing"
)

func TestAnalyzeCmd_HasSubcommands(t *testing.T) {
	expected := []string{
		"user-permissions", "group-permissions", "permissions-matrix",
		"policy", "least-privilege", "effective-user", "effective-group",
	}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in analyze command", name)
		}
	}
}

func TestAnalyzeCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "analyze") || !strings.Contains(output, "Analyze") {
		t.Error("help output should mention 'analyze'")
	}
}

func TestAnalyzeUserPermissionsCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"user-permissions"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("user-permissions should require exactly 1 argument")
	}
}

func TestAnalyzeUserPermissionsCmd_Flags(t *testing.T) {
	f := userPermissionsCmd.Flags().Lookup("export")
	if f == nil {
		t.Fatal("user-permissions command should have --export flag")
	}
	if f.Shorthand != "e" {
		t.Errorf("--export should have shorthand -e, got -%s", f.Shorthand)
	}
}

func TestAnalyzeGroupPermissionsCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"group-permissions"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("group-permissions should require exactly 1 argument")
	}
}

func TestAnalyzePermissionsMatrixCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"scope":  "s",
		"export": "e",
	}

	for name, shorthand := range flags {
		f := permissionsMatrixCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("permissions-matrix command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}
}

func TestAnalyzePolicyCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"policy"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("analyze policy should require exactly 1 argument")
	}
}

func TestAnalyzeLeastPrivilegeCmd_Flags(t *testing.T) {
	f := leastPrivilegeCmd.Flags().Lookup("export")
	if f == nil {
		t.Error("least-privilege command should have --export flag")
	}
}

func TestAnalyzeEffectiveUserCmd_Flags(t *testing.T) {
	flags := []string{"level", "level-id", "services", "export"}
	for _, name := range flags {
		f := effectiveUserCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("effective-user command should have --%s flag", name)
		}
	}
}

func TestAnalyzeEffectiveGroupCmd_Flags(t *testing.T) {
	flags := []string{"level", "level-id", "services", "export"}
	for _, name := range flags {
		f := effectiveGroupCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("effective-group command should have --%s flag", name)
		}
	}
}

func TestAnalyzeEffectiveUserCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"effective-user"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("effective-user should require exactly 1 argument")
	}
}

func TestAnalyzeEffectiveGroupCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"effective-group"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("effective-group should require exactly 1 argument")
	}
}
