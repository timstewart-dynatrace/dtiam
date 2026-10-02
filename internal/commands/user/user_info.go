package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

var infoCmd = &cobra.Command{
	Use:   "info IDENTIFIER",
	Short: "Show detailed user information",
	Long: `Display detailed information about a specific IAM user.

Shows all user fields including UID, email, name, status, group memberships,
and associated permissions. The IDENTIFIER can be a user UID or email address.

This is equivalent to 'dtiam describe user'.`,
	Example: `  # Show user info by email
  dtiam user info alice@example.com

  # Show user info by UID
  dtiam user info 8f2e4a6b-1c3d-5e7f-9a0b-2c4d6e8f0a1b

  # Output as JSON
  dtiam user info alice@example.com -o json

  # Machine-friendly output
  dtiam user info alice@example.com --plain`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
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

		// Get expanded information
		uid, _ := user["uid"].(string)
		if uid != "" {
			expanded, err := handler.GetExpanded(ctx, uid)
			if err == nil {
				user = expanded
			}
		}

		return printer.PrintDetail(user)
	},
}

var createCmd = &cobra.Command{
	Use:   "create EMAIL",
	Short: "Create a new user",
	Long: `Create a new user with the specified email address.

Optionally provide a first name, last name, and initial group memberships.
Groups are specified as comma-separated UUIDs.`,
	Example: `  # Create a user with just an email
  dtiam user create newuser@example.com

  # Create a user with full details
  dtiam user create newuser@example.com --first-name John --last-name Doe

  # Create a user and add to groups
  dtiam user create newuser@example.com --groups GROUP_UUID1,GROUP_UUID2

  # Dry run preview
  dtiam user create newuser@example.com --first-name Jane --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")
		groupsStr, _ := cmd.Flags().GetString("groups")

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create user: %s", email)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		ctx := context.Background()

		var firstNamePtr, lastNamePtr *string
		if firstName != "" {
			firstNamePtr = &firstName
		}
		if lastName != "" {
			lastNamePtr = &lastName
		}

		var groups []string
		if groupsStr != "" {
			groups = strings.Split(groupsStr, ",")
			for i := range groups {
				groups[i] = strings.TrimSpace(groups[i])
			}
		}

		user, err := handler.Create(ctx, email, firstNamePtr, lastNamePtr, groups)
		if err != nil {
			return err
		}

		printer.PrintSuccess("User created successfully")
		return printer.Print([]map[string]any{user}, output.UserColumns())
	},
}

func init() {
	createCmd.Flags().String("first-name", "", "User's first name")
	createCmd.Flags().String("last-name", "", "User's last name")
	createCmd.Flags().StringP("groups", "g", "", "Comma-separated list of group UUIDs")
}
