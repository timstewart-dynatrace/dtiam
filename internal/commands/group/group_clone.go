package group

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/resources"
	"github.com/jtimothystewart/dtiam/pkg/utils"
)

var cloneCmd = &cobra.Command{
	Use:   "clone SOURCE",
	Short: "Clone a group with optional members and policies",
	Long: `Clone an existing group to create a new group with the same configuration.

By default, only the group name and description are cloned. Use --include-members
to copy all group members and --include-policies to copy all policy bindings
(including boundaries) to the new group.`,
	Example: `  # Clone a group with a new name
  dtiam group clone "Production Team" --name "Staging Team"

  # Clone with members and policies
  dtiam group clone "Production Team" --name "Staging Team" --include-members --include-policies

  # Clone with a custom description
  dtiam group clone "Production Team" --name "Staging Team" --description "Staging environment team"

  # Preview what would be cloned
  dtiam group clone "Production Team" --name "Staging Team" --include-members --include-policies --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		includeMembers, _ := cmd.Flags().GetBool("include-members")
		includePolicies, _ := cmd.Flags().GetBool("include-policies")

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would clone group %q as %q", args[0], name)
			if includeMembers {
				printer.PrintWarning("  --include-members: would copy all members")
			}
			if includePolicies {
				printer.PrintWarning("  --include-policies: would copy all policy bindings")
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		groupHandler := resources.NewGroupHandler(c)
		ctx := context.Background()

		// Resolve source group
		source, err := resources.GetOrResolve(ctx, groupHandler, args[0])
		if err != nil {
			return err
		}
		if source == nil {
			return fmt.Errorf("source group %q not found", args[0])
		}
		sourceUUID := utils.StringFrom(source, "uuid")

		// Use source description if none provided
		if description == "" {
			if srcDesc, ok := source["description"].(string); ok {
				description = srcDesc
			}
		}

		// Create new group
		newGroup, err := groupHandler.Create(ctx, map[string]any{
			"name":        name,
			"description": description,
		})
		if err != nil {
			return fmt.Errorf("failed to create group: %w", err)
		}
		newUUID := utils.StringFrom(newGroup, "uuid")
		printer.PrintSuccess("Group %q created (UUID: %s)", name, newUUID)

		// Copy members
		if includeMembers {
			members, err := groupHandler.GetMembers(ctx, sourceUUID)
			if err != nil {
				return fmt.Errorf("failed to get source members: %w", err)
			}
			copied := 0
			for _, member := range members {
				email, _ := member["email"].(string)
				if email == "" {
					continue
				}
				if err := groupHandler.AddMember(ctx, newUUID, email); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to add member %s: %v\n", email, err)
					continue
				}
				copied++
				if cli.GlobalState.IsVerbose() {
					fmt.Fprintf(os.Stderr, "  Added member: %s\n", email)
				}
			}
			printer.PrintSuccess("  Copied %d of %d member(s)", copied, len(members))
		}

		// Copy policy bindings
		if includePolicies {
			bindingHandler := resources.NewBindingHandler(c)
			bindings, err := bindingHandler.GetForGroup(ctx, sourceUUID)
			if err != nil {
				return fmt.Errorf("failed to get source bindings: %w", err)
			}
			copied := 0
			for _, binding := range bindings {
				policyUUID := utils.StringFrom(binding, "policyUuid")
				var boundaries []string
				if bList, ok := binding["boundaries"].([]string); ok {
					boundaries = bList
				}
				if _, err := bindingHandler.Create(ctx, newUUID, policyUUID, boundaries, nil); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to copy binding for policy %s: %v\n", policyUUID, err)
					continue
				}
				copied++
				if cli.GlobalState.IsVerbose() {
					fmt.Fprintf(os.Stderr, "  Copied binding: policy %s\n", policyUUID)
				}
			}
			printer.PrintSuccess("  Copied %d of %d binding(s)", copied, len(bindings))
		}

		return nil
	},
}

func init() {
	cloneCmd.Flags().StringP("name", "n", "", "Name for the new group (required)")
	cloneCmd.Flags().StringP("description", "d", "", "Description for the new group")
	cloneCmd.Flags().Bool("include-members", false, "Copy group members to the new group")
	cloneCmd.Flags().Bool("include-policies", false, "Copy policy bindings to the new group")
	_ = cloneCmd.MarkFlagRequired("name")
}
