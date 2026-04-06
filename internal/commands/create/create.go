// Package create provides commands for creating resources.
package create

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// Cmd is the create command.
var Cmd = &cobra.Command{
	Use:   "create",
	Short: "Create a resource",
	Long:  "Commands for creating IAM resources.",
}

func init() {
	Cmd.AddCommand(groupCmd)
	Cmd.AddCommand(policyCmd)
	Cmd.AddCommand(bindingCmd)
	Cmd.AddCommand(boundaryCmd)
}

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Create a new group",
	Example: `  # Create a group
  dtiam create group --name "Platform Team"

  # Create a group with a description
  dtiam create group --name "Platform Team" --description "Platform engineering team"

  # Dry run to preview
  dtiam create group --name "Platform Team" --dry-run

  # Machine-friendly output
  dtiam create group --name "Platform Team" -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create group: %s", name)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		ctx := context.Background()

		data := map[string]any{
			"name": name,
		}
		if description != "" {
			data["description"] = description
		}

		group, err := handler.Create(ctx, data)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Group created successfully")
		return printer.Print([]map[string]any{group}, output.GroupColumns())
	},
}

func init() {
	groupCmd.Flags().StringP("name", "n", "", "Group name (required)")
	groupCmd.Flags().StringP("description", "d", "", "Group description")
}

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Create a new policy",
	Example: `  # Create a policy with a statement
  dtiam create policy --name "Read Only" --statement "ALLOW iam:policies:read;"

  # Create a policy with a description
  dtiam create policy --name "Read Only" --statement "ALLOW iam:policies:read;" --description "Read-only access"

  # Dry run to preview
  dtiam create policy --name "Read Only" --statement "ALLOW iam:policies:read;" --dry-run

  # Machine-friendly output
  dtiam create policy --name "Read Only" --statement "ALLOW iam:policies:read;" -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		statement, _ := cmd.Flags().GetString("statement")
		description, _ := cmd.Flags().GetString("description")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if statement == "" {
			return fmt.Errorf("--statement is required")
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create policy: %s", name)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewPolicyHandler(c)
		ctx := context.Background()

		data := map[string]any{
			"name":           name,
			"statementQuery": statement,
		}
		if description != "" {
			data["description"] = description
		}

		policy, err := handler.Create(ctx, data)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Policy created successfully")
		return printer.Print([]map[string]any{policy}, output.PolicyColumns())
	},
}

func init() {
	policyCmd.Flags().StringP("name", "n", "", "Policy name (required)")
	policyCmd.Flags().StringP("statement", "s", "", "Policy statement query (required)")
	policyCmd.Flags().StringP("description", "d", "", "Policy description")
}

var bindingCmd = &cobra.Command{
	Use:   "binding",
	Short: "Create a new policy binding",
	Example: `  # Create a binding between a group and a policy
  dtiam create binding --group GROUP_UUID --policy POLICY_UUID

  # Create a binding with boundary constraints
  dtiam create binding --group GROUP_UUID --policy POLICY_UUID --boundary BOUNDARY_UUID

  # Dry run to preview
  dtiam create binding --group GROUP_UUID --policy POLICY_UUID --dry-run

  # Machine-friendly output
  dtiam create binding --group GROUP_UUID --policy POLICY_UUID -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group")
		policyID, _ := cmd.Flags().GetString("policy")
		boundaries, _ := cmd.Flags().GetStringSlice("boundary")

		if groupID == "" {
			return fmt.Errorf("--group is required")
		}
		if policyID == "" {
			return fmt.Errorf("--policy is required")
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create binding: group=%s policy=%s", groupID, policyID)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		ctx := context.Background()

		binding, err := handler.Create(ctx, groupID, policyID, boundaries)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Binding created successfully")
		return printer.Print([]map[string]any{binding}, output.BindingColumns())
	},
}

func init() {
	bindingCmd.Flags().StringP("group", "g", "", "Group UUID (required)")
	bindingCmd.Flags().StringP("policy", "p", "", "Policy UUID (required)")
	bindingCmd.Flags().StringSliceP("boundary", "b", nil, "Boundary UUIDs")
}

var boundaryCmd = &cobra.Command{
	Use:   "boundary",
	Short: "Create a new boundary",
	Example: `  # Create a boundary with management zones
  dtiam create boundary --name "Production" --zone "Production" --zone "Staging"

  # Create a boundary with a custom query
  dtiam create boundary --name "Apps Only" --query 'shared:app-id IN ("dynatrace.dashboards")'

  # Create a boundary with a description
  dtiam create boundary --name "Production" --zone "Production" --description "Production boundary"

  # Dry run to preview
  dtiam create boundary --name "Production" --zone "Production" --dry-run

  # Machine-friendly output
  dtiam create boundary --name "Production" --zone "Production" -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		zones, _ := cmd.Flags().GetStringSlice("zone")
		query, _ := cmd.Flags().GetString("query")
		description, _ := cmd.Flags().GetString("description")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if len(zones) == 0 && query == "" {
			return fmt.Errorf("either --zone or --query is required")
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create boundary: %s", name)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		ctx := context.Background()

		var queryPtr, descPtr *string
		if query != "" {
			queryPtr = &query
		}
		if description != "" {
			descPtr = &description
		}

		boundary, err := handler.Create(ctx, name, zones, queryPtr, descPtr)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Boundary created successfully")
		return printer.Print([]map[string]any{boundary}, output.BoundaryColumns())
	},
}

func init() {
	boundaryCmd.Flags().StringP("name", "n", "", "Boundary name (required)")
	boundaryCmd.Flags().StringSliceP("zone", "z", nil, "Management zone names")
	boundaryCmd.Flags().StringP("query", "q", "", "Boundary query")
	boundaryCmd.Flags().StringP("description", "d", "", "Boundary description")
}
