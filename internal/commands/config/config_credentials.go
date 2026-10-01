package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/config"
	"github.com/jtimothystewart/dtiam/internal/output"
)

var setCredentialsCmd = &cobra.Command{
	Use:   "set-credentials NAME",
	Short: "Create or update a credential set",
	Long: `Create a new credential set or update an existing one.

Both --client-id and --client-secret are required. The credential name is used
by contexts via --credentials-ref.

The client secret is stored in the OS keyring when one is available, and the
config file records only a reference to it. Where the secret ended up is always
reported, so you can reason about your own exposure.

When no keyring is available -- headless Linux, containers, CI -- the secret
falls back to the config file in plaintext and a warning says so. Use
--no-keyring to force file storage, or --require-keyring to fail rather than
write a plaintext secret.`,
	Example: `  # Create credentials for production (secret goes to the keyring)
  dtiam config set-credentials prod --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY

  # Refuse to store the secret in plaintext if no keyring exists
  dtiam config set-credentials prod --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY --require-keyring

  # Force plaintext file storage
  dtiam config set-credentials ci --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY --no-keyring`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		clientID, _ := cmd.Flags().GetString("client-id")
		clientSecret, _ := cmd.Flags().GetString("client-secret")
		noKeyring, _ := cmd.Flags().GetBool("no-keyring")
		requireKeyring, _ := cmd.Flags().GetBool("require-keyring")

		if clientID == "" || clientSecret == "" {
			return fmt.Errorf("both --client-id and --client-secret are required")
		}
		if noKeyring && requireKeyring {
			return fmt.Errorf("--no-keyring and --require-keyring are mutually exclusive")
		}

		printer := cli.GlobalState.NewPrinter()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// storedSecret is what lands in the config file: either the keyring
		// marker or, as a fallback, the secret itself.
		storedSecret := clientSecret
		usedKeyring := false

		if !noKeyring {
			store := &config.SecretStore{AllowFileFallback: !requireKeyring}
			usedKeyring, err = store.SetSecret(name, clientSecret)
			if err != nil {
				return fmt.Errorf("failed to store client secret: %w", err)
			}
			if usedKeyring {
				storedSecret = config.KeyringMarker()
			}
		}

		cfg.SetCredential(name, clientID, storedSecret)

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		printer.PrintSuccess("Credentials %q updated", name)
		if usedKeyring {
			printer.PrintMessage("Client secret stored in the OS keyring (service %q).", config.KeyringService)
		} else {
			path, _ := config.GetConfigPath()
			printer.PrintWarning(
				"No OS keyring available: the client secret is stored in plaintext at %s. "+
					"Keep that file at mode 0600.", path)
		}
		return nil
	},
}

func init() {
	setCredentialsCmd.Flags().String("client-id", "", "OAuth2 client ID")
	setCredentialsCmd.Flags().String("client-secret", "", "OAuth2 client secret")
	setCredentialsCmd.Flags().Bool("no-keyring", false,
		"Store the secret in the config file instead of the OS keyring")
	setCredentialsCmd.Flags().Bool("require-keyring", false,
		"Fail rather than fall back to storing the secret in plaintext")
}

var deleteCredentialsCmd = &cobra.Command{
	Use:   "delete-credentials NAME",
	Short: "Delete a credential set",
	Long: `Remove a credential set from the configuration file.

Any client secret held in the OS keyring for this credential is removed too,
so deleting a credential does not leave an orphaned secret behind.

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

		// Remove the keyring entry after the config is saved. If this fails the
		// credential is already gone from the config, so report it rather than
		// returning an error that would imply nothing happened.
		if err := config.NewSecretStore().DeleteSecret(name); err != nil {
			printer.PrintWarning("Credentials deleted, but the keyring secret could not be removed: %v", err)
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
