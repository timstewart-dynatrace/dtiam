package config

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfigCmd_HasSubcommands(t *testing.T) {
	expected := []string{
		"view", "path", "get-contexts", "current-context",
		"use-context", "set-context", "delete-context",
		"set-credentials", "delete-credentials", "get-credentials",
	}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in config command", name)
		}
	}
}

func TestConfigCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "config") {
		t.Error("help output should mention 'config'")
	}
}

func TestConfigViewCmd_Flags(t *testing.T) {
	f := viewCmd.Flags().Lookup("show-secrets")
	if f == nil {
		t.Error("view command should have --show-secrets flag")
	}
}

func TestConfigSetContextCmd_Flags(t *testing.T) {
	flags := []string{"account-uuid", "credentials-ref"}
	for _, name := range flags {
		f := setContextCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("set-context command should have --%s flag", name)
		}
	}
}

func TestConfigSetCredentialsCmd_Flags(t *testing.T) {
	flags := []string{"client-id", "client-secret"}
	for _, name := range flags {
		f := setCredentialsCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("set-credentials command should have --%s flag", name)
		}
	}
}

func TestConfigUseContextCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"use-context"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("use-context should require exactly 1 argument")
	}
}

func TestConfigGetContextsCmd_Aliases(t *testing.T) {
	found := false
	for _, a := range getContextsCmd.Aliases {
		if a == "contexts" {
			found = true
			break
		}
	}
	if !found {
		t.Error("get-contexts command should have 'contexts' alias")
	}
}

func TestConfigAllSubcommands_HaveExamples(t *testing.T) {
	// Skip auto-generated cobra subcommands (completion, help)
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		if c.Example == "" {
			t.Errorf("config %s should have example text", c.Name())
		}
	}
}
