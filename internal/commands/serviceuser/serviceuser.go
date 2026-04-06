// Package serviceuser provides service user (OAuth client) management commands.
package serviceuser

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// Cmd is the service-user command.
var Cmd = &cobra.Command{
	Use:     "service-user",
	Aliases: []string{"serviceuser"},
	Short:   "Service user (OAuth client) management commands",
	Long: `Commands for managing service users (OAuth clients).

Service users are non-human identities used for automation, CI/CD pipelines,
and API integrations. Use subcommands to list, create, update, delete, and
manage group memberships for service users.`,
	Example: `  # List all service users
  dtiam service-user list

  # Get details for a service user
  dtiam service-user get my-automation-user

  # Create a new service user
  dtiam service-user create --name "CI Pipeline" --description "CI/CD automation"

  # Add a service user to a group
  dtiam service-user add-to-group my-automation-user --group GROUP_UUID`,
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(addToGroupCmd)
	Cmd.AddCommand(removeFromGroupCmd)
	Cmd.AddCommand(listGroupsCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all service users",
	Long:  `List all service users in the account. Returns service user details in table format by default.`,
	Example: `  # List all service users
  dtiam service-user list

  # List as JSON
  dtiam service-user list -o json

  # List with verbose output
  dtiam service-user list -v`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewServiceUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		users, err := handler.List(ctx, nil)
		if err != nil {
			return err
		}

		return printer.Print(users, output.ServiceUserColumns())
	},
}

var getCmd = &cobra.Command{
	Use:   "get IDENTIFIER",
	Short: "Get details for a service user",
	Long: `Get detailed information about a specific service user.

The service user can be identified by UID or name. If a UID lookup
fails, the command automatically falls back to searching by name.`,
	Example: `  # Get by name
  dtiam service-user get my-automation-user

  # Get by UID
  dtiam service-user get 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Output as JSON
  dtiam service-user get my-automation-user -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewServiceUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
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

		return printer.PrintSingle(user, output.ServiceUserColumns())
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new service user",
	Long: `Create a new service user (OAuth client) with the specified name.

Optionally provide a description and initial group memberships. The
credentials returned at creation time cannot be retrieved later, so
save them immediately.`,
	Example: `  # Create a basic service user
  dtiam service-user create --name "CI Pipeline"

  # Create with description
  dtiam service-user create --name "CI Pipeline" --description "Used for CI/CD automation"

  # Create and add to groups
  dtiam service-user create --name "CI Pipeline" --groups GROUP_UUID1,GROUP_UUID2

  # Dry run preview
  dtiam service-user create --name "CI Pipeline" --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		groupsStr, _ := cmd.Flags().GetString("groups")

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create service user: %s", name)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewServiceUserHandler(c)
		ctx := context.Background()

		var descPtr *string
		if description != "" {
			descPtr = &description
		}

		var groups []string
		if groupsStr != "" {
			groups = strings.Split(groupsStr, ",")
			for i := range groups {
				groups[i] = strings.TrimSpace(groups[i])
			}
		}

		user, err := handler.Create(ctx, name, descPtr, groups)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Service user created successfully")
		printer.PrintWarning("Save the credentials below - they cannot be retrieved later!")
		return printer.PrintDetail(user)
	},
}

func init() {
	createCmd.Flags().StringP("name", "n", "", "Service user name (required)")
	createCmd.Flags().StringP("description", "d", "", "Service user description")
	createCmd.Flags().StringP("groups", "g", "", "Comma-separated list of group UUIDs")
}

var updateCmd = &cobra.Command{
	Use:   "update IDENTIFIER",
	Short: "Update a service user",
	Long: `Update an existing service user's name or description.

The service user can be identified by UID or name. If a UID lookup
fails, the command automatically falls back to searching by name.`,
	Example: `  # Update name
  dtiam service-user update my-automation-user --name "New Name"

  # Update description
  dtiam service-user update my-automation-user --description "Updated description"

  # Update both name and description
  dtiam service-user update my-automation-user --name "New Name" --description "New desc"

  # Dry run preview
  dtiam service-user update my-automation-user --name "New Name" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would update service user: %s", args[0])
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewServiceUserHandler(c)
		ctx := context.Background()

		// Find the user first
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

		var namePtr, descPtr *string
		if name != "" {
			namePtr = &name
		}
		if description != "" {
			descPtr = &description
		}

		updated, err := handler.Update(ctx, uid, namePtr, descPtr, nil)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Service user updated successfully")
		return printer.PrintSingle(updated, output.ServiceUserColumns())
	},
}

func init() {
	updateCmd.Flags().StringP("name", "n", "", "New name")
	updateCmd.Flags().StringP("description", "d", "", "New description")
}

var deleteCmd = &cobra.Command{
	Use:   "delete IDENTIFIER",
	Short: "Delete a service user",
	Long: `Delete a service user by UID or name.

The service user can be identified by UID or name. If a UID lookup
fails, the command automatically falls back to searching by name.
This action is irreversible.`,
	Example: `  # Delete by name
  dtiam service-user delete my-automation-user

  # Delete by UID
  dtiam service-user delete 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Dry run preview
  dtiam service-user delete my-automation-user --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would delete service user: %s", args[0])
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

		printer.PrintSuccess("Service user deleted successfully")
		return nil
	},
}

var addToGroupCmd = &cobra.Command{
	Use:   "add-to-group IDENTIFIER",
	Short: "Add a service user to a group",
	Long: `Add a service user to a group by specifying the service user and group UUID.

The service user can be identified by UID or name. The group is specified
via the --group flag with the group UUID.`,
	Example: `  # Add service user to a group by name
  dtiam service-user add-to-group my-automation-user --group GROUP_UUID

  # Add by UID
  dtiam service-user add-to-group 8f6e5d4c-3b2a-1098-7654-321fedcba098 --group GROUP_UUID

  # Dry run preview
  dtiam service-user add-to-group my-automation-user --group GROUP_UUID --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group")
		if groupID == "" {
			return fmt.Errorf("--group is required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would add service user %s to group %s", args[0], groupID)
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
		if err := handler.AddToGroup(ctx, uid, groupID); err != nil {
			return err
		}

		printer.PrintSuccess("Service user added to group")
		return nil
	},
}

func init() {
	addToGroupCmd.Flags().StringP("group", "g", "", "Group UUID (required)")
}

var removeFromGroupCmd = &cobra.Command{
	Use:   "remove-from-group IDENTIFIER",
	Short: "Remove a service user from a group",
	Long: `Remove a service user from a group by specifying the service user and group UUID.

The service user can be identified by UID or name. The group is specified
via the --group flag with the group UUID.`,
	Example: `  # Remove service user from a group by name
  dtiam service-user remove-from-group my-automation-user --group GROUP_UUID

  # Remove by UID
  dtiam service-user remove-from-group 8f6e5d4c-3b2a-1098-7654-321fedcba098 --group GROUP_UUID

  # Dry run preview
  dtiam service-user remove-from-group my-automation-user --group GROUP_UUID --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group")
		if groupID == "" {
			return fmt.Errorf("--group is required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would remove service user %s from group %s", args[0], groupID)
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
		if err := handler.RemoveFromGroup(ctx, uid, groupID); err != nil {
			return err
		}

		printer.PrintSuccess("Service user removed from group")
		return nil
	},
}

func init() {
	removeFromGroupCmd.Flags().StringP("group", "g", "", "Group UUID (required)")
}

var listGroupsCmd = &cobra.Command{
	Use:   "list-groups IDENTIFIER",
	Short: "List groups a service user belongs to",
	Long: `List all groups that a service user belongs to.

The service user can be identified by UID or name. If a UID lookup
fails, the command automatically falls back to searching by name.`,
	Example: `  # List groups by service user name
  dtiam service-user list-groups my-automation-user

  # List groups by UID
  dtiam service-user list-groups 8f6e5d4c-3b2a-1098-7654-321fedcba098

  # Output as JSON
  dtiam service-user list-groups my-automation-user -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewServiceUserHandler(c)
		printer := cli.GlobalState.NewPrinter()
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
		groups, err := handler.GetGroups(ctx, uid)
		if err != nil {
			return err
		}

		return printer.Print(groups, output.GroupColumns())
	},
}
