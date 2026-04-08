package user

import (
	"bytes"
	"strings"
	"testing"
)

func TestUserCmd_HasSubcommands(t *testing.T) {
	expected := []string{
		"add-to-groups", "remove-from-groups", "replace-groups",
		"list-groups", "create",
	}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in user command", name)
		}
	}
}

func TestUserCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "user") {
		t.Error("help output should mention 'user'")
	}
}

func TestUserCmd_HasExample(t *testing.T) {
	if Cmd.Example == "" {
		t.Error("user command should have example text")
	}
}

func TestAddToGroupsCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"add-to-groups"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to add-to-groups")
	}
}

func TestAddToGroupsCmd_GroupsFlag(t *testing.T) {
	f := addToGroupsCmd.Flags().Lookup("groups")
	if f == nil {
		t.Error("add-to-groups command should have --groups flag")
		return
	}
	if f.Shorthand != "g" {
		t.Errorf("--groups should have shorthand -g, got -%s", f.Shorthand)
	}
}

func TestRemoveFromGroupsCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"remove-from-groups"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to remove-from-groups")
	}
}

func TestRemoveFromGroupsCmd_GroupsFlag(t *testing.T) {
	f := removeFromGroupsCmd.Flags().Lookup("groups")
	if f == nil {
		t.Error("remove-from-groups command should have --groups flag")
		return
	}
	if f.Shorthand != "g" {
		t.Errorf("--groups should have shorthand -g, got -%s", f.Shorthand)
	}
}

func TestReplaceGroupsCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"replace-groups"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to replace-groups")
	}
}

func TestReplaceGroupsCmd_GroupsFlag(t *testing.T) {
	f := replaceGroupsCmd.Flags().Lookup("groups")
	if f == nil {
		t.Error("replace-groups command should have --groups flag")
		return
	}
	if f.Shorthand != "g" {
		t.Errorf("--groups should have shorthand -g, got -%s", f.Shorthand)
	}
}

func TestListGroupsCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"list-groups"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to list-groups")
	}
}

func TestCreateCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"create"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to create")
	}
}

func TestCreateCmd_Flags(t *testing.T) {
	flags := []string{"first-name", "last-name", "groups"}
	for _, name := range flags {
		f := createCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("create command should have --%s flag", name)
		}
	}
}

func TestCreateCmd_GroupsFlagShorthand(t *testing.T) {
	f := createCmd.Flags().Lookup("groups")
	if f == nil {
		t.Fatal("create command should have --groups flag")
	}
	if f.Shorthand != "g" {
		t.Errorf("--groups should have shorthand -g, got -%s", f.Shorthand)
	}
}

func TestUserAllSubcommands_HaveExamples(t *testing.T) {
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		if c.Example == "" {
			t.Errorf("user %s should have example text", c.Name())
		}
	}
}
