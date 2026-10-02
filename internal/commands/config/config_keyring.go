package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
)

var migrateSecretsCmd = &cobra.Command{
	Use:   "migrate-secrets",
	Short: "Move plaintext secrets into the OS keyring",
	Long: `Move any plaintext client secrets and environment tokens in the config file
into the OS keyring, replacing each with a reference.

dtiam historically stored client secrets as plaintext in the config file, and
still does when no keyring is available. This moves existing secrets into the
keyring so the file no longer holds them.

Credentials that already reference the keyring are left alone, so the command is
safe to run repeatedly. If the keyring is unavailable, nothing is changed.`,
	Example: `  # Preview what would move
  dtiam config migrate-secrets --dry-run

  # Migrate
  dtiam config migrate-secrets`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !config.KeyringAvailable() {
			return fmt.Errorf(
				"no OS keyring available, so there is nowhere to migrate secrets to; " +
					"unset DTIAM_DISABLE_KEYRING if you set it")
		}

		// Collect first so dry-run and the real run report identically.
		// pending is one plaintext secret to move: the credential it belongs to,
		// the config field holding it, and its keyring key.
		type pending struct {
			name   string
			field  string
			key    string
			label  string
			secret string
		}
		var todo []pending
		alreadyMigrated := 0

		for _, named := range cfg.Credentials {
			secrets := []pending{
				{named.Name, "client-secret", named.Name, "client secret", named.Credential.ClientSecret},
				{named.Name, "environment-token", config.EnvironmentTokenKeyringKey(named.Name),
					"environment token", named.Credential.EnvironmentToken},
			}
			for _, p := range secrets {
				switch {
				case p.secret == "":
					continue
				case config.IsKeyringReference(p.secret):
					alreadyMigrated++
				default:
					todo = append(todo, p)
				}
			}
		}

		if len(todo) == 0 {
			printer.PrintMessage("Nothing to migrate: %d secret(s) already use the keyring.", alreadyMigrated)
			return nil
		}

		if cli.GlobalState.IsDryRun() {
			for _, p := range todo {
				printer.PrintWarning("Dry run: would move the %s for %q into the keyring", p.label, p.name)
			}
			return nil
		}

		store := &config.SecretStore{AllowFileFallback: false}
		migrated := 0
		for _, p := range todo {
			newValue, ok, err := store.MigrateSecretToKeyring(p.key, p.secret)
			if err != nil {
				// Report and continue: one failure should not block the rest, and
				// a partially-migrated config is still valid because untouched
				// credentials keep their plaintext secret.
				printer.PrintError("Could not migrate the %s for %q: %v", p.label, p.name, err)
				continue
			}
			if ok {
				if !cfg.SetCredentialField(p.name, p.field, newValue) {
					// The secret is in the keyring but the file still holds the
					// plaintext; never count that as migrated.
					printer.PrintError("Could not update the %s for %q in the config file", p.label, p.name)
					continue
				}
				migrated++
			}
		}

		if migrated == 0 {
			return fmt.Errorf("no secrets were migrated")
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config after migrating %d secret(s): %w", migrated, err)
		}

		printer.PrintSuccess("Moved %d secret(s) into the OS keyring", migrated)
		if migrated < len(todo) {
			printer.PrintWarning("%d secret(s) could not be migrated and remain in the config file",
				len(todo)-migrated)
		}
		return nil
	},
}

var keyringStatusCmd = &cobra.Command{
	Use:   "keyring-status",
	Short: "Show whether the OS keyring is in use and where each secret lives",
	Long: `Report whether an OS keyring is available and, for each credential,
whether its client secret and environment token are held in the keyring or in
the config file.

Use this to confirm no plaintext secrets remain after 'config migrate-secrets'.`,
	Example: `  # Show keyring status
  dtiam config keyring-status

  # Machine-readable
  dtiam config keyring-status --plain`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		available := config.KeyringAvailable()

		rows := make([]map[string]any, 0, len(cfg.Credentials))
		for _, named := range cfg.Credentials {
			rows = append(rows, map[string]any{
				"credential":        named.Name,
				"secret":            secretLocation(named.Credential.ClientSecret),
				"environment_token": secretLocation(named.Credential.EnvironmentToken),
			})
		}

		if !available {
			printer.PrintWarning("No OS keyring available on this system (or DTIAM_DISABLE_KEYRING is set).")
		}

		return printer.Print(rows, keyringStatusColumns())
	},
}

// keyringStatusColumns returns the columns for keyring-status output.
func keyringStatusColumns() []output.Column {
	return []output.Column{
		{Key: "credential", Header: "CREDENTIAL"},
		{Key: "secret", Header: "SECRET LOCATION"},
		{Key: "environment_token", Header: "ENVIRONMENT TOKEN"},
	}
}

// secretLocation describes where a stored secret value lives.
func secretLocation(stored string) string {
	switch {
	case stored == "":
		return "not set"
	case config.IsKeyringReference(stored):
		return "OS keyring"
	default:
		return "config file (plaintext)"
	}
}
