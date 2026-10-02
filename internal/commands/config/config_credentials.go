package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
)

var setCredentialsCmd = &cobra.Command{
	Use:   "set-credentials NAME",
	Short: "Create or update a credential set",
	Long: `Create a new credential set or update an existing one.

A new credential needs both --client-id and --client-secret. An existing one can
be updated with any subset of flags; fields you do not pass keep their values.
The credential name is used by contexts via --credentials-ref.

Optional per-credential settings:
  --api-url             alternative host for all Account Management API calls
  --environment-url     default environment for get apps/schemas/env-users/env-groups
  --environment-token   token for those environment commands

Pass a flag with an empty value (--environment-token "") to clear it.

Secrets -- the client secret and the environment token -- are stored in the OS
keyring when one is available, and the config file records only a reference.
Where each secret ended up is always reported. When no keyring is available --
headless Linux, containers, CI -- secrets fall back to the config file in
plaintext and a warning says so. Use --no-keyring to force file storage, or
--require-keyring to fail rather than write a plaintext secret.`,
	Example: `  # Create credentials for production (secret goes to the keyring)
  dtiam config set-credentials prod --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY

  # Add an environment URL and token to existing credentials
  dtiam config set-credentials prod --environment-url abc12345 --environment-token dt0s16.XXX

  # Point a credential at a different API host
  dtiam config set-credentials dev --api-url https://api.example.com

  # Refuse to store secrets in plaintext if no keyring exists
  dtiam config set-credentials prod --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY --require-keyring

  # Force plaintext file storage
  dtiam config set-credentials ci --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY --no-keyring`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		flags := cmd.Flags()

		clientID, _ := flags.GetString("client-id")
		clientSecret, _ := flags.GetString("client-secret")
		noKeyring, _ := flags.GetBool("no-keyring")
		requireKeyring, _ := flags.GetBool("require-keyring")

		if noKeyring && requireKeyring {
			return fmt.Errorf("--no-keyring and --require-keyring are mutually exclusive")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		exists := cfg.GetCredential(name) != nil

		settingClient := flags.Changed("client-id") || flags.Changed("client-secret")
		if !exists || settingClient {
			if clientID == "" || clientSecret == "" {
				if !exists {
					return fmt.Errorf("credentials %q do not exist yet: both --client-id and --client-secret are required", name)
				}
				return fmt.Errorf("--client-id and --client-secret must be given together")
			}
		}

		optional := []string{"api-url", "environment-url", "environment-token"}
		changedOptional := false
		for _, f := range optional {
			changedOptional = changedOptional || flags.Changed(f)
		}
		if exists && !settingClient && !changedOptional {
			return fmt.Errorf("nothing to update: pass --client-id/--client-secret or an optional setting")
		}

		printer := cli.GlobalState.NewPrinter()
		store := &config.SecretStore{AllowFileFallback: !requireKeyring}

		// storeSecret puts a secret in the keyring when allowed and possible and
		// returns what the config file should hold: the keyring marker, or the
		// secret itself as a fallback.
		storeSecret := func(key, secret string) (fileValue string, usedKeyring bool, err error) {
			if noKeyring {
				return secret, false, nil
			}
			usedKeyring, err = store.SetSecret(key, secret)
			if err != nil {
				return "", false, err
			}
			if usedKeyring {
				return config.KeyringMarker(), true, nil
			}
			return secret, false, nil
		}

		var report []string
		var plaintext []string

		if settingClient || !exists {
			fileValue, usedKeyring, err := storeSecret(name, clientSecret)
			if err != nil {
				return fmt.Errorf("failed to store client secret: %w", err)
			}
			cfg.SetCredential(name, clientID, fileValue)
			if usedKeyring {
				report = append(report, "client secret")
			} else {
				plaintext = append(plaintext, "client secret")
			}
		}

		for _, f := range []string{"api-url", "environment-url"} {
			if flags.Changed(f) {
				v, _ := flags.GetString(f)
				cfg.SetCredentialField(name, f, v)
			}
		}

		if flags.Changed("environment-token") {
			token, _ := flags.GetString("environment-token")
			key := config.EnvironmentTokenKeyringKey(name)
			if token == "" {
				cfg.SetCredentialField(name, "environment-token", "")
				if err := config.NewSecretStore().DeleteSecret(key); err != nil {
					printer.PrintWarning("Environment token cleared, but its keyring entry could not be removed: %v", err)
				}
			} else {
				fileValue, usedKeyring, err := storeSecret(key, token)
				if err != nil {
					return fmt.Errorf("failed to store environment token: %w", err)
				}
				cfg.SetCredentialField(name, "environment-token", fileValue)
				if usedKeyring {
					report = append(report, "environment token")
				} else {
					plaintext = append(plaintext, "environment token")
				}
			}
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		printer.PrintSuccess("Credentials %q updated", name)
		if len(report) > 0 {
			printer.PrintMessage("Stored in the OS keyring (service %q): %s.", config.KeyringService, strings.Join(report, ", "))
		}
		if len(plaintext) > 0 {
			path, _ := config.GetConfigPath()
			reason := "No OS keyring available"
			if noKeyring {
				reason = "--no-keyring was set"
			}
			printer.PrintWarning("%s: %s stored in plaintext at %s. Keep that file at mode 0600.",
				reason, strings.Join(plaintext, " and "), path)
		}
		return nil
	},
}

func init() {
	setCredentialsCmd.Flags().String("client-id", "", "OAuth2 client ID")
	setCredentialsCmd.Flags().String("client-secret", "", "OAuth2 client secret")
	setCredentialsCmd.Flags().String("api-url", "", "Alternative Account Management API host, e.g. https://api.example.com")
	setCredentialsCmd.Flags().String("environment-url", "", "Default environment ID or URL for environment commands")
	setCredentialsCmd.Flags().String("environment-token", "", "Token for environment commands (stored in the OS keyring)")
	setCredentialsCmd.Flags().Bool("no-keyring", false,
		"Store secrets in the config file instead of the OS keyring")
	setCredentialsCmd.Flags().Bool("require-keyring", false,
		"Fail rather than fall back to storing secrets in plaintext")
}

var deleteCredentialsCmd = &cobra.Command{
	Use:   "delete-credentials NAME",
	Short: "Delete a credential set",
	Long: `Remove a credential set from the configuration file.

Any client secret or environment token held in the OS keyring for this
credential is removed too, so deleting a credential does not leave an orphaned
secret behind.

Any contexts referencing these credentials will become invalid until updated
with a new credentials-ref.`,
	Example: `  # Delete the staging credentials
  dtiam config delete-credentials staging

  # Preview the deletion
  dtiam config delete-credentials staging --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Dry run: would delete credentials %q and any keyring secret for it", name)
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !cfg.DeleteCredential(name) {
			return fmt.Errorf("credentials %q not found", name)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		// Remove the keyring entries after the config is saved. If this fails the
		// credential is already gone from the config, so report it rather than
		// returning an error that would imply nothing happened.
		for _, key := range []string{name, config.EnvironmentTokenKeyringKey(name)} {
			if err := config.NewSecretStore().DeleteSecret(key); err != nil {
				printer.PrintWarning("Credentials deleted, but a keyring secret could not be removed: %v", err)
			}
		}

		printer.PrintSuccess("Credentials %q deleted", name)
		return nil
	},
}

var getCredentialsCmd = &cobra.Command{
	Use:     "get-credentials",
	Aliases: []string{"credentials"},
	Short:   "List all configured credential sets",
	Long: `List all credential sets defined in the configuration file.

Only the credential name and client ID are shown. Use config view --show-secrets
to see full secrets.`,
	Example: `  # List all credentials
  dtiam config get-credentials

  # Output as JSON
  dtiam config get-credentials -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		printer := cli.GlobalState.NewPrinter()

		data := make([]map[string]any, len(cfg.Credentials))
		for i, cred := range cfg.Credentials {
			data[i] = map[string]any{
				"name":      cred.Name,
				"client_id": cred.Credential.ClientID,
			}
		}

		return printer.Print(data, output.CredentialColumns())
	},
}
