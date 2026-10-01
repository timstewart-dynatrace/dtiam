package bulk

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

var createGroupsWithPoliciesCmd = &cobra.Command{
	Use:   "create-groups-with-policies",
	Short: "Create groups with policy bindings from a file",
	Long: `Create groups and bind policies from a CSV, JSON, or YAML file.

Each row/entry creates a group (or reuses an existing one by name), resolves
the policy by name, and creates a binding. An optional boundary can be specified.

CSV columns: group_name, description, policy_name, boundary_name
YAML format: array of objects with the same fields.`,
	Example: `  # Create groups with policies from CSV
  dtiam bulk create-groups-with-policies --file groups-policies.csv

  # Create from YAML
  dtiam bulk create-groups-with-policies --file groups-policies.yaml

  # Continue on errors
  dtiam bulk create-groups-with-policies --file groups-policies.csv --continue-on-error

  # Preview without creating
  dtiam bulk create-groups-with-policies --file groups-policies.csv --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		records, err := loadInputFile(filePath)
		if err != nil {
			return err
		}

		if len(records) == 0 {
			return fmt.Errorf("no records found in file")
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would process %d group-policy records from %s", len(records), filePath)
			for _, r := range records {
				fmt.Fprintf(os.Stderr, "  Group: %s → Policy: %s\n", r["group_name"], r["policy_name"])
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		groupHandler := resources.NewGroupHandler(c)
		policyHandler := resources.NewPolicyHandler(c)
		bindingHandler := resources.NewBindingHandler(c)
		boundaryHandler := resources.NewBoundaryHandler(c)
		ctx := context.Background()

		var successCount, failCount int
		groupCache := make(map[string]string) // name → UUID

		for i, record := range records {
			groupName := record["group_name"]
			policyName := record["policy_name"]
			description := record["description"]
			boundaryName := record["boundary_name"]

			if groupName == "" || policyName == "" {
				fmt.Fprintf(os.Stderr, "  Row %d: skipping (missing group_name or policy_name)\n", i+1)
				failCount++
				continue
			}

			fmt.Fprintf(os.Stderr, "Processing row %d: %s → %s\n", i+1, groupName, policyName)

			// Create or find group
			groupUUID, ok := groupCache[groupName]
			if !ok {
				existing, _ := groupHandler.Resolve(ctx, groupName)
				if existing != nil {
					groupUUID = utils.StringFrom(existing, "uuid")
					if cli.GlobalState.IsVerbose() {
						fmt.Fprintf(os.Stderr, "  Found existing group: %s (%s)\n", groupName, groupUUID)
					}
				} else {
					data := map[string]any{"name": groupName}
					if description != "" {
						data["description"] = description
					}
					newGroup, err := groupHandler.Create(ctx, data)
					if err != nil {
						failCount++
						fmt.Fprintf(os.Stderr, "  Failed to create group %q: %v\n", groupName, err)
						if !continueOnError {
							return err
						}
						continue
					}
					groupUUID = utils.StringFrom(newGroup, "uuid")
					fmt.Fprintf(os.Stderr, "  Created group: %s (%s)\n", groupName, groupUUID)
				}
				groupCache[groupName] = groupUUID
			}

			// Resolve policy
			policy, err := resources.GetOrResolve(ctx, policyHandler, policyName)
			if err != nil || policy == nil {
				failCount++
				fmt.Fprintf(os.Stderr, "  Policy %q not found\n", policyName)
				if !continueOnError {
					return fmt.Errorf("policy %q not found", policyName)
				}
				continue
			}
			policyUUID := utils.StringFrom(policy, "uuid")

			// Resolve optional boundary
			var boundaries []string
			if boundaryName != "" {
				b, err := resources.GetOrResolve(ctx, boundaryHandler, boundaryName)
				if err != nil || b == nil {
					fmt.Fprintf(os.Stderr, "  Warning: boundary %q not found, creating binding without boundary\n", boundaryName)
				} else {
					boundaries = []string{utils.StringFrom(b, "uuid")}
				}
			}

			// Create binding
			_, err = bindingHandler.Create(ctx, groupUUID, policyUUID, boundaries, nil)
			if err != nil {
				failCount++
				fmt.Fprintf(os.Stderr, "  Failed to create binding: %v\n", err)
				if !continueOnError {
					return err
				}
				continue
			}

			successCount++
		}

		printer.PrintSuccess("Processed %d records: %d succeeded, %d failed", len(records), successCount, failCount)
		return nil
	},
}

func init() {
	createGroupsWithPoliciesCmd.Flags().StringP("file", "f", "", "Input file (CSV, JSON, or YAML) (required)")
	createGroupsWithPoliciesCmd.Flags().Bool("continue-on-error", false, "Continue processing on errors")
	_ = createGroupsWithPoliciesCmd.MarkFlagRequired("file")
}
