package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

const (
	// KeyringService is the service name under which dtiam stores secrets in the
	// OS keyring.
	KeyringService = "dtiam"

	// EnvDisableKeyring disables keyring use when set to any non-empty value.
	// Useful in CI, containers, and headless sessions where no keyring daemon is
	// running and the probe would otherwise cost a timeout on every invocation.
	EnvDisableKeyring = "DTIAM_DISABLE_KEYRING"

	// keyringSecretMarker is written to the config file's client-secret field in
	// place of the secret once the secret itself lives in the keyring. It is a
	// deliberately invalid secret, so a tool that reads the file and ignores the
	// marker fails loudly rather than authenticating with nonsense.
	keyringSecretMarker = "keyring:dtiam"
)

// ErrKeyringUnavailable reports that no usable OS keyring was found.
var ErrKeyringUnavailable = errors.New("no OS keyring available")

// SecretStore stores OAuth client secrets, preferring the OS keyring and
// falling back to the config file.
//
// The fallback is deliberate: dtiam must keep working on headless Linux hosts
// and in CI where no keyring daemon exists. Callers can see which backend was
// used via UsesKeyring, so the choice is never silent.
type SecretStore struct {
	// AllowFileFallback permits writing the secret to the config file when the
	// keyring is unavailable. When false, a failed keyring write is an error.
	AllowFileFallback bool
}

// NewSecretStore returns a store that falls back to file storage.
func NewSecretStore() *SecretStore {
	return &SecretStore{AllowFileFallback: true}
}

// KeyringAvailable reports whether the OS keyring can be used.
//
// Availability is probed with a real lookup rather than assumed from the OS,
// because a keyring may be absent on a headless Linux box and present on the
// same distribution with a desktop session.
func KeyringAvailable() bool {
	if os.Getenv(EnvDisableKeyring) != "" {
		return false
	}

	// A missing entry is a successful probe: it proves the backend answered.
	_, err := keyring.Get(KeyringService, "__dtiam_probe__")
	if err == nil || errors.Is(err, keyring.ErrNotFound) {
		return true
	}
	return false
}

// EnvironmentTokenKeyringKey is the keyring entry name for a credential's
// environment token. Client secrets are stored under the bare credential name,
// so the token needs its own key to sit alongside it.
func EnvironmentTokenKeyringKey(credentialName string) string {
	return credentialName + "/environment-token"
}

// IsKeyringReference reports whether a stored client-secret value is a marker
// pointing at the keyring rather than the secret itself.
func IsKeyringReference(value string) bool {
	return value == keyringSecretMarker
}

// KeyringMarker returns the value written to the config file when the real
// secret is held in the keyring.
func KeyringMarker() string {
	return keyringSecretMarker
}

// SetSecret stores a credential's client secret.
//
// Returns usedKeyring so the caller can tell the user where the secret went;
// hiding that would leave them unable to reason about their own exposure.
func (s *SecretStore) SetSecret(credentialName, secret string) (usedKeyring bool, err error) {
	if secret == "" {
		return false, fmt.Errorf("client secret is empty")
	}

	if KeyringAvailable() {
		if err := keyring.Set(KeyringService, credentialName, secret); err == nil {
			return true, nil
		} else if !s.AllowFileFallback {
			return false, fmt.Errorf("failed to store secret in keyring: %w", err)
		}
	} else if !s.AllowFileFallback {
		return false, ErrKeyringUnavailable
	}

	return false, nil
}

// GetSecret retrieves a credential's client secret.
//
// stored is the value read from the config file. When it is the keyring marker
// the secret is fetched from the keyring; otherwise stored is returned as-is, so
// existing plaintext configs keep working without migration.
func (s *SecretStore) GetSecret(credentialName, stored string) (string, error) {
	if !IsKeyringReference(stored) {
		return stored, nil
	}

	secret, err := keyring.Get(KeyringService, credentialName)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", fmt.Errorf(
				"credential %q refers to the keyring but no secret is stored there; "+
					"re-run 'dtiam config set-credentials %s'", credentialName, credentialName)
		}
		return "", fmt.Errorf("failed to read secret for %q from the keyring: %w", credentialName, err)
	}
	return secret, nil
}

// DeleteSecret removes a credential's secret from the keyring.
//
// A missing entry is not an error: deleting a credential that was stored in the
// config file should succeed rather than fail on a keyring that never held it.
func (s *SecretStore) DeleteSecret(credentialName string) error {
	if !KeyringAvailable() {
		return nil
	}

	err := keyring.Delete(KeyringService, credentialName)
	if err == nil || errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return fmt.Errorf("failed to delete secret for %q from the keyring: %w", credentialName, err)
}

// MigrateSecretToKeyring moves a plaintext secret from the config file into the
// keyring, returning the value that should replace it in the file.
//
// Returns the original secret unchanged when the keyring is unavailable, so a
// failed migration leaves a working configuration rather than a broken one.
func (s *SecretStore) MigrateSecretToKeyring(credentialName, plaintextSecret string) (newFileValue string, migrated bool, err error) {
	if plaintextSecret == "" || IsKeyringReference(plaintextSecret) {
		return plaintextSecret, false, nil
	}
	if !KeyringAvailable() {
		return plaintextSecret, false, ErrKeyringUnavailable
	}

	if err := keyring.Set(KeyringService, credentialName, plaintextSecret); err != nil {
		return plaintextSecret, false, fmt.Errorf("failed to store secret in keyring: %w", err)
	}
	return keyringSecretMarker, true, nil
}
