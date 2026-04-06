// Package group provides advanced group management commands.
package group

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// Cmd is the group command.
var Cmd = &cobra.Command{
	Use:   "group",
	Short: "Advanced group management commands",
	Long: `Commands for advanced group operations such as listing members,
adding and removing members, and viewing policy bindings for a group.

Groups can be identified by UUID or name in all subcommands.`,
	Example: `  # List members of a group
  dtiam group members "Production Team"

  # Add a user to a group
  dtiam group add-member "Production Team" --email user@example.com

  # View policy bindings for a group
  dtiam group bindings "Production Team"`,
}

func init() {
	Cmd.AddCommand(membersCmd)
	Cmd.AddCommand(addMemberCmd)
	Cmd.AddCommand(removeMemberCmd)
	Cmd.AddCommand(bindingsCmd)
}

var membersCmd = &cobra.Command{
	Use:   "members IDENTIFIER",
	Short: "List members of a group",
	Long: `List all members of a group.

The group can be identified by UUID or name. Returns user details
for all members in table format by default.`,
	Example: `  # List members by group name
  dtiam group members "Production Team"

  # List members by group UUID
  dtiam group members 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Output as JSON
  dtiam group members "Production Team" -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid, _ := group["uuid"].(string)
		members, err := handler.GetMembers(ctx, uuid)
		if err != nil {
			return err
		}

		return printer.Print(members, output.UserColumns())
	},
}

var addMemberCmd = &cobra.Command{
	Use:   "add-member IDENTIFIER",
	Short: "Add a user to a group",
	Long: `Add a user to a group by specifying the group and user email.

The group can be identified by UUID or name. The user is specified
via the --email flag.`,
	Example: `  # Add a user to a group by name
  dtiam group add-member "Production Team" --email user@example.com

  # Add a user to a group by UUID
  dtiam group add-member 8f6e5d4c-3b2a-1098-7654-321fedcba098 --email user@example.com

  # Dry run preview
  dtiam group add-member "Production Team" --email user@example.com --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			return fmt.Errorf("--email is required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would add user %s to group %s", email, args[0])
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid, _ := group["uuid"].(string)
		if err := handler.AddMember(ctx, uuid, email); err != nil {
			return err
		}

		printer.PrintSuccess("User %s added to group", email)
		return nil
	},
}

func init() {
	addMemberCmd.Flags().StringP("email", "e", "", "User email to add (required)")
}

var removeMemberCmd = &cobra.Command{
	Use:   "remove-member IDENTIFIER",
	Short: "Remove a user from a group",
	Long: `Remove a user from a group by specifying the group and user UID.

The group can be identified by UUID or name. The user is specified
via the --user flag with their UID.`,
	Example: `  # Remove a user from a group by name
  dtiam group remove-member "Production Team" --user USER_UID

  # Remove a user from a group by UUID
  dtiam group remove-member 8f6e5d4c-3b2a-1098-7654-321fedcba098 --user USER_UID

  # Dry run preview
  dtiam group remove-member "Production Team" --user USER_UID --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, _ := cmd.Flags().GetString("user")
		if userID == "" {
			return fmt.Errorf("--user is required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would remove user %s from group %s", userID, args[0])
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid, _ := group["uuid"].(string)
		if err := handler.RemoveMember(ctx, uuid, userID); err != nil {
			return err
		}

		printer.PrintSuccess("User removed from group")
		return nil
	},
}

func init() {
	removeMemberCmd.Flags().StringP("user", "u", "", "User UID to remove (required)")
}

var bindingsCmd = &cobra.Command{
	Use:   "bindings IDENTIFIER",
	Short: "List policy bindings for a group",
	Long: `List all policy bindings associated with a group.

The group can be identified by UUID or name. Returns binding details
including the policy and level information.`,
	Example: `  # List bindings by group name
  dtiam group bindings "Production Team"

  # List bindings by group UUID
  dtiam group bindings 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Output as JSON
  dtiam group bindings "Production Team" -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		groupHandler := resources.NewGroupHandler(c)
		bindingHandler := resources.NewBindingHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		group, err := resources.GetOrResolve(ctx, groupHandler, args[0])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group %q not found", args[0])
		}

		uuid, _ := group["uuid"].(string)
		bindings, err := bindingHandler.GetForGroup(ctx, uuid)
		if err != nil {
			return err
		}

		return printer.Print(bindings, output.BindingColumns())
	},
}
