package account

import (
	"bytes"
	"strings"
	"testing"
)

func TestAccountCmd_HasSubcommands(t *testing.T) {
	expected := []string{"limits", "check-capacity", "subscriptions", "forecast"}

	subcmds := Cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcmds {
		names[c.Name()] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found in account command", name)
		}
	}
}

func TestAccountCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"--help"})

	err := Cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "account") {
		t.Error("help output should mention 'account'")
	}
}

func TestAccountCmd_HasExample(t *testing.T) {
	if Cmd.Example == "" {
		t.Error("account command should have example text")
	}
}

func TestLimitsCmd_SummaryFlag(t *testing.T) {
	f := limitsCmd.Flags().Lookup("summary")
	if f == nil {
		t.Error("limits command should have --summary flag")
	}
}

func TestCheckCapacityCmd_Args(t *testing.T) {
	var buf bytes.Buffer
	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"check-capacity"})

	err := Cmd.Execute()
	if err == nil {
		t.Error("expected error when no args provided to check-capacity")
	}
}

func TestCheckCapacityCmd_AdditionalFlag(t *testing.T) {
	f := checkCapacityCmd.Flags().Lookup("additional")
	if f == nil {
		t.Error("check-capacity command should have --additional flag")
		return
	}
	if f.DefValue != "1" {
		t.Errorf("--additional default should be 1, got %s", f.DefValue)
	}
}

func TestSubscriptionsCmd_AcceptsOptionalArg(t *testing.T) {
	// subscriptions uses MaximumNArgs(1) — should accept 0 or 1 args
	if subscriptionsCmd.Args == nil {
		t.Error("subscriptions command should have args validation")
	}
}

func TestForecastCmd_AcceptsOptionalArg(t *testing.T) {
	// forecast uses MaximumNArgs(1) — should accept 0 or 1 args
	if forecastCmd.Args == nil {
		t.Error("forecast command should have args validation")
	}
}

func TestAccountAllSubcommands_HaveExamples(t *testing.T) {
	skip := map[string]bool{"completion": true, "help": true}
	for _, c := range Cmd.Commands() {
		if skip[c.Name()] {
			continue
		}
		if c.Example == "" {
			t.Errorf("account %s should have example text", c.Name())
		}
	}
}
