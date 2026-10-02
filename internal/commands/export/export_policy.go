package export

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/utils"
)

var policyCmd = &cobra.Command{
	Use:   "policy IDENTIFIER",
	Short: "Export a single policy with its details",
	Long: `Export a single policy with its details.

With --as-template, exports in template format with variable placeholders.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		asTemplate, _ := cmd.Flags().GetBool("as-template")

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewPolicyHandler(c)
		ctx := context.Background()

		// Resolve policy
		policy, err := handler.Resolve(ctx, identifier)
		if err != nil {
			return fmt.Errorf("policy not found: %s", identifier)
		}

		policyName := utils.StringFrom(policy, "name")

		var exportData map[string]any

		if asTemplate {
			// Export as Go text/template format compatible with dtiam template apply
			exportData = map[string]any{
				"kind": "Policy",
				"spec": map[string]any{
					"name":           "{{.name}}",
					"description":    fmt.Sprintf("{{.description | default \"%s\"}}", policyName),
					"statementQuery": policy["statementQuery"],
				},
			}
		} else {
			exportData = map[string]any{
				"apiVersion": "v1",
				"kind":       "Policy",
				"metadata": map[string]any{
					"uuid":       policy["uuid"],
					"exportedAt": time.Now().Format(time.RFC3339),
				},
				"spec": map[string]any{
					"name":           policyName,
					"description":    policy["description"],
					"statementQuery": policy["statementQuery"],
				},
			}
		}

		// Format output
		var output string
		switch format {
		case "json":
			data, err := json.MarshalIndent(exportData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			output = string(data)
		default:
			data, err := yaml.Marshal(exportData)
			if err != nil {
				return fmt.Errorf("failed to marshal YAML: %w", err)
			}
			output = string(data)
		}

		// Write or print output
		if outputFile != "" {
			if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			fmt.Printf("Exported policy '%s' to %s\n", policyName, outputFile)
		} else {
			fmt.Println(output)
		}

		return nil
	},
}

func init() {
	policyCmd.Flags().StringP("output", "o", "", "Output file")
	policyCmd.Flags().StringP("format", "f", "yaml", "Output format (yaml, json)")
	policyCmd.Flags().BoolP("as-template", "t", false, "Export as reusable template")
}

// exportResourceToDir exports a list of resources to a file in the given directory.
