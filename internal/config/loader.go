package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	configDir  = "dtiam"
	configFile = "config"
)

// Environment variable names
const (
	EnvBearerToken    = "DTIAM_BEARER_TOKEN"
	EnvClientID       = "DTIAM_CLIENT_ID"
	EnvClientSecret   = "DTIAM_CLIENT_SECRET"
	EnvAccountUUID    = "DTIAM_ACCOUNT_UUID"
	EnvContext        = "DTIAM_CONTEXT"
	EnvOutput         = "DTIAM_OUTPUT"
	EnvVerbose        = "DTIAM_VERBOSE"
	EnvAPIURL         = "DTIAM_API_URL"
	EnvScopes         = "DTIAM_SCOPES"
	EnvEnvironmentURL = "DTIAM_ENVIRONMENT_URL"
	EnvEnvironmentTkn = "DTIAM_ENVIRONMENT_TOKEN"
)

// V is the package-level Viper instance for env/flag binding.
var V = viper.New()

func init() {
	V.SetEnvPrefix("DTIAM")
	V.AutomaticEnv()
	V.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind known env vars with defaults
	V.SetDefault("context", "")
	V.SetDefault("output", "")
	V.SetDefault("verbose", false)
	V.SetDefault("plain", false)
	V.SetDefault("dry_run", false)
	V.SetDefault("api_url", "")
	V.SetDefault("scopes", "")
	V.SetDefault("environment_url", "")
	V.SetDefault("environment_token", "")
}

// GetConfigDir returns the configuration directory path using XDG base directories.
func GetConfigDir() (string, error) {
	return filepath.Join(xdg.ConfigHome, configDir), nil
}

// GetConfigPath returns the configuration file path.
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFile), nil
}

// Load loads the configuration from the default path.
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFromPath(path)
}

// LoadFromPath loads configuration from a specific path.
func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return new config if file doesn't exist
			cfg := NewConfig()
			applyEnvOverrides(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides via Viper
	applyEnvOverrides(config)

	return config, nil
}

// Save saves the configuration to the default path.
func Save(config *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}
	return SaveToPath(config, path)
}

// SaveToPath saves configuration to a specific path.
func SaveToPath(config *Config, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides to the config via Viper.
func applyEnvOverrides(config *Config) {
	// Context override
	if ctx := V.GetString("context"); ctx != "" {
		config.CurrentContext = ctx
	}

	// Output preference override
	if output := V.GetString("output"); output != "" {
		config.Preferences.Output = output
	}
}

// GetEffectiveCredentials returns the effective credentials, considering env vars.
// Returns clientID, clientSecret, accountUUID, bearerToken, and whether OAuth is used.
func GetEffectiveCredentials(config *Config) (clientID, clientSecret, accountUUID, bearerToken string, useOAuth bool) {
	// Check for bearer token first
	bearerToken = os.Getenv(EnvBearerToken)

	// Check for OAuth credentials from env
	clientID = os.Getenv(EnvClientID)
	clientSecret = os.Getenv(EnvClientSecret)
	accountUUID = os.Getenv(EnvAccountUUID)

	// If env credentials are set, use them
	if clientID != "" && clientSecret != "" && accountUUID != "" {
		return clientID, clientSecret, accountUUID, "", true
	}

	// If bearer token is set with account UUID, use it
	if bearerToken != "" && accountUUID != "" {
		return "", "", accountUUID, bearerToken, false
	}

	// Fall back to config file credentials
	if config != nil {
		ctx := config.GetCurrentContext()
		if ctx != nil {
			if accountUUID == "" {
				accountUUID = ctx.AccountUUID
			}

			cred := config.GetCurrentCredential()
			if cred != nil && clientID == "" && clientSecret == "" {
				clientID = cred.ClientID
				clientSecret = cred.ClientSecret
				useOAuth = true
			}
		}
	}

	return clientID, clientSecret, accountUUID, bearerToken, useOAuth
}

// Exists checks if a config file exists at the default path.
func Exists() bool {
	path, err := GetConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// GetViperString returns a string from Viper (env or flag binding).
func GetViperString(key string) string {
	return V.GetString(key)
}

// GetViperBool returns a bool from Viper (env or flag binding).
func GetViperBool(key string) bool {
	return V.GetBool(key)
}
