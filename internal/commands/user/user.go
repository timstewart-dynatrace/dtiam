// Package user provides user management commands.
package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

// Cmd is the user command.
var Cmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long: `Commands for managing user operations such as group membership,
user creation, and group listing.

Use subcommands to add users to groups, remove them, replace their
group memberships, list their groups, or create new users.`,
	Example: `  # Add a user to groups
  dtiam user add-to-groups user@example.com --groups "My Group"

  # List groups a user belongs to
  dtiam user list-groups user@example.com

  # Create a new user
  dtiam user create newuser@example.com --first-name John --last-name Doe`,
}

func init() {
	Cmd.AddCommand(addToGroupsCmd)
	Cmd.AddCommand(removeFromGroupsCmd)
	Cmd.AddCommand(replaceGroupsCmd)
	Cmd.AddCommand(listGroupsCmd)
	Cmd.AddCommand(createCmd)
}

var addToGroupsCmd = &cobra.Command{
	Use:   "add-to-groups EMAIL",
	Short: "Add a user to one or more groups",
	Long: `Add a user to one or more groups by email address.

Groups can be specified by UUID or name. Multiple groups can be provided
as a comma-separated list. Group names are resolved to UUIDs automatically.`,
	Example: `  # Add user to a single group by UUID
  dtiam user add-to-groups user@example.com --groups 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Add user to a group by name
  dtiam user add-to-groups user@example.com --groups "Production Team"

  # Add user to multiple groups
  dtiam user add-to-groups user@example.com --groups "Group A,Group B,Group C"

  # Dry run preview
  dtiam user add-to-groups user@example.com --groups "My Group" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		groupsStr, _ := cmd.Flags().GetString("groups")

		if groupsStr == "" {
			return fmt.Errorf("--groups is required")
		}

		groups := strings.Split(groupsStr, ",")
		for i := range groups {
			groups[i] = strings.TrimSpace(groups[i])
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would add user %s to groups: %s", email, groupsStr)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		ctx := context.Background()

		// Resolve group names to UUIDs
		groupHandler := resources.NewGroupHandler(c)
		groupUUIDs := make([]string, 0, len(groups))
		for _, g := range groups {
			group, err := resources.GetOrResolve(ctx, groupHandler, g)
			if err != nil {
				return fmt.Errorf("failed to resolve group %q: %w", g, err)
			}
			if group == nil {
				return fmt.Errorf("group %q not found", g)
			}
			groupUUIDs = append(groupUUIDs, utils.StringFrom(group, "uuid"))
		}

		if err := handler.AddToGroups(ctx, email, groupUUIDs); err != nil {
			return err
		}

		printer.PrintSuccess("User %s added to %d group(s)", email, len(groupUUIDs))
		return nil
	},
}

func init() {
	addToGroupsCmd.Flags().StringP("groups", "g", "", "Comma-separated list of group UUIDs or names (required)")
}

var removeFromGroupsCmd = &cobra.Command{
	Use:   "remove-from-groups EMAIL",
	Short: "Remove a user from one or more groups",
	Long: `Remove a user from one or more groups by email address.

Groups can be specified by UUID or name. Multiple groups can be provided
as a comma-separated list. Group names are resolved to UUIDs automatically.`,
	Example: `  # Remove user from a single group by UUID
  dtiam user remove-from-groups user@example.com --groups 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Remove user from a group by name
  dtiam user remove-from-groups user@example.com --groups "Staging Team"

  # Remove user from multiple groups
  dtiam user remove-from-groups user@example.com --groups "Group A,Group B"

  # Dry run preview
  dtiam user remove-from-groups user@example.com --groups "My Group" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		groupsStr, _ := cmd.Flags().GetString("groups")

		if groupsStr == "" {
			return fmt.Errorf("--groups is required")
		}

		groups := strings.Split(groupsStr, ",")
		for i := range groups {
			groups[i] = strings.TrimSpace(groups[i])
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would remove user %s from groups: %s", email, groupsStr)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		ctx := context.Background()

		// Resolve group names to UUIDs
		groupHandler := resources.NewGroupHandler(c)
		groupUUIDs := make([]string, 0, len(groups))
		for _, g := range groups {
			group, err := resources.GetOrResolve(ctx, groupHandler, g)
			if err != nil {
				return fmt.Errorf("failed to resolve group %q: %w", g, err)
			}
			if group == nil {
				return fmt.Errorf("group %q not found", g)
			}
			groupUUIDs = append(groupUUIDs, utils.StringFrom(group, "uuid"))
		}

		if err := handler.RemoveFromGroups(ctx, email, groupUUIDs); err != nil {
			return err
		}

		printer.PrintSuccess("User %s removed from %d group(s)", email, len(groupUUIDs))
		return nil
	},
}

func init() {
	removeFromGroupsCmd.Flags().StringP("groups", "g", "", "Comma-separated list of group UUIDs or names (required)")
}

var replaceGroupsCmd = &cobra.Command{
	Use:   "replace-groups EMAIL",
	Short: "Replace all group memberships for a user",
	Long: `Replace all group memberships for a user by email address.

This removes the user from all current groups and adds them to the
specified groups. Groups can be specified by UUID or name. Pass an
empty --groups flag to remove the user from all groups.`,
	Example: `  # Replace user's groups with a new set
  dtiam user replace-groups user@example.com --groups "Production,Staging"

  # Replace with a single group by UUID
  dtiam user replace-groups user@example.com --groups 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Remove user from all groups
  dtiam user replace-groups user@example.com

  # Dry run preview
  dtiam user replace-groups user@example.com --groups "New Group" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		groupsStr, _ := cmd.Flags().GetString("groups")

		var groups []string
		if groupsStr != "" {
			groups = strings.Split(groupsStr, ",")
			for i := range groups {
				groups[i] = strings.TrimSpace(groups[i])
			}
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would replace groups for user %s with: %s", email, groupsStr)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewUserHandler(c)
		ctx := context.Background()

		// Resolve group names to UUIDs
		groupHandler := resources.NewGroupHandler(c)
		groupUUIDs := make([]string, 0, len(groups))
		for _, g := range groups {
			group, err := resources.GetOrResolve(ctx, groupHandler, g)
			if err != nil {
				return fmt.Errorf("failed to resolve group %q: %w", g, err)
			}
			if group == nil {
				return fmt.Errorf("group %q not found", g)
			}
			groupUUIDs = append(groupUUIDs, utils.StringFrom(group, "uuid"))
		}

		if err := handler.ReplaceGroups(ctx, email, groupUUIDs); err != nil {
			return err
		}

		printer.PrintSuccess("User %s groups replaced with %d group(s)", email, len(groupUUIDs))
		return nil
	},
}

func init() {
	replaceGroupsCmd.Flags().StringP("groups", "g", "", "Comma-separated list of group UUIDs or names")
}

var listGroupsCmd = &cobra.Command{
	Use:   "list-groups IDENTIFIER",
	Short: "List groups a user belongs to",
	Long: `List all groups that a user belongs to.

The user can be identified by UID or email address. If a UID lookup
fails, the command automatically falls back to searching by email.`,
	Example: `  # List groups by email
  dtiam user list-groups user@example.com

  # List groups by UID
  dtiam user list-groups 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Output as JSON
  dtiam user list-groups user@example.com -o json`,
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

		uid, _ := user["uid"].(string)
		groups, err := handler.GetGroups(ctx, uid)
		if err != nil {
			return err
		}

		return printer.Print(groups, output.GroupColumns())
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
