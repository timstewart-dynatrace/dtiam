package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/prompt"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

var userCmd = &cobra.Command{
	Use:   "user IDENTIFIER",
	Short: "Delete a user by UID or email",
	Long:  `Delete a user from the Dynatrace account. Requires confirmation unless --force is set.`,
	Example: `  # Delete a user by email
  dtiam delete user user@example.com

  # Delete without confirmation
  dtiam delete user user@example.com --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()
		force, _ := cmd.Flags().GetBool("force")

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would delete user: %s", args[0])
			return nil
		}

		if !prompt.ConfirmDelete("user", args[0], force || cli.GlobalState.IsPlain()) {
			printer.PrintMessage("Aborted.")
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		ctx := context.Background()

		user, err := handler.Get(ctx, args[0])
		if err != nil {
			user, err = handler.GetByEmail(ctx, args[0])
			if err != nil {
				return err
			}
		}
		if user == nil {
			return fmt.Errorf("user %q not found", args[0])
		}

		uid, _ := user["uid"].(string)
		if err := handler.Delete(ctx, uid); err != nil {
			return err
		}

		printer.PrintSuccess("User %q deleted successfully", args[0])
		return nil
	},
}

func init() {
	userCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

var serviceUserCmd = &cobra.Command{
	Use:     "service-user IDENTIFIER",
	Aliases: []string{"serviceuser"},
	Short:   "Delete a service user by name or UID",
	Long:    `Delete a service user (OAuth client) from the Dynatrace account. Requires confirmation unless --force is set.`,
	Example: `  # Delete a service user by name
  dtiam delete service-user "My Service Account"

  # Delete without confirmation
  dtiam delete service-user abc-123 --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()
		force, _ := cmd.Flags().GetBool("force")

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would delete service user: %s", args[0])
			return nil
		}

		if !prompt.ConfirmDelete("service user", args[0], force || cli.GlobalState.IsPlain()) {
			printer.PrintMessage("Aborted.")
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewServiceUserHandler(c)
		ctx := context.Background()

		user, err := handler.Get(ctx, args[0])
		if err != nil {
			user, err = handler.GetByName(ctx, args[0])
			if err != nil {
				return err
			}
		}
		if user == nil {
			return fmt.Errorf("service user %q not found", args[0])
		}

		uid, _ := user["uid"].(string)
		if err := handler.Delete(ctx, uid); err != nil {
			return err
		}

		printer.PrintSuccess("Service user %q deleted successfully", args[0])
		return nil
	},
}

func init() {
	serviceUserCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

var tokenCmd = &cobra.Command{
	Use:   "token IDENTIFIER",
	Short: "Delete a platform token by ID",
	Example: `  # Delete a token by ID
  dtiam delete token abc-123

  # Delete without confirmation
  dtiam delete token abc-123 --force

  # Preview deletion
  dtiam delete token abc-123 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would delete platform token: %s", args[0])
			return nil
		}

		if !prompt.ConfirmDelete("platform token", args[0], force || cli.GlobalState.IsPlain()) {
			printer.PrintMessage("Aborted.")
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewTokenHandler(c)
		ctx := context.Background()

		if err := handler.Delete(ctx, args[0]); err != nil {
			return err
		}

		printer.PrintSuccess("Platform token %q deleted successfully", args[0])
		return nil
	},
}

func init() {
	tokenCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}
