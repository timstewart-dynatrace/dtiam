package boundary

import (
	"bytes"
	"strings"
	"testing"
)

func TestBoundaryCmd_HasSubcommands(t *testing.T) {
	expected := []string{"attach", "detach", "list-attached"}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in boundary command", name)
		}
	}
}

func TestBoundaryCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "boundary") {
		t.Error("help output should mention 'boundary'")
	}
}

func TestBoundaryCmd_HasExample(t *testing.T) {
	if Cmd.Example == "" {
		t.Error("boundary command should have example text")
	}
}

func TestAttachCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"group":    "g",
		"policy":   "p",
		"boundary": "b",
	}

	for name, shorthand := range flags {
		f := attachCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("attach command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}
}

func TestDetachCmd_Flags(t *testing.T) {
	flags := map[string]string{
		"group":    "g",
		"policy":   "p",
		"boundary": "b",
	}

	for name, shorthand := range flags {
		f := detachCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("detach command should have --%s flag", name)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("flag --%s should have shorthand -%s, got -%s", name, shorthand, f.Shorthand)
		}
	}
}

func TestListAttachedCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"list-attached"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to list-attached")
	}
}

func TestBoundaryAllSubcommands_HaveExamples(t *testing.T) {
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		if c.Example == "" {
			t.Errorf("boundary %s should have example text", c.Name())
		}
	}
}
