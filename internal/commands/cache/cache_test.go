package cache

import (
	"bytes"
	"strings"
	"testing"
)

func TestCacheCmd_HasSubcommands(t *testing.T) {
	expected := []string{"clear", "stats"}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in cache command", name)
		}
	}
}

func TestCacheCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "cache") {
		t.Error("help output should mention 'cache'")
	}
}

func TestCacheCmd_HasExample(t *testing.T) {
	if Cmd.Example == "" {
		t.Error("cache command should have example text")
	}
}

func TestCacheClearCmd_Runs(t *testing.T) {
	// clear is a no-op in the Go implementation, so it should succeed
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"clear"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("cache clear should not return error: %v", err)
	}
}

func TestCacheStatsCmd_Runs(t *testing.T) {
	// stats is a no-op in the Go implementation, so it should succeed
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"stats"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("cache stats should not return error: %v", err)
	}
}

func TestCacheAllSubcommands_HaveExamples(t *testing.T) {
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		if c.Example == "" {
			t.Errorf("cache %s should have example text", c.Name())
		}
	}
}
