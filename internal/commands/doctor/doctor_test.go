package doctor

import (
	"strings"
	"testing"

	"github.com/jtimothystewart/dtiam/internal/auth"
	"github.com/jtimothystewart/dtiam/internal/config"
)

func TestVersionCheckAlwaysOK(t *testing.T) {
	r := versionCheck()
	if r.Status != StatusOK {
		t.Errorf("Status = %q, want ok", r.Status)
	}
	if r.Detail == "" {
		t.Error("Detail should carry the version")
	}
}

func TestContextCheck(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *config.Config
		want   string
		detail string
	}{
		{
			name: "should warn when no context is selected",
			cfg:  &config.Config{},
			want: StatusWarn,
		},
		{
			name: "should fail when the selected context is not defined",
			cfg:  &config.Config{CurrentContext: "missing"},
			want: StatusFail,
		},
		{
			name: "should pass when the context exists",
			cfg: &config.Config{
				CurrentContext: "prod",
				Contexts: []config.NamedContext{
					{Name: "prod", Context: config.Context{AccountUUID: "u"}},
				},
			},
			want: StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contextCheck(tt.cfg); got.Status != tt.want {
				t.Errorf("Status = %q, want %q (detail: %s)", got.Status, tt.want, got.Detail)
			}
		})
	}
}

func TestAccountCheck(t *testing.T) {
	if got := accountCheck(""); got.Status != StatusFail {
		t.Errorf("empty UUID: Status = %q, want fail", got.Status)
	}
	if got := accountCheck("abc-123"); got.Status != StatusOK {
		t.Errorf("valid UUID: Status = %q, want ok", got.Status)
	}
}

func TestCredentialsCheck(t *testing.T) {
	tests := []struct {
		name                           string
		clientID, clientSecret, bearer string
		useOAuth                       bool
		want                           string
	}{
		{
			name:     "should pass with a complete OAuth client",
			clientID: "dt0s02.ABC", clientSecret: "dt0s02.ABC.XYZ", useOAuth: true, want: StatusOK,
		},
		{
			name:     "should fail when the OAuth secret is missing",
			clientID: "dt0s02.ABC", useOAuth: true, want: StatusFail,
		},
		{
			name:         "should fail when no client ID could be derived",
			clientSecret: "secret", useOAuth: true, want: StatusFail,
		},
		{
			// A static token cannot refresh, which is worth flagging even though
			// it works right now.
			name:   "should warn for a static bearer token",
			bearer: "token", useOAuth: false, want: StatusWarn,
		},
		{
			name:     "should fail when nothing is configured",
			useOAuth: false, want: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := credentialsCheck(tt.clientID, tt.clientSecret, tt.bearer, tt.useOAuth)
			if got.Status != tt.want {
				t.Errorf("Status = %q, want %q (detail: %s)", got.Status, tt.want, got.Detail)
			}
		})
	}
}

func TestScopesCheck_SkipsForBearerToken(t *testing.T) {
	if got := scopesCheck(&config.Config{}, false); got.Status != StatusSkip {
		t.Errorf("Status = %q, want skip for bearer auth", got.Status)
	}
}

func TestScopesCheck_DefaultSetIsOK(t *testing.T) {
	t.Setenv(config.EnvScopes, "")
	got := scopesCheck(&config.Config{}, true)
	if got.Status != StatusOK {
		t.Errorf("Status = %q, want ok", got.Status)
	}
	if !strings.Contains(got.Detail, "default set") {
		t.Errorf("Detail = %q, want it to mention the default set", got.Detail)
	}
}

func TestScopesCheck_WarnsAboutOmittedScopes(t *testing.T) {
	// An override silently drops what it omits, which is the most common cause
	// of an unexpected 403 -- so the missing scopes must be named.
	t.Setenv(config.EnvScopes, "account-idm-read")
	got := scopesCheck(&config.Config{}, true)
	if got.Status != StatusWarn {
		t.Fatalf("Status = %q, want warn", got.Status)
	}
	if !strings.Contains(got.Detail, "account-uac-read") {
		t.Errorf("Detail = %q, want it to name the omitted scopes", got.Detail)
	}
}

func TestScopesCheck_FullOverrideIsOK(t *testing.T) {
	t.Setenv(config.EnvScopes, strings.Join(auth.DefaultScopeList, " "))
	if got := scopesCheck(&config.Config{}, true); got.Status != StatusOK {
		t.Errorf("Status = %q, want ok when the override covers every default", got.Status)
	}
}

func TestSkipChecksCoversBothNetworkChecks(t *testing.T) {
	got := skipChecks(true, "--offline")
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2", len(got))
	}
	for _, r := range got {
		if r.Status != StatusSkip {
			t.Errorf("%s: Status = %q, want skip", r.Name, r.Status)
		}
	}
}

func TestResultsToMapsPreservesEveryField(t *testing.T) {
	in := []CheckResult{{Name: "a", Status: StatusOK, Detail: "d"}}
	got := resultsToMaps(in)
	if len(got) != 1 {
		t.Fatalf("got %d maps, want 1", len(got))
	}
	if got[0]["check"] != "a" || got[0]["status"] != StatusOK || got[0]["detail"] != "d" {
		t.Errorf("got %v", got[0])
	}
}

func TestCmdMetadata(t *testing.T) {
	if Cmd.Short == "" || Cmd.Long == "" || Cmd.Example == "" {
		t.Error("Short, Long and Example are all required")
	}
	if Cmd.RunE == nil {
		t.Error("must use RunE")
	}
	if Cmd.Flags().Lookup("offline") == nil {
		t.Error("--offline flag is missing")
	}
}

func TestDoctorColumns(t *testing.T) {
	cols := DoctorColumns()
	if len(cols) != 3 {
		t.Fatalf("got %d columns, want 3", len(cols))
	}
	want := []string{"check", "status", "detail"}
	for i, w := range want {
		if cols[i].Key != w {
			t.Errorf("column %d key = %q, want %q", i, cols[i].Key, w)
		}
	}
}

func TestSecretStorageCheck(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{
			name: "should skip when no credentials are stored",
			cfg:  &config.Config{},
			want: StatusSkip,
		},
		{
			name: "should pass when every secret is in the keyring",
			cfg: &config.Config{Credentials: []config.NamedCredential{
				{Name: "a", Credential: config.Credential{ClientSecret: config.KeyringMarker()}},
			}},
			want: StatusOK,
		},
		{
			// Plaintext works, so this is a warning -- but it must be surfaced
			// every run, because a secret in a dotfile is easy to forget.
			name: "should warn about plaintext secrets",
			cfg: &config.Config{Credentials: []config.NamedCredential{
				{Name: "a", Credential: config.Credential{ClientSecret: "dt0s01.ABC.XYZ"}},
			}},
			want: StatusWarn,
		},
		{
			name: "should skip when a credential has no secret at all",
			cfg: &config.Config{Credentials: []config.NamedCredential{
				{Name: "a", Credential: config.Credential{ClientSecret: ""}},
			}},
			want: StatusSkip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := secretStorageCheck(tt.cfg)
			if got.Status != tt.want {
				t.Errorf("Status = %q, want %q (detail: %s)", got.Status, tt.want, got.Detail)
			}
		})
	}
}

func TestSecretStorageCheck_PointsAtTheFixWhenKeyringIsAvailable(t *testing.T) {
	// When a keyring exists, the warning should name the command that fixes it.
	t.Setenv(config.EnvDisableKeyring, "")
	cfg := &config.Config{Credentials: []config.NamedCredential{
		{Name: "a", Credential: config.Credential{ClientSecret: "dt0s01.ABC.XYZ"}},
	}}

	got := secretStorageCheck(cfg)
	if got.Status != StatusWarn {
		t.Fatalf("Status = %q, want warn", got.Status)
	}
	if config.KeyringAvailable() && !strings.Contains(got.Detail, "migrate-secrets") {
		t.Errorf("Detail = %q, want it to name 'migrate-secrets'", got.Detail)
	}
}
