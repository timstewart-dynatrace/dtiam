// Package boundary provides boundary management commands.
package boundary

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// Cmd is the boundary command.
var Cmd = &cobra.Command{
	Use:   "boundary",
	Short: "Attach, detach, and inspect boundaries on policy bindings",
	Long: `Commands for attaching and detaching boundaries from policy bindings.

Boundaries restrict the scope of a policy binding to a subset of resources
such as management zones, app IDs, or schema IDs. Use these commands to
manage which boundaries are associated with specific group/policy bindings.`,
	Example: `  # Attach a boundary to a binding
  dtiam boundary attach --group GROUP_UUID --policy POLICY_UUID --boundary BOUNDARY_UUID

  # Detach a boundary from a binding
  dtiam boundary detach --group GROUP_UUID --policy POLICY_UUID --boundary BOUNDARY_UUID

  # List policies using a specific boundary
  dtiam boundary list-attached BOUNDARY_UUID`,
}

func init() {
	Cmd.AddCommand(attachCmd)
	Cmd.AddCommand(detachCmd)
	Cmd.AddCommand(listAttachedCmd)
}

var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach a boundary to a policy binding",
	Long: `Attach a boundary to an existing policy binding identified by group and policy UUIDs.

The boundary constrains the scope of the policy for the specified group. All three
flags (--group, --policy, --boundary) are required.`,
	Example: `  # Attach a boundary to a group/policy binding
  dtiam boundary attach --group abc-123 --policy def-456 --boundary ghi-789

  # Preview with dry run
  dtiam boundary attach --group abc-123 --policy def-456 --boundary ghi-789 --dry-run

  # Short flags
  dtiam boundary attach -g abc-123 -p def-456 -b ghi-789`,
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group")
		policyID, _ := cmd.Flags().GetString("policy")
		boundaryID, _ := cmd.Flags().GetString("boundary")

		if groupID == "" || policyID == "" || boundaryID == "" {
			return fmt.Errorf("--group, --policy, and --boundary are all required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would attach boundary %s to binding (group=%s, policy=%s)", boundaryID, groupID, policyID)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		ctx := context.Background()

		if err := handler.AddBoundary(ctx, groupID, policyID, boundaryID); err != nil {
			return err
		}

		printer.PrintSuccess("Boundary attached successfully")
		return nil
	},
}

func init() {
	attachCmd.Flags().StringP("group", "g", "", "Group UUID (required)")
	attachCmd.Flags().StringP("policy", "p", "", "Policy UUID (required)")
	attachCmd.Flags().StringP("boundary", "b", "", "Boundary UUID (required)")
}

var detachCmd = &cobra.Command{
	Use:   "detach",
	Short: "Detach a boundary from a policy binding",
	Long: `Detach a boundary from an existing policy binding identified by group and policy UUIDs.

Removing a boundary widens the scope of the policy for the specified group back to
its unrestricted state. All three flags (--group, --policy, --boundary) are required.`,
	Example: `  # Detach a boundary from a group/policy binding
  dtiam boundary detach --group abc-123 --policy def-456 --boundary ghi-789

  # Preview with dry run
  dtiam boundary detach --group abc-123 --policy def-456 --boundary ghi-789 --dry-run

  # Short flags
  dtiam boundary detach -g abc-123 -p def-456 -b ghi-789`,
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group")
		policyID, _ := cmd.Flags().GetString("policy")
		boundaryID, _ := cmd.Flags().GetString("boundary")

		if groupID == "" || policyID == "" || boundaryID == "" {
			return fmt.Errorf("--group, --policy, and --boundary are all required")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would detach boundary %s from binding (group=%s, policy=%s)", boundaryID, groupID, policyID)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		ctx := context.Background()

		if err := handler.RemoveBoundary(ctx, groupID, policyID, boundaryID); err != nil {
			return err
		}

		printer.PrintSuccess("Boundary detached successfully")
		return nil
	},
}

func init() {
	detachCmd.Flags().StringP("group", "g", "", "Group UUID (required)")
	detachCmd.Flags().StringP("policy", "p", "", "Policy UUID (required)")
	detachCmd.Flags().StringP("boundary", "b", "", "Boundary UUID (required)")
}

var listAttachedCmd = &cobra.Command{
	Use:   "list-attached IDENTIFIER",
	Short: "List policies using a boundary",
	Long: `List all policy bindings that reference a given boundary.

IDENTIFIER can be a boundary UUID or name. If a name is provided it will be
resolved to the corresponding UUID automatically.`,
	Example: `  # List policies attached to a boundary by UUID
  dtiam boundary list-attached abc-123-def-456

  # List policies attached to a boundary by name
  dtiam boundary list-attached "Production Zone Boundary"

  # Output as JSON
  dtiam boundary list-attached abc-123-def-456 -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		boundary, err := resources.GetOrResolve(ctx, handler, args[0])
		if err != nil {
			return err
		}
		if boundary == nil {
			return fmt.Errorf("boundary %q not found", args[0])
		}

		uuid, _ := boundary["uuid"].(string)
		attached, err := handler.GetAttachedPolicies(ctx, uuid)
		if err != nil {
			return err
		}

		if len(attached) == 0 {
			fmt.Println("No policies are using this boundary.")
			return nil
		}

		// Define columns for attached policies
		columns := []struct {
			Key    string
			Header string
		}{
			{"policyUuid", "POLICY_UUID"},
			{"groupUuid", "GROUP_UUID"},
		}

		// Convert to output.Column format
		var cols []struct {
			Key    string
			Header string
		}
		for _, c := range columns {
			cols = append(cols, c)
		}

		return printer.Print(attached, nil)
	},
}
