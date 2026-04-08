package group

import (
	"bytes"
	"strings"
	"testing"
)

func TestGroupCmd_HasSubcommands(t *testing.T) {
	expected := []string{"members", "add-member", "remove-member", "bindings"}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in group command", name)
		}
	}
}

func TestGroupCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "group") {
		t.Error("help output should mention 'group'")
	}
}

func TestGroupCmd_HasExample(t *testing.T) {
	if Cmd.Example == "" {
		t.Error("group command should have example text")
	}
}

func TestMembersCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"members"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to members")
	}
}

func TestAddMemberCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"add-member"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to add-member")
	}
}

func TestAddMemberCmd_EmailFlag(t *testing.T) {
	f := addMemberCmd.Flags().Lookup("email")
	if f == nil {
		t.Error("add-member command should have --email flag")
		return
	}
	if f.Shorthand != "e" {
		t.Errorf("--email should have shorthand -e, got -%s", f.Shorthand)
	}
}

func TestRemoveMemberCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"remove-member"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to remove-member")
	}
}

func TestRemoveMemberCmd_UserFlag(t *testing.T) {
	f := removeMemberCmd.Flags().Lookup("user")
	if f == nil {
		t.Error("remove-member command should have --user flag")
		return
	}
	if f.Shorthand != "u" {
		t.Errorf("--user should have shorthand -u, got -%s", f.Shorthand)
	}
}

func TestBindingsCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"bindings"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to bindings")
	}
}

func TestGroupAllSubcommands_HaveExamples(t *testing.T) {
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		if c.Example == "" {
			t.Errorf("group %s should have example text", c.Name())
		}
	}
}
