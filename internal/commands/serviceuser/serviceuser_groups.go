package serviceuser

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

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
