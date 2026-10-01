package analyze

import (
	"context"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/output"
	"github.com/jtimothystewart/dtiam/pkg/resources"
	"github.com/jtimothystewart/dtiam/pkg/utils"
)

var policyCmd = &cobra.Command{
	Use:   "policy IDENTIFIER",
	Short: "Analyze a policy's permissions and bindings",
	Long:  `Analyze a policy's permissions and bindings. Shows what permissions a policy grants and where it's bound.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		policyHandler := resources.NewPolicyHandler(c)
		bindingHandler := resources.NewBindingHandler(c)
		groupHandler := resources.NewGroupHandler(c)
		ctx := context.Background()

		// Resolve policy
		policy, err := policyHandler.Resolve(ctx, identifier)
		if err != nil {
			return fmt.Errorf("policy not found: %s", identifier)
		}

		policyUUID := utils.StringFrom(policy, "uuid")
		policyName := utils.StringFrom(policy, "name")
		statement := ""
		if s, ok := policy["statementQuery"].(string); ok {
			statement = s
		}

		// Parse permissions
		permissions := utils.ParseStatementQuery(statement)

		// Find bindings
		allBindings, err := bindingHandler.List(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to list bindings: %w", err)
		}

		var boundGroups []map[string]any
		for _, binding := range allBindings {
			if binding["policyUuid"] == policyUUID {
				groupUUID := utils.StringFrom(binding, "groupUuid")
				group, _ := groupHandler.Get(ctx, groupUUID)

				groupInfo := map[string]any{
					"uuid": groupUUID,
				}
				if group != nil {
					groupInfo["name"] = group["name"]
				}
				if boundary, ok := binding["boundaryUuid"]; ok {
					groupInfo["boundary"] = boundary
				}
				boundGroups = append(boundGroups, groupInfo)
			}
		}

		result := map[string]any{
			"policy": map[string]any{
				"uuid":        policyUUID,
				"name":        policyName,
				"description": policy["description"],
				"statement":   statement,
			},
			"permissions":      permissions,
			"permission_count": len(permissions),
			"bindings":         boundGroups,
			"binding_count":    len(boundGroups),
		}

		// Check output format
		format := cli.GlobalState.GetOutputFormat()
		if format == output.FormatJSON || format == output.FormatYAML {
			printer := cli.GlobalState.NewPrinter()
			return printer.PrintAny(result)
		}

		// Formatted output
		fmt.Println()
		fmt.Printf("=== Policy Analysis: %s ===\n", policyName)
		fmt.Printf("UUID: %s\n", policyUUID)
		if desc, ok := policy["description"].(string); ok && desc != "" {
			fmt.Printf("Description: %s\n", desc)
		}
		fmt.Println()

		// Statement
		fmt.Println("Statement Query:")
		fmt.Println("---")
		fmt.Println(statement)
		fmt.Println("---")
		fmt.Println()

		// Permissions
		fmt.Printf("Parsed Permissions (%d):\n", len(permissions))
		if len(permissions) > 0 {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Effect", "Action", "Conditions"})
			table.SetBorder(false)

			for _, perm := range permissions {
				conditions := perm.Conditions
				if conditions == "" {
					conditions = "-"
				} else if len(conditions) > 40 {
					conditions = conditions[:37] + "..."
				}
				table.Append([]string{perm.Effect, perm.Action, conditions})
			}

			table.Render()
		} else {
			fmt.Println("  No permissions parsed.")
		}
		fmt.Println()

		// Bindings
		fmt.Printf("Bound to Groups (%d):\n", len(boundGroups))
		if len(boundGroups) > 0 {
			for _, group := range boundGroups {
				name := ""
				if n, ok := group["name"].(string); ok {
					name = n
				} else {
					name = utils.StringFrom(group, "uuid")
				}
				boundaryInfo := ""
				if boundary, ok := group["boundary"].(string); ok && boundary != "" {
					boundaryInfo = fmt.Sprintf(" (boundary: %s)", boundary)
				}
				fmt.Printf("  - %s%s\n", name, boundaryInfo)
			}
		} else {
			fmt.Println("  Not bound to any groups.")
		}

		return nil
	},
}
