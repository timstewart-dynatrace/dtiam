package analyze

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

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
