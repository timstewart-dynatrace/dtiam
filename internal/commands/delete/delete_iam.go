package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/prompt"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

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
