package group

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
)

func runUpdate(t *testing.T, args ...string) error {
	t.Helper()
	updateCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	if err := updateCmd.ParseFlags(args); err != nil {
		return err
	}
	return updateCmd.RunE(updateCmd, updateCmd.Flags().Args())
}

func TestUpdateCmd(t *testing.T) {
	prev := cli.GlobalState.DryRun
	cli.GlobalState.DryRun = true // never reach the API from these tests
	t.Cleanup(func() { cli.GlobalState.DryRun = prev })

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "should require something to change", args: []string{"Team"}, wantErr: "nothing to update"},
		{name: "should reject an empty name", args: []string{"Team", "--name", ""}, wantErr: "cannot be empty"},
		{name: "should accept a rename in dry-run", args: []string{"Team", "--name", "Platform"}},
		{name: "should accept clearing the description", args: []string{"Team", "--description", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runUpdate(t, tt.args...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateCmd_IsRegistered(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() == "update" {
			return
		}
	}
	t.Fatal("group update is not registered")
}
