package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

var usersCmd = &cobra.Command{
	Use:     "users [identifier]",
	Aliases: []string{"user"},
	Short:   "List IAM users or get a specific user by UID or email",
	Example: `  # List all users
  dtiam get users

  # Get a specific user by UID
  dtiam get users 12345678-abcd-1234-abcd-1234567890ab

  # Get a specific user by email
  dtiam get users alice@example.com

  # Output as JSON
  dtiam get users -o json

  # Output as YAML
  dtiam get users -o yaml

  # Machine-friendly output (no colors, no headers)
  dtiam get users --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		if len(args) > 0 {
			// Get single user
			user, err := handler.Get(ctx, args[0])
			if err != nil {
				// Try by email
				user, err = handler.GetByEmail(ctx, args[0])
				if err != nil {
					return err
				}
			}
			if user == nil {
				return fmt.Errorf("user %q not found", args[0])
			}
			return printer.PrintSingle(user, output.UserColumns())
		}

		if watchRequested(cmd) {
			return runWatch(cmd, func(ctx context.Context) ([]map[string]any, error) {
				return handler.List(ctx, nil)
			}, output.UserColumns())
		}

		// List all users
		users, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(users, output.UserColumns())
	},
}
