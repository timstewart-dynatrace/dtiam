package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/config"
	"github.com/jtimothystewart/dtiam/internal/output"
)

var migrateSecretsCmd = &cobra.Command{
	Use:   "migrate-secrets",
	Short: "Move plaintext client secrets into the OS keyring",
	Long: `Move any plaintext client secrets in the config file into the OS keyring,
replacing each with a reference.

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
		type pending struct {
			name   string
			secret string
		}
		var todo []pending
		alreadyMigrated := 0

		for _, named := range cfg.Credentials {
			secret := named.Credential.ClientSecret
			switch {
			case secret == "":
				continue
			case config.IsKeyringReference(secret):
				alreadyMigrated++
			default:
				todo = append(todo, pending{name: named.Name, secret: secret})
			}
		}

		if len(todo) == 0 {
			printer.PrintMessage("Nothing to migrate: %d credential(s) already use the keyring.", alreadyMigrated)
			return nil
		}

		if cli.GlobalState.IsDryRun() {
			for _, p := range todo {
				printer.PrintWarning("Dry run: would move the secret for %q into the keyring", p.name)
			}
			return nil
		}

		store := &config.SecretStore{AllowFileFallback: false}
		migrated := 0
		for _, p := range todo {
			newValue, ok, err := store.MigrateSecretToKeyring(p.name, p.secret)
			if err != nil {
				// Report and continue: one failure should not block the rest, and
				// a partially-migrated config is still valid because untouched
				// credentials keep their plaintext secret.
				printer.PrintError("Could not migrate %q: %v", p.name, err)
				continue
			}
			if ok {
				cfg.SetCredentialField(p.name, "client-secret", newValue)
				migrated++
			}
		}

		if migrated == 0 {
			return fmt.Errorf("no secrets were migrated")
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config after migrating %d secret(s): %w", migrated, err)
		}

		printer.PrintSuccess("Moved %d client secret(s) into the OS keyring", migrated)
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
whether its client secret is held in the keyring or in the config file.

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
			location := "config file (plaintext)"
			if named.Credential.ClientSecret == "" {
				location = "not set"
			} else if config.IsKeyringReference(named.Credential.ClientSecret) {
				location = "OS keyring"
			}
			rows = append(rows, map[string]any{
				"credential": named.Name,
				"secret":     location,
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
	}
}
