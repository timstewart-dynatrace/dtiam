package token

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
)

func run(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	if err := cmd.ParseFlags(args); err != nil {
		return err
	}
	if err := cmd.ValidateArgs(cmd.Flags().Args()); err != nil {
		return err
	}
	return cmd.RunE(cmd, cmd.Flags().Args())
}

func TestTokenCmd_Subcommands(t *testing.T) {
	want := map[string]bool{"activate": false, "deactivate": false, "set-expiration": false}
	for _, c := range Cmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
		if c.Example == "" || c.Long == "" {
			t.Errorf("%s is missing Long or Example help", c.Name())
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("token %s is not registered", name)
		}
	}
	if f := deactivateCmd.Flags().Lookup("force"); f == nil || f.Shorthand != "f" {
		t.Error("deactivate needs --force/-f")
	}
}

func TestTokenCmds_DryRunAndValidation(t *testing.T) {
	prev := cli.GlobalState.DryRun
	cli.GlobalState.DryRun = true // never reach the API from these tests
	t.Cleanup(func() { cli.GlobalState.DryRun = prev })

	tests := []struct {
		name    string
		cmd     *cobra.Command
		args    []string
		wantErr string
	}{
		{name: "should preview a deactivation", cmd: deactivateCmd, args: []string{"dt0s16.A"}},
		{name: "should preview an activation", cmd: activateCmd, args: []string{"dt0s16.A"}},
		{name: "should require a token ID", cmd: deactivateCmd, args: nil, wantErr: "arg"},
		{name: "should preview a valid expiration", cmd: setExpirationCmd, args: []string{"dt0s16.A", "--date", "2027-01-01T00:00:00Z"}},
		{
			// Validated before the dry-run check, so a preview never approves
			// a date the real call would reject.
			name: "should reject a bad date even in dry-run", cmd: setExpirationCmd,
			args: []string{"dt0s16.A", "--date", "2027-01-01"}, wantErr: "RFC 3339",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(t, tt.cmd, tt.args...)
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
