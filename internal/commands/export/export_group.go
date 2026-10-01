package export

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/resources"
	"github.com/jtimothystewart/dtiam/pkg/utils"
)

var groupCmd = &cobra.Command{
	Use:   "group IDENTIFIER",
	Short: "Export a single group with its details",
	Long:  `Export a single group with its details in a format suitable for import/backup.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		includeMembers, _ := cmd.Flags().GetBool("include-members")
		includePolicies, _ := cmd.Flags().GetBool("include-policies")

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		groupHandler := resources.NewGroupHandler(c)
		bindingHandler := resources.NewBindingHandler(c)
		policyHandler := resources.NewPolicyHandler(c)
		ctx := context.Background()

		// Resolve group
		group, err := groupHandler.Resolve(ctx, identifier)
		if err != nil {
			return fmt.Errorf("group not found: %s", identifier)
		}

		groupUUID := utils.StringFrom(group, "uuid")
		groupName := utils.StringFrom(group, "name")

		exportData := map[string]any{
			"apiVersion": "v1",
			"kind":       "Group",
			"metadata": map[string]any{
				"uuid":       groupUUID,
				"exportedAt": time.Now().Format(time.RFC3339),
			},
			"spec": map[string]any{
				"name":        groupName,
				"description": group["description"],
			},
		}

		spec := exportData["spec"].(map[string]any)

		if includeMembers {
			members, err := groupHandler.GetMembers(ctx, groupUUID)
			if err == nil {
				var memberList []map[string]any
				for _, m := range members {
					memberList = append(memberList, map[string]any{
						"email": m["email"],
						"uid":   m["uid"],
					})
				}
				spec["members"] = memberList
			}
		}

		if includePolicies {
			bindings, err := bindingHandler.GetForGroup(ctx, groupUUID)
			if err == nil {
				var policyBindings []map[string]any
				for _, binding := range bindings {
					policyUUID := utils.StringFrom(binding, "policyUuid")
					policy, _ := policyHandler.Get(ctx, policyUUID)

					pb := map[string]any{
						"policyUuid": policyUUID,
					}
					if policy != nil {
						pb["policyName"] = policy["name"]
					}
					if boundary, ok := binding["boundaryUuid"]; ok {
						pb["boundaryUuid"] = boundary
					}
					policyBindings = append(policyBindings, pb)
				}
				spec["policyBindings"] = policyBindings
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
			fmt.Printf("Exported group '%s' to %s\n", groupName, outputFile)
		} else {
			fmt.Println(output)
		}

		return nil
	},
}

func init() {
	groupCmd.Flags().StringP("output", "o", "", "Output file")
	groupCmd.Flags().StringP("format", "f", "yaml", "Output format (yaml, json)")
	groupCmd.Flags().Bool("include-members", true, "Include member list")
	groupCmd.Flags().Bool("include-policies", true, "Include policy bindings")
}
