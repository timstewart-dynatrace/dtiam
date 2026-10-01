package analyze

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

var groupPermissionsCmd = &cobra.Command{
	Use:   "group-permissions GROUP",
	Short: "Calculate effective permissions for a group",
	Long: `Calculate effective permissions for a group.

Shows all permissions granted to a group through its policy bindings.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID := args[0]
		exportFile, _ := cmd.Flags().GetString("export")

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		calculator := utils.NewPermissionsCalculator(c)
		ctx := context.Background()

		result, err := calculator.GetGroupEffectivePermissions(ctx, groupID)
		if err != nil {
			return err
		}

		// Export to file
		if exportFile != "" {
			var data []byte
			if strings.HasSuffix(exportFile, ".json") {
				data, err = json.MarshalIndent(result, "", "  ")
			} else {
				data, err = yaml.Marshal(result)
			}
			if err != nil {
				return fmt.Errorf("failed to marshal data: %w", err)
			}
			if err := os.WriteFile(exportFile, data, 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			fmt.Printf("Exported to %s\n", exportFile)
			return nil
		}

		// Check output format
		format := cli.GlobalState.GetOutputFormat()
		if format == output.FormatJSON || format == output.FormatYAML {
			printer := cli.GlobalState.NewPrinter()
			return printer.PrintAny(result)
		}

		// Formatted output
		fmt.Println()
		fmt.Printf("=== Effective Permissions: %s ===\n", result.Group.Name)
		fmt.Printf("UUID: %s\n", result.Group.UUID)
		fmt.Printf("Policy Bindings: %d\n", result.BindingCount)
		fmt.Printf("Unique Permissions: %d\n", result.PermissionCount)
		fmt.Println()

		// Permissions table
		if len(result.EffectivePermissions) > 0 {
			fmt.Println("Effective Permissions:")
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Effect", "Action", "Source Policies"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)

			for _, perm := range result.EffectivePermissions {
				var sources []string
				for _, s := range perm.Sources {
					sources = append(sources, s.Policy)
				}
				sourcesStr := strings.Join(sources, ", ")
				if len(sourcesStr) > 50 {
					sourcesStr = sourcesStr[:47] + "..."
				}
				table.Append([]string{perm.Effect, perm.Action, sourcesStr})
			}

			table.Render()
		} else {
			fmt.Println("No permissions found.")
		}

		return nil
	},
}

func init() {
	groupPermissionsCmd.Flags().StringP("export", "e", "", "Export to file")
}
