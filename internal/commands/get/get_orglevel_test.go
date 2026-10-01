package get

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestEnvironmentIDFromURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "should extract from an apps URL", input: "https://abc12345.apps.dynatrace.com", want: "abc12345"},
		{name: "should extract from a live URL", input: "https://abc12345.live.dynatrace.com", want: "abc12345"},
		{name: "should extract with a trailing path", input: "https://abc12345.apps.dynatrace.com/platform", want: "abc12345"},
		{name: "should handle http", input: "http://abc12345.apps.dynatrace.com", want: "abc12345"},
		{name: "should pass a bare ID through", input: "abc12345", want: "abc12345"},
		{name: "should return empty for empty input", input: "", want: ""},
		{name: "should handle a sprint host", input: "https://xyz789.sprint.dynatracelabs.com", want: "xyz789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := environmentIDFromURL(tt.input); got != tt.want {
				t.Errorf("environmentIDFromURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOrgLevelTarget(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		accountUUID   string
		envURL        string
		wantLevelType string
		wantLevelID   string
		wantErr       bool
	}{
		{
			name: "should default to environment level", level: "", accountUUID: "acct-1",
			envURL:        "https://abc12345.apps.dynatrace.com",
			wantLevelType: "environment", wantLevelID: "abc12345",
		},
		{
			name: "should use the account UUID at account level", level: "account", accountUUID: "acct-1",
			envURL: "", wantLevelType: "account", wantLevelID: "acct-1",
		},
		{
			name: "should reject an invalid level", level: "global", accountUUID: "acct-1",
			envURL: "https://abc12345.apps.dynatrace.com", wantErr: true,
		},
		{
			name: "should error at environment level with no environment URL", level: "environment",
			accountUUID: "acct-1", envURL: "", wantErr: true,
		},
		{
			name: "should error at account level with no account UUID", level: "account",
			accountUUID: "", envURL: "", wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newLevelFlagCommand(tt.level)
			levelType, levelID, err := orgLevelTarget(cmd, tt.accountUUID, tt.envURL)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if levelType != tt.wantLevelType {
				t.Errorf("levelType = %q, want %q", levelType, tt.wantLevelType)
			}
			if levelID != tt.wantLevelID {
				t.Errorf("levelID = %q, want %q", levelID, tt.wantLevelID)
			}
		})
	}
}

// newLevelFlagCommand builds a bare command carrying just the --level flag.
func newLevelFlagCommand(level string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("level", "", "")
	if level != "" {
		_ = cmd.Flags().Set("level", level)
	}
	return cmd
}
