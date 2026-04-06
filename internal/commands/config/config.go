// Package config provides configuration management commands.
package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/config"
	"github.com/jtimothystewart/dtiam/internal/output"
)

// Cmd is the config command.
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage dtiam contexts, credentials, and configuration",
	Long: `Commands for managing dtiam contexts and credentials.

dtiam uses a kubeconfig-style configuration file to store multiple contexts
and credential sets. Each context references an account UUID and a named
credential. Switch between contexts to manage different Dynatrace accounts.`,
	Example: `  # View the current configuration
  dtiam config view

  # Set up credentials and a context
  dtiam config set-credentials prod --client-id dt0s01.XXX --client-secret dt0s01.XXX.YYY
  dtiam config set-context prod --account-uuid abc-123 --credentials-ref prod
  dtiam config use-context prod

  # List all contexts
  dtiam config get-contexts

  # Show config file location
  dtiam config path`,
}

func init() {
	Cmd.AddCommand(viewCmd)
	Cmd.AddCommand(pathCmd)
	Cmd.AddCommand(getContextsCmd)
	Cmd.AddCommand(currentContextCmd)
	Cmd.AddCommand(useContextCmd)
	Cmd.AddCommand(setContextCmd)
	Cmd.AddCommand(deleteContextCmd)
	Cmd.AddCommand(setCredentialsCmd)
	Cmd.AddCommand(deleteCredentialsCmd)
	Cmd.AddCommand(getCredentialsCmd)
}

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "Display the current configuration",
	Long: `Display the full configuration file in YAML format.

Secrets are masked by default. Use --show-secrets to reveal client secrets.`,
	Example: `  # View configuration with masked secrets
  dtiam config view

  # View configuration with secrets visible
  dtiam config view --show-secrets

  # Pipe to a file for backup
  dtiam config view --show-secrets > config-backup.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Mask secrets unless --show-secrets is provided
		showSecrets, _ := cmd.Flags().GetBool("show-secrets")
		if !showSecrets {
			for i := range cfg.Credentials {
				cfg.Credentials[i].Credential.ClientSecret = config.MaskSecret(cfg.Credentials[i].Credential.ClientSecret)
			}
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		fmt.Print(string(data))
		return nil
	},
}

func init() {
	viewCmd.Flags().Bool("show-secrets", false, "Show unmasked secrets")
}

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Display the configuration file path",
	Long:  "Print the absolute path to the dtiam configuration file.",
	Example: `  # Show the config file path
  dtiam config path`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.GetConfigPath()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

var getContextsCmd = &cobra.Command{
	Use:     "get-contexts",
	Aliases: []string{"contexts"},
	Short:   "List all configured contexts",
	Long: `List all contexts defined in the configuration file.

The current context is marked with an asterisk (*).`,
	Example: `  # List all contexts
  dtiam config get-contexts

  # Output as JSON
  dtiam config get-contexts -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		printer := cli.GlobalState.NewPrinter()

		// Build data for output
		data := make([]map[string]any, len(cfg.Contexts))
		for i, ctx := range cfg.Contexts {
			current := ""
			if ctx.Name == cfg.CurrentContext {
				current = "*"
			}
			data[i] = map[string]any{
				"name":            ctx.Name,
				"account_uuid":    ctx.Context.AccountUUID,
				"credentials_ref": ctx.Context.CredentialsRef,
				"current":         current,
			}
		}

		return printer.Print(data, output.ContextColumns())
	},
}

var currentContextCmd = &cobra.Command{
	Use:   "current-context",
	Short: "Display the current context name",
	Long:  "Print the name of the currently active context, or a message if none is set.",
	Example: `  # Show current context
  dtiam config current-context`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if cfg.CurrentContext == "" {
			fmt.Println("No current context set")
		} else {
			fmt.Println(cfg.CurrentContext)
		}
		return nil
	},
}

var useContextCmd = &cobra.Command{
	Use:   "use-context NAME",
	Short: "Set the current context",
	Long: `Switch the active context to the one identified by NAME.

The context must already exist in the configuration file. Use set-context to
create a new context first.`,
	Example: `  # Switch to the production context
  dtiam config use-context production

  # Switch to staging
  dtiam config use-context staging`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.UseContext(name); err != nil {
			return err
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Switched to context %q\n", name)
		return nil
	},
}

var setContextCmd = &cobra.Command{
	Use:   "set-context NAME",
	Short: "Create or update a context",
	Long: `Create a new context or update an existing one.

A context links an account UUID to a named credential set. Both --account-uuid
and --credentials-ref are optional when updating an existing context; only the
provided fields will be changed.`,
	Example: `  # Create a new context
  dtiam config set-context prod --account-uuid abc-123 --credentials-ref prod-creds

  # Update only the account UUID of an existing context
  dtiam config set-context prod --account-uuid new-uuid-456`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		accountUUID, _ := cmd.Flags().GetString("account-uuid")
		credentialsRef, _ := cmd.Flags().GetString("credentials-ref")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		var accountPtr, credPtr *string
		if accountUUID != "" {
			accountPtr = &accountUUID
		}
		if credentialsRef != "" {
			credPtr = &credentialsRef
		}

		if err := cfg.SetContext(name, accountPtr, credPtr); err != nil {
			return err
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Context %q updated\n", name)
		return nil
	},
}

func init() {
	setContextCmd.Flags().String("account-uuid", "", "Account UUID")
	setContextCmd.Flags().String("credentials-ref", "", "Credentials reference name")
}

var deleteContextCmd = &cobra.Command{
	Use:   "delete-context NAME",
	Short: "Delete a context",
	Long: `Remove a context from the configuration file.

If the deleted context is the current context, no context will be active until
you run use-context again.`,
	Example: `  # Delete the staging context
  dtiam config delete-context staging`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !cfg.DeleteContext(name) {
			return fmt.Errorf("context %q not found", name)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Context %q deleted\n", name)
		return nil
	},
}

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
