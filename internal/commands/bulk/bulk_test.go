package bulk

import (
	"bytes"
	"strings"
	"testing"
)

func TestBulkCmd_HasSubcommands(t *testing.T) {
	expected := []string{
		"add-users-to-group", "remove-users-from-group",
		"create-groups", "create-bindings", "export-group-members",
	}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in bulk command", name)
		}
	}
}

func TestBulkCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "ulk") {
		t.Error("help output should mention bulk operations")
	}
}

func TestBulkAddUsersToGroupCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"file":        "f",
		"group":       "g",
		"email-field": "e",
	}

	for name, shorthand := range flags {
		f := addUsersToGroupCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("add-users-to-group command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}

	f := addUsersToGroupCmd.Flags().Lookup("continue-on-error")
	if f == nil {
		t.Error("add-users-to-group command should have --continue-on-error flag")
	}
}

func TestBulkRemoveUsersFromGroupCmd_Flags(t *testing.T) {
	flags := []string{"file", "group", "user-field", "continue-on-error", "force"}
	for _, name := range flags {
		f := removeUsersFromGroupCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("remove-users-from-group command should have --%s flag", name)
		}
	}
}

func TestBulkCreateGroupsCmd_Flags(t *testing.T) {
	flags := []string{"file", "continue-on-error"}
	for _, name := range flags {
		f := createGroupsCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("create-groups command should have --%s flag", name)
		}
	}
}

func TestBulkCreateBindingsCmd_Flags(t *testing.T) {
	flags := []string{"file", "continue-on-error"}
	for _, name := range flags {
		f := createBindingsCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("create-bindings command should have --%s flag", name)
		}
	}
}

func TestBulkExportGroupMembersCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"group":  "g",
		"output": "o",
		"format": "F",
	}

	for name, shorthand := range flags {
		f := exportGroupMembersCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("export-group-members command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}
}

func TestBulkNoFlagConflicts(t *testing.T) {
	// This test verifies that the bulk command tree can be fully initialized
	// without panicking due to duplicate flag shorthand conflicts.
	// (Regression test for the -f flag conflict that was previously fixed.)
	for _, c := range Cmd.Commands() {
		_ = c.Flags()
	}
}
