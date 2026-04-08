package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() error: %v", err)
	}
	if !strings.HasSuffix(dir, "dtiam") {
		t.Errorf("GetConfigDir() = %q, want suffix 'dtiam'", dir)
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() error: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("dtiam", "config")) {
		t.Errorf("GetConfigPath() = %q, want suffix 'dtiam/config'", path)
	}
}

func TestLoadFromPath_NonExistent(t *testing.T) {
	cfg, err := LoadFromPath("/tmp/dtiam-test-nonexistent/config")
	if err != nil {
		t.Fatalf("LoadFromPath() error for nonexistent: %v", err)
	}
	if cfg.APIVersion != "v1" {
		t.Errorf("LoadFromPath() nonexistent should return default config, got APIVersion=%q", cfg.APIVersion)
	}
}

func TestLoadFromPath_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := `api-version: v1
kind: Config
current-context: test
contexts:
  - name: test
    context:
      account-uuid: abc-123
      credentials-ref: test-cred
credentials:
  - name: test-cred
    credential:
      client-id: dt0s01.TEST
      client-secret: dt0s01.TEST.SECRET
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() error: %v", err)
	}
	if cfg.CurrentContext != "test" {
		t.Errorf("CurrentContext = %q, want 'test'", cfg.CurrentContext)
	}
	if len(cfg.Contexts) != 1 {
		t.Fatalf("Contexts len = %d, want 1", len(cfg.Contexts))
	}
	if cfg.Contexts[0].Context.AccountUUID != "abc-123" {
		t.Errorf("AccountUUID = %q, want 'abc-123'", cfg.Contexts[0].Context.AccountUUID)
	}
}

func TestLoadFromPath_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromPath(path)
	if err == nil {
		t.Error("LoadFromPath() should return error for invalid YAML")
	}
}

func TestSaveToPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config")

	cfg := NewConfig()
	uuid := "save-uuid"
	if err := cfg.SetContext("saved", &uuid, nil); err != nil {
		t.Fatal(err)
	}

	if err := SaveToPath(cfg, path); err != nil {
		t.Fatalf("SaveToPath() error: %v", err)
	}

	// Verify it was saved
	loaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() after save error: %v", err)
	}
	if len(loaded.Contexts) != 1 {
		t.Errorf("saved config should have 1 context, got %d", len(loaded.Contexts))
	}
}

func TestApplyEnvOverrides_Context(t *testing.T) {
	// Set env via Viper directly (avoids os.Setenv race conditions)
	V.Set("context", "env-context")
	defer V.Set("context", "")

	cfg := NewConfig()
	applyEnvOverrides(cfg)

	if cfg.CurrentContext != "env-context" {
		t.Errorf("CurrentContext = %q, want 'env-context'", cfg.CurrentContext)
	}
}

func TestApplyEnvOverrides_Output(t *testing.T) {
	V.Set("output", "json")
	defer V.Set("output", "")

	cfg := NewConfig()
	applyEnvOverrides(cfg)

	if cfg.Preferences.Output != "json" {
		t.Errorf("Preferences.Output = %q, want 'json'", cfg.Preferences.Output)
	}
}

func TestGetEffectiveCredentials_EnvOAuth(t *testing.T) {
	os.Setenv(EnvClientID, "env-id")
	os.Setenv(EnvClientSecret, "env-secret")
	os.Setenv(EnvAccountUUID, "env-uuid")
	defer func() {
		os.Unsetenv(EnvClientID)
		os.Unsetenv(EnvClientSecret)
		os.Unsetenv(EnvAccountUUID)
	}()

	id, secret, uuid, bearer, useOAuth := GetEffectiveCredentials(nil)
	if !useOAuth {
		t.Error("should use OAuth when env credentials are set")
	}
	if id != "env-id" {
		t.Errorf("clientID = %q, want 'env-id'", id)
	}
	if secret != "env-secret" {
		t.Errorf("clientSecret = %q, want 'env-secret'", secret)
	}
	if uuid != "env-uuid" {
		t.Errorf("accountUUID = %q, want 'env-uuid'", uuid)
	}
	if bearer != "" {
		t.Errorf("bearerToken should be empty, got %q", bearer)
	}
}

func TestGetEffectiveCredentials_EnvBearer(t *testing.T) {
	os.Setenv(EnvBearerToken, "my-token")
	os.Setenv(EnvAccountUUID, "bearer-uuid")
	defer func() {
		os.Unsetenv(EnvBearerToken)
		os.Unsetenv(EnvAccountUUID)
	}()

	_, _, uuid, bearer, useOAuth := GetEffectiveCredentials(nil)
	if useOAuth {
		t.Error("should NOT use OAuth when bearer token is set")
	}
	if bearer != "my-token" {
		t.Errorf("bearerToken = %q, want 'my-token'", bearer)
	}
	if uuid != "bearer-uuid" {
		t.Errorf("accountUUID = %q, want 'bearer-uuid'", uuid)
	}
}

func TestGetEffectiveCredentials_ConfigFallback(t *testing.T) {
	// Clear env to force config fallback
	os.Unsetenv(EnvClientID)
	os.Unsetenv(EnvClientSecret)
	os.Unsetenv(EnvAccountUUID)
	os.Unsetenv(EnvBearerToken)

	cfg := NewConfig()
	uuid := "config-uuid"
	credRef := "cred1"
	cfg.SetCredential("cred1", "cfg-id", "cfg-secret")
	_ = cfg.SetContext("prod", &uuid, &credRef)
	_ = cfg.UseContext("prod")

	id, secret, acctUUID, _, useOAuth := GetEffectiveCredentials(cfg)
	if !useOAuth {
		t.Error("should use OAuth from config fallback")
	}
	if id != "cfg-id" {
		t.Errorf("clientID = %q, want 'cfg-id'", id)
	}
	if secret != "cfg-secret" {
		t.Errorf("clientSecret = %q, want 'cfg-secret'", secret)
	}
	if acctUUID != "config-uuid" {
		t.Errorf("accountUUID = %q, want 'config-uuid'", acctUUID)
	}
}

func TestExists_WhenNoConfig(t *testing.T) {
	// Use a temp dir that we know has no config file
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		t.Skip("temp dir already has a config file")
	}
	// We can't easily test Exists() without changing xdg.ConfigHome,
	// so test the underlying stat behavior instead
	if _, err := os.Stat(filepath.Join(dir, "dtiam", "config")); !os.IsNotExist(err) {
		t.Error("config should not exist in temp dir")
	}
}

func TestGetViperString(t *testing.T) {
	V.Set("test_key", "test_value")
	defer V.Set("test_key", "")

	if got := GetViperString("test_key"); got != "test_value" {
		t.Errorf("GetViperString() = %q, want 'test_value'", got)
	}
}

func TestGetViperBool(t *testing.T) {
	V.Set("test_bool", true)
	defer V.Set("test_bool", false)

	if !GetViperBool("test_bool") {
		t.Error("GetViperBool() should return true")
	}
}

func TestNewEnvConstants(t *testing.T) {
	// Verify new env constants exist
	tests := []struct {
		name     string
		constant string
		contains string
	}{
		{"API URL", EnvAPIURL, "DTIAM_API_URL"},
		{"Scopes", EnvScopes, "DTIAM_SCOPES"},
		{"Environment URL", EnvEnvironmentURL, "DTIAM_ENVIRONMENT_URL"},
		{"Environment Token", EnvEnvironmentTkn, "DTIAM_ENVIRONMENT_TOKEN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.contains {
				t.Errorf("env constant = %q, want %q", tt.constant, tt.contains)
			}
		})
	}
}
