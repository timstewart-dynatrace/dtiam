package serviceuser

import (
	"bytes"
	"strings"
	"testing"
)

func TestServiceUserCmd_HasSubcommands(t *testing.T) {
	expected := []string{
		"list", "get", "create", "update", "delete",
		"add-to-group", "remove-from-group", "list-groups",
	}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in service-user command", name)
		}
	}
}

func TestServiceUserCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "service-user") {
		t.Error("help output should mention 'service-user'")
	}
}

func TestServiceUserCmd_HasExample(t *testing.T) {
	if Cmd.Example == "" {
		t.Error("service-user command should have example text")
	}
}

func TestServiceUserCmd_Aliases(t *testing.T) {
	found := false
	for _, a := range Cmd.Aliases {
		if a == "serviceuser" {
			found = true
			break
		}
	}
	if !found {
		t.Error("service-user command should have 'serviceuser' alias")
	}
}

func TestGetCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"get"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to get")
	}
}

func TestCreateCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"name":        "n",
		"description": "d",
		"groups":      "g",
	}

	for name, shorthand := range flags {
		f := createCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("create command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}
}

func TestUpdateCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"update"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to update")
	}
}

func TestUpdateCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"name":        "n",
		"description": "d",
	}

	for name, shorthand := range flags {
		f := updateCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("update command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}
}

func TestDeleteCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"delete"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to delete")
	}
}

func TestAddToGroupCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"add-to-group"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to add-to-group")
	}
}

func TestAddToGroupCmd_GroupFlag(t *testing.T) {
	f := addToGroupCmd.Flags().Lookup("group")
	if f == nil {
		t.Error("add-to-group command should have --group flag")
		return
	}
	if f.Shorthand != "g" {
		t.Errorf("--group should have shorthand -g, got -%s", f.Shorthand)
	}
}

func TestRemoveFromGroupCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"remove-from-group"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to remove-from-group")
	}
}

func TestRemoveFromGroupCmd_GroupFlag(t *testing.T) {
	f := removeFromGroupCmd.Flags().Lookup("group")
	if f == nil {
		t.Error("remove-from-group command should have --group flag")
		return
	}
	if f.Shorthand != "g" {
		t.Errorf("--group should have shorthand -g, got -%s", f.Shorthand)
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

func TestServiceUserAllSubcommands_HaveExamples(t *testing.T) {
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		if c.Example == "" {
			t.Errorf("service-user %s should have example text", c.Name())
		}
	}
}
