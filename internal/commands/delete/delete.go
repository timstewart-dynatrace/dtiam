// Package delete provides commands for deleting resources.
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

// Cmd is the delete command.
var Cmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a resource",
	Long:  "Commands for deleting IAM resources. All delete operations require confirmation unless --force is set.",
}

func init() {
	Cmd.AddCommand(groupCmd)
	Cmd.AddCommand(policyCmd)
	Cmd.AddCommand(bindingCmd)
	Cmd.AddCommand(boundaryCmd)
	Cmd.AddCommand(userCmd)
	Cmd.AddCommand(serviceUserCmd)
}

var groupCmd = &cobra.Command{
	Use:   "group IDENTIFIER",
	Short: "Delete a group by name or UUID",
	Long:  `Delete a group from the Dynatrace account. Requires confirmation unless --force is set.`,
	Example: `  # Delete a group by name
  dtiam delete group "My Group"

  # Delete by UUID without confirmation
  dtiam delete group abc-123 --force

  # Preview deletion
  dtiam delete group "My Group" --dry-run

  # Machine-friendly (skip prompts, JSON output)
  dtiam delete group "My Group" --force --plain`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()
		force, _ := cmd.Flags().GetBool("force")

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would delete group: %s", args[0])
			return nil
		}

		if !prompt.ConfirmDelete("group", args[0], force || cli.GlobalState.IsPlain()) {
			printer.PrintMessage("Aborted.")
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
		if err := handler.Delete(ctx, uuid); err != nil {
			return err
		}

		printer.PrintSuccess("Group %q deleted successfully", args[0])
		return nil
	},
}

func init() {
	groupCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

var policyCmd = &cobra.Command{
	Use:   "policy IDENTIFIER",
	Short: "Delete a policy by name or UUID",
	Long:  `Delete a policy from the Dynatrace account. Requires confirmation unless --force is set.`,
	Example: `  # Delete a policy by name
  dtiam delete policy "Read Only"

  # Delete by UUID without confirmation
  dtiam delete policy abc-123 --force

  # Preview deletion
  dtiam delete policy "Read Only" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()
		force, _ := cmd.Flags().GetBool("force")

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would delete policy: %s", args[0])
			return nil
		}

		if !prompt.ConfirmDelete("policy", args[0], force || cli.GlobalState.IsPlain()) {
			printer.PrintMessage("Aborted.")
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewPolicyHandler(c)
		ctx := context.Background()

		policy, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if policy == nil {
			return fmt.Errorf("policy %q not found", args[0])
		}

		uuid, _ := policy["uuid"].(string)
		if err := handler.Delete(ctx, uuid); err != nil {
			return err
		}

		printer.PrintSuccess("Policy %q deleted successfully", args[0])
		return nil
	},
}

func init() {
	policyCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

var bindingCmd = &cobra.Command{
	Use:   "binding",
	Short: "Delete a policy binding",
	Long:  `Delete a policy binding between a group and a policy. Requires both --group and --policy flags.`,
	Example: `  # Delete a binding
  dtiam delete binding --group GROUP_UUID --policy POLICY_UUID

  # Delete without confirmation
  dtiam delete binding --group GROUP_UUID --policy POLICY_UUID --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()
		groupID, _ := cmd.Flags().GetString("group")
		policyID, _ := cmd.Flags().GetString("policy")
		force, _ := cmd.Flags().GetBool("force")

		if groupID == "" || policyID == "" {
			return fmt.Errorf("both --group and --policy are required")
		}

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would delete binding: group=%s policy=%s", groupID, policyID)
			return nil
		}

		if !prompt.Confirm(
			fmt.Sprintf("Delete binding for group %q and policy %q?", groupID, policyID),
			force || cli.GlobalState.IsPlain(),
		) {
			printer.PrintMessage("Aborted.")
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		ctx := context.Background()

		if err := handler.Delete(ctx, groupID, policyID); err != nil {
			return err
		}

		printer.PrintSuccess("Binding deleted successfully")
		return nil
	},
}

func init() {
	bindingCmd.Flags().StringP("group", "g", "", "Group UUID (required)")
	bindingCmd.Flags().StringP("policy", "p", "", "Policy UUID (required)")
	bindingCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

var boundaryCmd = &cobra.Command{
	Use:   "boundary IDENTIFIER",
	Short: "Delete a boundary by name or UUID",
	Long:  `Delete a boundary from the Dynatrace account. Requires confirmation unless --force is set.`,
	Example: `  # Delete a boundary by name
  dtiam delete boundary "My Boundary"

  # Delete by UUID without confirmation
  dtiam delete boundary abc-123 --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()
		force, _ := cmd.Flags().GetBool("force")

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would delete boundary: %s", args[0])
			return nil
		}

		if !prompt.ConfirmDelete("boundary", args[0], force || cli.GlobalState.IsPlain()) {
			printer.PrintMessage("Aborted.")
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		ctx := context.Background()

		boundary, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if boundary == nil {
			return fmt.Errorf("boundary %q not found", args[0])
		}

		uuid, _ := boundary["uuid"].(string)
		if err := handler.Delete(ctx, uuid); err != nil {
			return err
		}

		printer.PrintSuccess("Boundary %q deleted successfully", args[0])
		return nil
	},
}

func init() {
	boundaryCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

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
