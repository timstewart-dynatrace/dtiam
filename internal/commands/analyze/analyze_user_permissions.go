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
	"github.com/jtimothystewart/dtiam/pkg/output"
	"github.com/jtimothystewart/dtiam/pkg/utils"
)

var userPermissionsCmd = &cobra.Command{
	Use:   "user-permissions USER",
	Short: "Calculate effective permissions for a user",
	Long: `Calculate effective permissions for a user.

Shows all permissions granted to a user through their group memberships
and policy bindings.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		exportFile, _ := cmd.Flags().GetString("export")

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		calculator := utils.NewPermissionsCalculator(c)
		ctx := context.Background()

		result, err := calculator.GetUserEffectivePermissions(ctx, userID)
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
		fmt.Printf("=== Effective Permissions: %s ===\n", result.User.Email)
		fmt.Printf("UID: %s\n", result.User.UID)
		fmt.Printf("Groups: %d\n", result.GroupCount)
		fmt.Printf("Policy Bindings: %d\n", result.BindingCount)
		fmt.Printf("Unique Permissions: %d\n", result.PermissionCount)
		fmt.Println()

		// Groups
		if len(result.Groups) > 0 {
			fmt.Println("Group Memberships:")
			for _, group := range result.Groups {
				name := ""
				if n, ok := group["name"].(string); ok {
					name = n
				} else if n, ok := group["groupName"].(string); ok {
					name = n
				}
				fmt.Printf("  - %s\n", name)
			}
			fmt.Println()
		}

		// Permissions table
		if len(result.EffectivePermissions) > 0 {
			fmt.Println("Effective Permissions:")
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Effect", "Action", "Sources"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)

			for _, perm := range result.EffectivePermissions {
				var sources []string
				for _, s := range perm.Sources {
					sources = append(sources, fmt.Sprintf("%s/%s", s.Group, s.Policy))
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
	userPermissionsCmd.Flags().StringP("export", "e", "", "Export to file")
}
