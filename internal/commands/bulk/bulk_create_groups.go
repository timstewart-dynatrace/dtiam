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

var createGroupsCmd = &cobra.Command{
	Use:   "create-groups",
	Short: "Create multiple groups from a file",
	Long: `Create multiple groups from a file.

JSON/YAML example:
  groups:
    - name: "Group A"
      description: "Description for Group A"
    - name: "Group B"
      description: "Description for Group B"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		if filePath == "" {
			return fmt.Errorf("--file is required")
		}

		// Load YAML file
		records, err := loadYAMLFile(filePath, "groups")
		if err != nil {
			return err
		}

		if len(records) == 0 {
			fmt.Println("Warning: No records found in file.")
			return nil
		}

		// Validate records have name
		var validGroups []map[string]any
		for _, record := range records {
			if _, ok := record["name"]; !ok {
				fmt.Printf("Warning: Record missing 'name' field: %v\n", record)
				continue
			}
			validGroups = append(validGroups, record)
		}

		if len(validGroups) == 0 {
			return fmt.Errorf("no valid group definitions found")
		}

		fmt.Printf("Found %d groups to create\n", len(validGroups))

		// Dry run check
		if cli.GlobalState.IsDryRun() {
			fmt.Println("Would create the following groups:")
			for _, group := range validGroups {
				fmt.Printf("  - %s\n", group["name"])
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		ctx := context.Background()

		// Process creations
		var successCount, failCount int
		for _, groupDef := range validGroups {
			name := utils.StringFrom(groupDef, "name")
			data := map[string]any{
				"name": name,
			}
			if desc, ok := groupDef["description"].(string); ok && desc != "" {
				data["description"] = desc
			}

			_, err := handler.Create(ctx, data)
			if err != nil {
				failCount++
				fmt.Printf("  Failed to create '%s': %v\n", name, err)
				if !continueOnError {
					return err
				}
			} else {
				successCount++
				if cli.GlobalState.IsVerbose() {
					fmt.Printf("  Created: %s\n", name)
				}
			}
		}

		fmt.Printf("\nSuccessfully created: %d groups\n", successCount)
		if failCount > 0 {
			fmt.Printf("Failed: %d groups\n", failCount)
		}

		return nil
	},
}

func init() {
	createGroupsCmd.Flags().StringP("file", "f", "", "File with group definitions (JSON or YAML)")
	createGroupsCmd.Flags().Bool("continue-on-error", false, "Continue processing on errors")
}
