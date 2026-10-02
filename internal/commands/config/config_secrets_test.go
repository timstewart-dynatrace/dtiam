package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zalando/go-keyring"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
)

// isolate points the config directory at a temp dir and the keyring at an
// in-memory mock, and fails the test before anything runs if the config path
// is not inside the temp dir. These tests write config files and secrets; they
// must never reach the real ones.
func isolate(t *testing.T, withKeyring bool) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	xdg.Reload()
	t.Cleanup(xdg.Reload)

	path, err := config.GetConfigPath()
	if err != nil || !strings.HasPrefix(path, home) {
		t.Fatalf("config path %q is not isolated under %q", path, home)
	}

	keyring.MockInit()
	if withKeyring {
		t.Setenv(config.EnvDisableKeyring, "")
	} else {
		t.Setenv(config.EnvDisableKeyring, "1")
	}
	for _, v := range []string{config.EnvEnvironmentTkn, config.EnvClientID, config.EnvClientSecret} {
		t.Setenv(v, "")
	}
	return path
}

// run executes a command with fresh flag state; cobra keeps flag values and
// Changed bits on the package-level command between invocations.
func run(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	if err := cmd.ParseFlags(args); err != nil {
		return err
	}
	return cmd.RunE(cmd, cmd.Flags().Args())
}

func loadCred(t *testing.T, name string) *config.Credential {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cred := cfg.GetCredential(name)
	if cred == nil {
		t.Fatalf("credential %q not found", name)
	}
	return cred
}

func TestSetCredentials_NewCredentialNeedsClientIDAndSecret(t *testing.T) {
	isolate(t, false)
	err := run(t, setCredentialsCmd, "prod", "--environment-url", "abc12345")
	if err == nil || !strings.Contains(err.Error(), "do not exist yet") {
		t.Fatalf("error = %v, want a request for --client-id and --client-secret", err)
	}
}

func TestSetCredentials_UpdatesOptionalFieldsAlone(t *testing.T) {
	isolate(t, true)
	if err := run(t, setCredentialsCmd, "prod", "--client-id", "dt0s01.ID", "--client-secret", "dt0s01.ID.SECRET"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := run(t, setCredentialsCmd, "prod",
		"--api-url", "https://api.example.com",
		"--environment-url", "abc12345",
		"--environment-token", "dt0s16.TOKEN"); err != nil {
		t.Fatalf("update: %v", err)
	}

	cred := loadCred(t, "prod")
	if cred.ClientID != "dt0s01.ID" {
		t.Errorf("client-id = %q; an update without --client-id must keep it", cred.ClientID)
	}
	if cred.APIURL != "https://api.example.com" || cred.EnvironmentURL != "abc12345" {
		t.Errorf("api-url = %q, environment-url = %q", cred.APIURL, cred.EnvironmentURL)
	}
	if !config.IsKeyringReference(cred.EnvironmentToken) {
		t.Fatalf("environment-token in file = %q, want the keyring reference", cred.EnvironmentToken)
	}
	if got, _ := keyring.Get(config.KeyringService, config.EnvironmentTokenKeyringKey("prod")); got != "dt0s16.TOKEN" {
		t.Errorf("keyring holds %q, want the token", got)
	}

	resolved, err := config.ResolveEnvironmentToken(&config.Config{
		CurrentContext: "c",
		Contexts:       []config.NamedContext{{Name: "c", Context: config.Context{CredentialsRef: "prod"}}},
		Credentials:    []config.NamedCredential{{Name: "prod", Credential: *cred}},
	})
	if err != nil || resolved != "dt0s16.TOKEN" {
		t.Errorf("ResolveEnvironmentToken() = %q, %v; want the token from the keyring", resolved, err)
	}
}

func TestSetCredentials_ClearsEnvironmentToken(t *testing.T) {
	isolate(t, true)
	_ = run(t, setCredentialsCmd, "prod", "--client-id", "a", "--client-secret", "b", "--environment-token", "tok")
	if err := run(t, setCredentialsCmd, "prod", "--environment-token", ""); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if cred := loadCred(t, "prod"); cred.EnvironmentToken != "" {
		t.Errorf("environment-token = %q, want cleared", cred.EnvironmentToken)
	}
	if _, err := keyring.Get(config.KeyringService, config.EnvironmentTokenKeyringKey("prod")); err == nil {
		t.Error("the keyring entry for the cleared token was left behind")
	}
}

func TestSetCredentials_NoKeyringStoresPlaintext(t *testing.T) {
	isolate(t, true)
	if err := run(t, setCredentialsCmd, "ci", "--client-id", "a", "--client-secret", "b",
		"--environment-token", "tok", "--no-keyring"); err != nil {
		t.Fatalf("set: %v", err)
	}
	cred := loadCred(t, "ci")
	if cred.ClientSecret != "b" || cred.EnvironmentToken != "tok" {
		t.Errorf("secret = %q, token = %q; --no-keyring must store both in the file", cred.ClientSecret, cred.EnvironmentToken)
	}
}

// TestMigrateSecrets_RemovesPlaintextFromFile is the regression test for
// migrate-secrets copying secrets into the keyring while leaving the plaintext
// in the config file, because SetCredentialField ignored "client-secret".
func TestMigrateSecrets_RemovesPlaintextFromFile(t *testing.T) {
	path := isolate(t, true)
	if err := run(t, setCredentialsCmd, "prod", "--client-id", "a", "--client-secret", "PLAIN-SECRET",
		"--environment-token", "PLAIN-TOKEN", "--no-keyring"); err != nil {
		t.Fatalf("set: %v", err)
	}

	if err := run(t, migrateSecretsCmd); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, secret := range []string{"PLAIN-SECRET", "PLAIN-TOKEN"} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("config file still contains %q after migrate-secrets", secret)
		}
	}
	if got, _ := keyring.Get(config.KeyringService, "prod"); got != "PLAIN-SECRET" {
		t.Errorf("keyring client secret = %q", got)
	}
	if got, _ := keyring.Get(config.KeyringService, config.EnvironmentTokenKeyringKey("prod")); got != "PLAIN-TOKEN" {
		t.Errorf("keyring environment token = %q", got)
	}
}
