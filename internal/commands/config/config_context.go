package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
)

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
				"safety_level":    ctx.Context.EffectiveSafetyLevel(),
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
provided fields will be changed.

--safety-level limits what commands may do in the context:
  readonly    blocks every change and requests only read OAuth scopes
  no-delete   allows create and update, blocks deletes and anything that
              removes access (members, bindings, boundaries, permissions)
  readwrite   allows everything (the default)

Dry runs are allowed at every level.`,
	Example: `  # Create a new context
  dtiam config set-context prod --account-uuid abc-123 --credentials-ref prod-creds

  # Update only the account UUID of an existing context
  dtiam config set-context prod --account-uuid new-uuid-456

  # Make a context read-only
  dtiam config set-context prod --safety-level readonly`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		accountUUID, _ := cmd.Flags().GetString("account-uuid")
		credentialsRef, _ := cmd.Flags().GetString("credentials-ref")
		safetyLevel, _ := cmd.Flags().GetString("safety-level")
		if cmd.Flags().Changed("safety-level") {
			if _, err := config.ParseSafetyLevel(safetyLevel); err != nil {
				return err
			}
		}

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
		if cmd.Flags().Changed("safety-level") {
			if err := cfg.SetContextSafetyLevel(name, safetyLevel); err != nil {
				return err
			}
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
	setContextCmd.Flags().String("safety-level", "", "readonly, no-delete or readwrite")
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
