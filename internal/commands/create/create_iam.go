package create

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

  # Create a binding with parameters for parameterized policies
  dtiam create binding --group GROUP_UUID --policy POLICY_UUID --param env=production --param region=us-east

  # Dry run to preview
  dtiam create binding --group GROUP_UUID --policy POLICY_UUID --dry-run

  # Machine-friendly output
  dtiam create binding --group GROUP_UUID --policy POLICY_UUID -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group")
		policyID, _ := cmd.Flags().GetString("policy")
		boundaries, _ := cmd.Flags().GetStringSlice("boundary")
		params, _ := cmd.Flags().GetStringSlice("param")

		if groupID == "" {
			return fmt.Errorf("--group is required")
		}
		if policyID == "" {
			return fmt.Errorf("--policy is required")
		}

		// Parse --param key=value pairs
		var parameters map[string]string
		if len(params) > 0 {
			parameters = make(map[string]string)
			for _, p := range params {
				parts := strings.SplitN(p, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --param format %q: expected key=value", p)
				}
				parameters[parts[0]] = parts[1]
			}
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create binding: group=%s policy=%s", groupID, policyID)
			if len(parameters) > 0 {
				printer.PrintWarning("  with parameters: %v", parameters)
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBindingHandler(c)
		ctx := context.Background()

		binding, err := handler.Create(ctx, groupID, policyID, boundaries, parameters)
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
	bindingCmd.Flags().StringSlice("param", nil, "Bind parameters as key=value (repeatable)")
}
