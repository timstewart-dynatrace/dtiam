package describe

import (
	"bytes"
	"strings"
	"testing"
)

func TestDescribeCmd_HasSubcommands(t *testing.T) {
	expected := []string{
		"group", "user", "policy", "environment",
		"boundary", "service-user",
	}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in describe command", name)
		}
	}
}

func TestDescribeCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "describe") {
		t.Error("help output should mention 'describe'")
	}
}

func TestDescribeAllSubcommands_RequireExactlyOneArg(t *testing.T) {
	// Skip auto-generated cobra subcommands (completion, help)
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		var buf bytes.Buffer
		Cmd.SetOut(&buf)
		Cmd.SetErr(&buf)
		Cmd.SetArgs([]string{c.Name()})

		err := Cmd.Execute()
		if err == nil {
			t.Errorf("describe %s should require exactly 1 argument", c.Name())
		}
	}
}

func TestDescribeAllSubcommands_HaveExamples(t *testing.T) {
	// Skip auto-generated cobra subcommands (completion, help)
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		if c.Example == "" {
			t.Errorf("describe %s should have example text", c.Name())
		}
	}
}

func TestDescribeEnvironmentCmd_Aliases(t *testing.T) {
	found := false
	for _, a := range environmentCmd.Aliases {
		if a == "env" {
			found = true
			break
		}
	}
	if !found {
		t.Error("environment command should have 'env' alias")
	}
}

func TestDescribeServiceUserCmd_Aliases(t *testing.T) {
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
