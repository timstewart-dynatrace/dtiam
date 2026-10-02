package get

import (
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/watch"
)

func TestAddWatchFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addWatchFlags(cmd)

	w := cmd.Flags().Lookup("watch")
	if w == nil {
		t.Fatal("--watch flag not registered")
	}
	if w.Shorthand != "w" {
		t.Errorf("--watch shorthand = %q, want 'w'", w.Shorthand)
	}
	if cmd.Flags().Lookup("watch-interval") == nil {
		t.Error("--watch-interval flag not registered")
	}
}

func TestWatchRequested(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addWatchFlags(cmd)

	if watchRequested(cmd) {
		t.Error("watchRequested() = true before the flag is set")
	}
	if err := cmd.Flags().Set("watch", "true"); err != nil {
		t.Fatal(err)
	}
	if !watchRequested(cmd) {
		t.Error("watchRequested() = false after --watch was set")
	}
}

func TestEffectiveInterval(t *testing.T) {
	// The startup message must not promise an interval the watch will not use.
	tests := []struct {
		name  string
		given time.Duration
		want  time.Duration
	}{
		{name: "zero reports the default", given: 0, want: watch.DefaultInterval},
		{name: "below minimum reports the default", given: time.Millisecond, want: watch.DefaultInterval},
		{name: "valid interval reports itself", given: 30 * time.Second, want: 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveInterval(tt.given); got != tt.want {
				t.Errorf("effectiveInterval(%v) = %v, want %v", tt.given, got, tt.want)
			}
		})
	}
}

func TestWatchFlagsRegisteredOnTheRightCommands(t *testing.T) {
	// Offered on collections worth monitoring during a change.
	for _, c := range []*cobra.Command{groupsCmd, usersCmd, policiesCmd, bindingsCmd} {
		t.Run(c.Name()+" has --watch", func(t *testing.T) {
			if c.Flags().Lookup("watch") == nil {
				t.Errorf("%s is missing --watch", c.Name())
			}
		})
	}

	// Not offered where it would be pointless: reference data and environments
	// do not change on a human timescale.
	for _, c := range []*cobra.Command{availablePermissionsCmd, environmentsCmd} {
		t.Run(c.Name()+" has no --watch", func(t *testing.T) {
			if c.Flags().Lookup("watch") != nil {
				t.Errorf("%s should not offer --watch", c.Name())
			}
		})
	}
}
