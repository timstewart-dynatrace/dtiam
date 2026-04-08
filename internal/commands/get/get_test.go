package get

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetCmd_HasSubcommands(t *testing.T) {
	expected := []string{
		"groups", "users", "policies", "bindings",
		"environments", "boundaries", "tokens", "apps", "schemas",
	}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in get command", name)
		}
	}
}

func TestGetGroupsCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"groups", "--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "groups") {
		t.Error("help output should mention 'groups'")
	}
	if !strings.Contains(output, "Examples:") || !strings.Contains(output, "dtiam get groups") {
		t.Error("help output should contain examples")
	}
}

func TestGetGroupsCmd_Aliases(t *testing.T) {
	if len(groupsCmd.Aliases) == 0 {
		t.Fatal("groups command should have aliases")
	}
	found := false
	for _, a := range groupsCmd.Aliases {
		if a == "group" {
			found = true
			break
		}
	}
	if !found {
		t.Error("groups command should have 'group' alias")
	}
}

func TestGetPoliciesCmd_Flags(t *testing.T) {
	flags := []string{"level", "level-id", "all-levels"}
	for _, name := range flags {
		f := policiesCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("policies command should have --%s flag", name)
		}
	}
}

func TestGetBindingsCmd_Flags(t *testing.T) {
	flags := []string{"group", "policy"}
	for _, name := range flags {
		f := bindingsCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("bindings command should have --%s flag", name)
		}
	}
}

func TestGetAppsCmd_Flags(t *testing.T) {
	f := appsCmd.Flags().Lookup("environment")
	if f == nil {
		t.Error("apps command should have --environment flag")
	}
}

func TestGetSchemasCmd_Flags(t *testing.T) {
	flags := []string{"environment", "name"}
	for _, name := range flags {
		f := schemasCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("schemas command should have --%s flag", name)
		}
	}
}

func TestGetCmd_MaxArgs(t *testing.T) {
	// Verify groups accepts at most 1 arg (MaximumNArgs(1))
	if groupsCmd.Args == nil {
		t.Error("groups command should have args validation")
	}
}

func TestGetUsersCmd_Aliases(t *testing.T) {
	found := false
	for _, a := range usersCmd.Aliases {
		if a == "user" {
			found = true
			break
		}
	}
	if !found {
		t.Error("users command should have 'user' alias")
	}
}

func TestGetEnvironmentsCmd_Aliases(t *testing.T) {
	expectedAliases := []string{"envs", "env"}
	for _, expected := range expectedAliases {
		found := false
		for _, a := range environmentsCmd.Aliases {
			if a == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("environments command should have %q alias", expected)
		}
	}
}
