package config

import (
	"strings"
	"testing"
)

func TestIsKeyringReference(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "should recognize the marker", value: KeyringMarker(), want: true},
		{name: "should reject a real secret", value: "dt0s01.ABC.XYZ", want: false},
		{name: "should reject empty", value: "", want: false},
		{name: "should reject a similar-looking value", value: "keyring", want: false},
		{name: "should reject a different service marker", value: "keyring:dtctl", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKeyringReference(tt.value); got != tt.want {
				t.Errorf("IsKeyringReference(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestKeyringMarkerIsNotAValidSecret(t *testing.T) {
	// The marker lands in the config file where a secret would be. It must not
	// look like a usable Dynatrace secret, so a tool that ignores the marker
	// fails loudly rather than authenticating with nonsense.
	m := KeyringMarker()
	if strings.HasPrefix(m, "dt0s") {
		t.Errorf("marker %q looks like a real Dynatrace secret", m)
	}
	if m == "" {
		t.Error("marker must not be empty, or it would read as 'no secret set'")
	}
}

func TestKeyringAvailable_RespectsDisableEnvVar(t *testing.T) {
	t.Setenv(EnvDisableKeyring, "1")
	if KeyringAvailable() {
		t.Error("KeyringAvailable() = true despite DTIAM_DISABLE_KEYRING being set")
	}
}

func TestSecretStore_GetSecretPassesThroughPlaintext(t *testing.T) {
	// Existing plaintext configs must keep working with no migration step.
	s := NewSecretStore()
	got, err := s.GetSecret("prod", "dt0s01.ABC.XYZ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "dt0s01.ABC.XYZ" {
		t.Errorf("GetSecret() = %q, want the stored value unchanged", got)
	}
}

func TestSecretStore_GetSecretPassesThroughEmpty(t *testing.T) {
	s := NewSecretStore()
	got, err := s.GetSecret("prod", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("GetSecret() = %q, want empty", got)
	}
}

func TestSecretStore_SetSecretRejectsEmpty(t *testing.T) {
	s := NewSecretStore()
	if _, err := s.SetSecret("prod", ""); err == nil {
		t.Fatal("expected an error for an empty secret")
	}
}

func TestSecretStore_SetSecretFallsBackToFileWhenKeyringDisabled(t *testing.T) {
	t.Setenv(EnvDisableKeyring, "1")
	s := NewSecretStore()

	usedKeyring, err := s.SetSecret("prod", "dt0s01.ABC.XYZ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usedKeyring {
		t.Error("usedKeyring = true with the keyring disabled")
	}
}

func TestSecretStore_SetSecretFailsWhenKeyringRequiredButUnavailable(t *testing.T) {
	// --require-keyring must refuse rather than quietly writing plaintext.
	t.Setenv(EnvDisableKeyring, "1")
	s := &SecretStore{AllowFileFallback: false}

	if _, err := s.SetSecret("prod", "dt0s01.ABC.XYZ"); err == nil {
		t.Fatal("expected an error when the keyring is required but unavailable")
	}
}

func TestSecretStore_MigrateIsANoOpForAlreadyMigrated(t *testing.T) {
	s := NewSecretStore()
	value, migrated, err := s.MigrateSecretToKeyring("prod", KeyringMarker())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if migrated {
		t.Error("migrated = true for a value already in the keyring")
	}
	if value != KeyringMarker() {
		t.Errorf("value = %q, want the marker unchanged", value)
	}
}

func TestSecretStore_MigrateIsANoOpForEmptySecret(t *testing.T) {
	s := NewSecretStore()
	_, migrated, err := s.MigrateSecretToKeyring("prod", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if migrated {
		t.Error("migrated = true for an empty secret")
	}
}

func TestSecretStore_MigrateLeavesSecretIntactWhenKeyringUnavailable(t *testing.T) {
	// A failed migration must leave a working config, not a broken one.
	t.Setenv(EnvDisableKeyring, "1")
	s := NewSecretStore()

	value, migrated, err := s.MigrateSecretToKeyring("prod", "dt0s01.ABC.XYZ")
	if err == nil {
		t.Fatal("expected an error when no keyring is available")
	}
	if migrated {
		t.Error("migrated = true despite the error")
	}
	if value != "dt0s01.ABC.XYZ" {
		t.Errorf("value = %q, want the original secret preserved", value)
	}
}

func TestSecretStore_DeleteSecretToleratesNoKeyring(t *testing.T) {
	// Deleting a credential whose secret lives in the config file must succeed.
	t.Setenv(EnvDisableKeyring, "1")
	if err := NewSecretStore().DeleteSecret("prod"); err != nil {
		t.Errorf("DeleteSecret() error = %v, want nil when there is no keyring", err)
	}
}

func TestCurrentCredentialNameResolvesThroughContext(t *testing.T) {
	ref := "prod-creds"
	cfg := &Config{
		CurrentContext: "prod",
		Contexts: []NamedContext{
			{Name: "prod", Context: Context{AccountUUID: "u", CredentialsRef: ref}},
		},
	}
	if got := cfg.CurrentCredentialName(); got != ref {
		t.Errorf("CurrentCredentialName() = %q, want %q", got, ref)
	}
}

func TestCurrentCredentialNameEmptyWithoutContext(t *testing.T) {
	if got := (&Config{}).CurrentCredentialName(); got != "" {
		t.Errorf("CurrentCredentialName() = %q, want empty", got)
	}
}
