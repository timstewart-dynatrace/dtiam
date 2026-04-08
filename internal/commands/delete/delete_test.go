package delete

import (
	"bytes"
	"strings"
	"testing"
)

func TestDeleteCmd_HasSubcommands(t *testing.T) {
	expected := []string{
		"group", "policy", "binding", "boundary",
		"user", "service-user", "token",
	}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in delete command", name)
		}
	}
}

func TestDeleteCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "delete") || !strings.Contains(output, "Delete") {
		t.Error("help output should mention 'delete'")
	}
}

func TestDeleteGroupCmd_Args(t *testing.T) {
	// group command requires exactly 1 arg
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"group"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to delete group")
	}
}

func TestDeleteGroupCmd_ForceFlag(t *testing.T) {
	f := groupCmd.Flags().Lookup("force")
	if f == nil {
		t.Error("group command should have --force flag")
		return
	}
	if f.Shorthand != "f" {
		t.Errorf("--force should have shorthand -f, got -%s", f.Shorthand)
	}
}

func TestDeletePolicyCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"policy"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to delete policy")
	}
}

func TestDeleteTokenCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"token"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to delete token")
	}
}

func TestDeleteBindingCmd_Flags(t *testing.T) {
	flags := []string{"group", "policy", "force"}
	for _, name := range flags {
		f := bindingCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("binding command should have --%s flag", name)
		}
	}
}

func TestDeleteServiceUserCmd_Aliases(t *testing.T) {
	found := false
	for _, a := range serviceUserCmd.Aliases {
		if a == "serviceuser" {
			found = true
			break
		}
	}
	if !found {
		t.Error("service-user command should have 'serviceuser' alias")
	}
}

func TestDeleteAllSubcommands_HaveForceFlag(t *testing.T) {
	cmdsWithForce := []*struct {
		name string
		flag *string
	}{
		{"group", nil},
		{"policy", nil},
		{"binding", nil},
		{"boundary", nil},
		{"user", nil},
		{"service-user", nil},
		{"token", nil},
	}

	for _, c := range Cmd.Commands() {
		for _, expected := range cmdsWithForce {
			if c.Name() == expected.name {
				f := c.Flags().Lookup("force")
				if f == nil {
					t.Errorf("delete %s should have --force flag", expected.name)
				}
			}
		}
	}
}

func TestDeleteAllSubcommands_HaveExamples(t *testing.T) {
	// Skip auto-generated cobra subcommands (completion, help)
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		if c.Example == "" {
			t.Errorf("delete %s should have example text", c.Name())
		}
	}
}
