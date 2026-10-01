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
by contexts via --credentials-ref.`,
	Example: `  # Create credentials for production
  dtiam config set-credentials prod --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY

  # Update existing credentials
  dtiam config set-credentials prod --client-id dt0s01.NEW --client-secret dt0s01.NEW.SECRET`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		clientID, _ := cmd.Flags().GetString("client-id")
		clientSecret, _ := cmd.Flags().GetString("client-secret")

		if clientID == "" || clientSecret == "" {
			return fmt.Errorf("both --client-id and --client-secret are required")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg.SetCredential(name, clientID, clientSecret)

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Credentials %q updated\n", name)
		return nil
	},
}

func init() {
	setCredentialsCmd.Flags().String("client-id", "", "OAuth2 client ID")
	setCredentialsCmd.Flags().String("client-secret", "", "OAuth2 client secret")
}

var deleteCredentialsCmd = &cobra.Command{
	Use:   "delete-credentials NAME",
	Short: "Delete a credential set",
	Long: `Remove a credential set from the configuration file.

Any contexts referencing these credentials will become invalid until updated
with a new credentials-ref.`,
	Example: `  # Delete the staging credentials
  dtiam config delete-credentials staging`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

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

		fmt.Printf("Credentials %q deleted\n", name)
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
