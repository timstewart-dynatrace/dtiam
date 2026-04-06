// Package analyze provides analysis commands for IAM resources.
package analyze

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

// Cmd is the analyze command.
var Cmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze IAM permissions",
	Long:  "Commands for analyzing IAM permissions, effective permissions, and policy compliance.",
}

func init() {
	Cmd.AddCommand(userPermissionsCmd)
	Cmd.AddCommand(groupPermissionsCmd)
	Cmd.AddCommand(permissionsMatrixCmd)
	Cmd.AddCommand(policyCmd)
	Cmd.AddCommand(leastPrivilegeCmd)
	Cmd.AddCommand(effectiveUserCmd)
	Cmd.AddCommand(effectiveGroupCmd)
}

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

var permissionsMatrixCmd = &cobra.Command{
	Use:   "permissions-matrix",
	Short: "Generate a permissions matrix",
	Long: `Generate a permissions matrix.

Shows which permissions are granted by each policy or group.
Useful for security audits and compliance reviews.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		exportFile, _ := cmd.Flags().GetString("export")

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		matrixGen := utils.NewPermissionsMatrix(c)
		ctx := context.Background()

		var result *utils.MatrixResult
		var nameField string

		if scope == "groups" {
			result, err = matrixGen.GenerateGroupMatrix(ctx)
			nameField = "group_name"
		} else {
			result, err = matrixGen.GeneratePolicyMatrix(ctx)
			nameField = "policy_name"
		}

		if err != nil {
			return err
		}

		// Export to CSV
		if exportFile != "" {
			file, err := os.Create(exportFile)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer file.Close()

			writer := csv.NewWriter(file)
			defer writer.Flush()

			if len(result.Matrix) > 0 {
				// Get fields from first row
				var fields []string
				fields = append(fields, nameField)
				for _, perm := range result.Permissions {
					fields = append(fields, perm)
				}

				if err := writer.Write(fields); err != nil {
					return err
				}

				for _, row := range result.Matrix {
					var values []string
					values = append(values, fmt.Sprintf("%v", row[nameField]))
					for _, perm := range result.Permissions {
						if row[perm] == true {
							values = append(values, "true")
						} else {
							values = append(values, "false")
						}
					}
					if err := writer.Write(values); err != nil {
						return err
					}
				}
			}

			fmt.Printf("Exported matrix to %s\n", exportFile)
			return nil
		}

		// Check output format
		format := cli.GlobalState.GetOutputFormat()
		if format == output.FormatJSON || format == output.FormatYAML {
			printer := cli.GlobalState.NewPrinter()
			return printer.PrintAny(result)
		}

		// Table output
		fmt.Println()
		fmt.Printf("=== Permissions Matrix (%s) ===\n", scope)
		if scope == "groups" {
			fmt.Printf("Total groups: %d\n", result.GroupCount)
		} else {
			fmt.Printf("Total policies: %d\n", result.PolicyCount)
		}
		fmt.Printf("Unique permissions: %d\n", result.PermissionCount)
		fmt.Println()

		if len(result.Matrix) == 0 {
			fmt.Println("No data found.")
			return nil
		}

		// Create table with limited columns for readability
		table := tablewriter.NewWriter(os.Stdout)
		headers := []string{"Name"}

		// Limit to first 5 permissions
		displayPerms := result.Permissions
		if len(displayPerms) > 5 {
			displayPerms = displayPerms[:5]
		}
		for _, perm := range displayPerms {
			// Shorten permission name
			short := perm
			if strings.Contains(perm, ":") {
				parts := strings.Split(perm, ":")
				short = parts[len(parts)-1]
			}
			if len(short) > 15 {
				short = short[:15]
			}
			headers = append(headers, short)
		}
		if len(result.Permissions) > 5 {
			headers = append(headers, "...")
		}
		table.SetHeader(headers)
		table.SetBorder(false)

		// Limit rows to 20
		displayRows := result.Matrix
		if len(displayRows) > 20 {
			displayRows = displayRows[:20]
		}

		for _, row := range displayRows {
			cells := []string{fmt.Sprintf("%v", row[nameField])}
			for _, perm := range displayPerms {
				if row[perm] == true {
					cells = append(cells, "✓")
				} else {
					cells = append(cells, "")
				}
			}
			if len(result.Permissions) > 5 {
				cells = append(cells, "")
			}
			table.Append(cells)
		}

		if len(result.Matrix) > 20 {
			fmt.Printf("(Showing first 20 of %d rows)\n", len(result.Matrix))
		}

		table.Render()
		fmt.Println()
		fmt.Println("Use --export to get full matrix as CSV")

		return nil
	},
}

func init() {
	permissionsMatrixCmd.Flags().StringP("scope", "s", "policies", "Scope: policies or groups")
	permissionsMatrixCmd.Flags().StringP("export", "e", "", "Export to CSV file")
}

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

var leastPrivilegeCmd = &cobra.Command{
	Use:   "least-privilege",
	Short: "Analyze policies for least-privilege compliance",
	Long:  `Analyze policies for least-privilege compliance. Identifies policies that may grant excessive permissions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		exportFile, _ := cmd.Flags().GetString("export")

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		policyHandler := resources.NewPolicyHandler(c)
		ctx := context.Background()

		// Broad permission patterns that may indicate over-permissioning
		broadPatterns := []struct {
			pattern     string
			description string
			severity    string
		}{
			{"*", "Wildcard permission", "high"},
			{":*", "Resource wildcard", "medium"},
			{"write", "Write access", "medium"},
			{"manage", "Management access", "medium"},
			{"delete", "Delete capability", "medium"},
			{"admin", "Admin access", "high"},
		}

		policies, err := policyHandler.List(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to list policies: %w", err)
		}

		var findings []map[string]any

		for _, policy := range policies {
			policyUUID := utils.StringFrom(policy, "uuid")
			policyName := utils.StringFrom(policy, "name")

			policyDetail, err := policyHandler.Get(ctx, policyUUID)
			if err != nil || policyDetail == nil {
				continue
			}

			statement := ""
			if s, ok := policyDetail["statementQuery"].(string); ok {
				statement = s
			}

			var policyFindings []map[string]any

			// Check for broad patterns
			for _, bp := range broadPatterns {
				if strings.Contains(strings.ToLower(statement), bp.pattern) {
					policyFindings = append(policyFindings, map[string]any{
						"type":        "broad_permission",
						"pattern":     bp.pattern,
						"description": bp.description,
						"severity":    bp.severity,
					})
				}
			}

			// Check for no conditions (unrestricted)
			permissions := utils.ParseStatementQuery(statement)
			unrestricted := 0
			for _, perm := range permissions {
				if perm.Conditions == "" {
					unrestricted++
				}
			}
			if unrestricted > 0 && unrestricted == len(permissions) {
				policyFindings = append(policyFindings, map[string]any{
					"type":        "no_conditions",
					"description": "All permissions lack conditions/restrictions",
					"severity":    "medium",
				})
			}

			if len(policyFindings) > 0 {
				findings = append(findings, map[string]any{
					"policy_uuid":   policyUUID,
					"policy_name":   policyName,
					"findings":      policyFindings,
					"finding_count": len(policyFindings),
				})
			}
		}

		result := map[string]any{
			"total_policies":         len(policies),
			"policies_with_findings": len(findings),
			"findings":               findings,
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
			fmt.Printf("Exported findings to %s\n", exportFile)
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
		fmt.Println("=== Least-Privilege Analysis ===")
		fmt.Printf("Policies analyzed: %d\n", len(policies))
		fmt.Printf("Policies with findings: %d\n", len(findings))
		fmt.Println()

		if len(findings) == 0 {
			fmt.Println("No issues found.")
			return nil
		}

		for _, policyFinding := range findings {
			fmt.Printf("%s\n", policyFinding["policy_name"])
			for _, finding := range policyFinding["findings"].([]map[string]any) {
				severity := utils.StringFrom(finding, "severity")
				fmt.Printf("  [%s] %s\n", strings.ToUpper(severity), finding["description"])
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	leastPrivilegeCmd.Flags().StringP("export", "e", "", "Export findings to file")
}

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
