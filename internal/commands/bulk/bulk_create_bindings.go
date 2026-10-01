package bulk

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/resources"
	"github.com/jtimothystewart/dtiam/pkg/utils"
)

var createBindingsCmd = &cobra.Command{
	Use:   "create-bindings",
	Short: "Create multiple policy bindings from a file",
	Long: `Create multiple policy bindings from a file.

JSON/YAML example:
  bindings:
    - group: "group-uuid-or-name"
      policy: "policy-uuid-or-name"
      boundary: "optional-boundary-uuid"  # optional
    - group: "another-group"
      policy: "another-policy"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		if filePath == "" {
			return fmt.Errorf("--file is required")
		}

		// Load YAML file
		records, err := loadYAMLFile(filePath, "bindings")
		if err != nil {
			return err
		}

		if len(records) == 0 {
			fmt.Println("Warning: No records found in file.")
			return nil
		}

		// Validate records have group and policy
		var validBindings []map[string]any
		for _, record := range records {
			if _, ok := record["group"]; !ok {
				fmt.Printf("Warning: Record missing 'group' field: %v\n", record)
				continue
			}
			if _, ok := record["policy"]; !ok {
				fmt.Printf("Warning: Record missing 'policy' field: %v\n", record)
				continue
			}
			validBindings = append(validBindings, record)
		}

		if len(validBindings) == 0 {
			return fmt.Errorf("no valid binding definitions found")
		}

		fmt.Printf("Found %d bindings to create\n", len(validBindings))

		// Dry run check
		if cli.GlobalState.IsDryRun() {
			fmt.Println("Would create the following bindings:")
			for _, binding := range validBindings {
				fmt.Printf("  - Group: %s -> Policy: %s\n", binding["group"], binding["policy"])
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		bindingHandler := resources.NewBindingHandler(c)
		groupHandler := resources.NewGroupHandler(c)
		policyHandler := resources.NewPolicyHandler(c)
		ctx := context.Background()

		// Process creations
		var successCount, failCount int
		for _, bindingDef := range validBindings {
			groupID := utils.StringFrom(bindingDef, "group")
			policyID := utils.StringFrom(bindingDef, "policy")

			// Resolve group
			group, err := groupHandler.Resolve(ctx, groupID)
			if err != nil {
				failCount++
				fmt.Printf("  Failed: Group not found: %s\n", groupID)
				if !continueOnError {
					return fmt.Errorf("group not found: %s", groupID)
				}
				continue
			}
			groupUUID := utils.StringFrom(group, "uuid")

			// Resolve policy
			policy, err := policyHandler.Resolve(ctx, policyID)
			if err != nil {
				failCount++
				fmt.Printf("  Failed: Policy not found: %s\n", policyID)
				if !continueOnError {
					return fmt.Errorf("policy not found: %s", policyID)
				}
				continue
			}
			policyUUID := utils.StringFrom(policy, "uuid")

			// Optional boundary
			var boundaries []string
			if boundaryID, ok := bindingDef["boundary"].(string); ok && boundaryID != "" {
				boundaries = []string{boundaryID}
			}

			_, err = bindingHandler.Create(ctx, groupUUID, policyUUID, boundaries, nil)
			if err != nil {
				failCount++
				fmt.Printf("  Failed to create binding %s -> %s: %v\n", groupID, policyID, err)
				if !continueOnError {
					return err
				}
			} else {
				successCount++
				if cli.GlobalState.IsVerbose() {
					fmt.Printf("  Created: %s -> %s\n", groupID, policyID)
				}
			}
		}

		fmt.Printf("\nSuccessfully created: %d bindings\n", successCount)
		if failCount > 0 {
			fmt.Printf("Failed: %d bindings\n", failCount)
		}

		return nil
	},
}

func init() {
	createBindingsCmd.Flags().StringP("file", "f", "", "File with binding definitions (JSON or YAML)")
	createBindingsCmd.Flags().Bool("continue-on-error", false, "Continue processing on errors")
}
