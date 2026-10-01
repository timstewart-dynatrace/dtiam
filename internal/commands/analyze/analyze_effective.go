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

var effectiveUserCmd = &cobra.Command{
	Use:   "effective-user USER",
	Short: "Get effective permissions for a user via the Dynatrace API",
	Long: `Get effective permissions for a user via the Dynatrace API.

This calls the Dynatrace resolution API directly to get permissions as
computed by the platform, which is the authoritative source.

Example:
  dtiam analyze effective-user admin@example.com
  dtiam analyze effective-user admin@example.com --level environment --level-id env123
  dtiam analyze effective-user admin@example.com --services settings,entities`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		levelType, _ := cmd.Flags().GetString("level")
		levelID, _ := cmd.Flags().GetString("level-id")
		servicesStr, _ := cmd.Flags().GetString("services")
		exportFile, _ := cmd.Flags().GetString("export")

		var services []string
		if servicesStr != "" {
			for _, s := range strings.Split(servicesStr, ",") {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					services = append(services, trimmed)
				}
			}
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		api := utils.NewEffectivePermissionsAPI(c)
		ctx := context.Background()

		result, err := api.GetUserEffectivePermissions(ctx, userID, levelType, levelID, services)
		if err != nil {
			return err
		}

		if result.Error != "" {
			return fmt.Errorf("%s", result.Error)
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
		fmt.Printf("=== Effective Permissions (API): %s ===\n", userID)
		fmt.Printf("Entity ID: %s\n", result.EntityID)
		fmt.Printf("Level: %s/%s\n", result.LevelType, result.LevelID)
		fmt.Printf("Total Permissions: %d\n", result.Total)
		fmt.Println()

		if len(result.EffectivePermissions) == 0 {
			fmt.Println("No effective permissions found.")
			return nil
		}

		// Display permissions in a table
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Permission", "Effect", "Service"})
		table.SetBorder(false)

		displayLimit := 50
		for i, perm := range result.EffectivePermissions {
			if i >= displayLimit {
				break
			}
			permName := ""
			if p, ok := perm["permission"].(string); ok {
				permName = p
			} else if p, ok := perm["name"].(string); ok {
				permName = p
			}
			effect := "ALLOW"
			if e, ok := perm["effect"].(string); ok {
				effect = e
			}
			service := "-"
			if s, ok := perm["service"].(string); ok {
				service = s
			}

			table.Append([]string{permName, effect, service})
		}

		table.Render()

		if len(result.EffectivePermissions) > displayLimit {
			fmt.Printf("\nShowing %d of %d permissions. Use --export for full list.\n", displayLimit, len(result.EffectivePermissions))
		}

		return nil
	},
}

func init() {
	effectiveUserCmd.Flags().StringP("level", "l", "account", "Level type: account, environment, global")
	effectiveUserCmd.Flags().String("level-id", "", "Level ID (uses account UUID if not specified)")
	effectiveUserCmd.Flags().StringP("services", "s", "", "Comma-separated service filter")
	effectiveUserCmd.Flags().StringP("export", "e", "", "Export to file")
}

var effectiveGroupCmd = &cobra.Command{
	Use:   "effective-group GROUP",
	Short: "Get effective permissions for a group via the Dynatrace API",
	Long: `Get effective permissions for a group via the Dynatrace API.

This calls the Dynatrace resolution API directly to get permissions as
computed by the platform, which is the authoritative source.

Example:
  dtiam analyze effective-group "DevOps Team"
  dtiam analyze effective-group "DevOps Team" --level environment --level-id env123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID := args[0]
		levelType, _ := cmd.Flags().GetString("level")
		levelID, _ := cmd.Flags().GetString("level-id")
		servicesStr, _ := cmd.Flags().GetString("services")
		exportFile, _ := cmd.Flags().GetString("export")

		var services []string
		if servicesStr != "" {
			for _, s := range strings.Split(servicesStr, ",") {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					services = append(services, trimmed)
				}
			}
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		api := utils.NewEffectivePermissionsAPI(c)
		ctx := context.Background()

		result, err := api.GetGroupEffectivePermissions(ctx, groupID, levelType, levelID, services)
		if err != nil {
			return err
		}

		if result.Error != "" {
			return fmt.Errorf("%s", result.Error)
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
		fmt.Printf("=== Effective Permissions (API): %s ===\n", groupID)
		fmt.Printf("Entity ID: %s\n", result.EntityID)
		fmt.Printf("Level: %s/%s\n", result.LevelType, result.LevelID)
		fmt.Printf("Total Permissions: %d\n", result.Total)
		fmt.Println()

		if len(result.EffectivePermissions) == 0 {
			fmt.Println("No effective permissions found.")
			return nil
		}

		// Display permissions in a table
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Permission", "Effect", "Service"})
		table.SetBorder(false)

		displayLimit := 50
		for i, perm := range result.EffectivePermissions {
			if i >= displayLimit {
				break
			}
			permName := ""
			if p, ok := perm["permission"].(string); ok {
				permName = p
			} else if p, ok := perm["name"].(string); ok {
				permName = p
			}
			effect := "ALLOW"
			if e, ok := perm["effect"].(string); ok {
				effect = e
			}
			service := "-"
			if s, ok := perm["service"].(string); ok {
				service = s
			}

			table.Append([]string{permName, effect, service})
		}

		table.Render()

		if len(result.EffectivePermissions) > displayLimit {
			fmt.Printf("\nShowing %d of %d permissions. Use --export for full list.\n", displayLimit, len(result.EffectivePermissions))
		}

		return nil
	},
}

func init() {
	effectiveGroupCmd.Flags().StringP("level", "l", "account", "Level type: account, environment, global")
	effectiveGroupCmd.Flags().String("level-id", "", "Level ID (uses account UUID if not specified)")
	effectiveGroupCmd.Flags().StringP("services", "s", "", "Comma-separated service filter")
	effectiveGroupCmd.Flags().StringP("export", "e", "", "Export to file")
}
