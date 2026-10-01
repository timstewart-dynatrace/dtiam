package bulk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/resources"
	"github.com/jtimothystewart/dtiam/pkg/utils"
)

var exportGroupMembersCmd = &cobra.Command{
	Use:   "export-group-members",
	Short: "Export group members to a file",
	Long:  `Export group members to a file. Useful for backups or migration purposes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group")
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")

		if groupID == "" {
			return fmt.Errorf("--group is required")
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		ctx := context.Background()

		// Resolve group
		group, err := handler.Resolve(ctx, groupID)
		if err != nil {
			return fmt.Errorf("group not found: %s", groupID)
		}
		groupUUID := utils.StringFrom(group, "uuid")
		groupName := utils.StringFrom(group, "name")

		// Get members
		members, err := handler.GetMembers(ctx, groupUUID)
		if err != nil {
			return fmt.Errorf("failed to get members: %w", err)
		}

		if len(members) == 0 {
			fmt.Printf("Group '%s' has no members.\n", groupName)
			return nil
		}

		// Format output
		var output string
		switch format {
		case "json":
			data, err := json.MarshalIndent(members, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			output = string(data)
		case "yaml":
			data, err := yaml.Marshal(members)
			if err != nil {
				return fmt.Errorf("failed to marshal YAML: %w", err)
			}
			output = string(data)
		case "csv":
			var sb strings.Builder
			// Get fields from first member
			if len(members) > 0 {
				var fields []string
				for k := range members[0] {
					fields = append(fields, k)
				}
				sb.WriteString(strings.Join(fields, ",") + "\n")
				for _, member := range members {
					var values []string
					for _, f := range fields {
						if v, ok := member[f]; ok {
							values = append(values, fmt.Sprintf("%v", v))
						} else {
							values = append(values, "")
						}
					}
					sb.WriteString(strings.Join(values, ",") + "\n")
				}
			}
			output = sb.String()
		default:
			return fmt.Errorf("unknown format: %s (use json, yaml, or csv)", format)
		}

		// Write or print output
		if outputFile != "" {
			if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			fmt.Printf("Exported %d members to %s\n", len(members), outputFile)
		} else {
			fmt.Println(output)
		}

		return nil
	},
}

func init() {
	exportGroupMembersCmd.Flags().StringP("group", "g", "", "Group UUID or name")
	exportGroupMembersCmd.Flags().StringP("output", "o", "", "Output file path")
	exportGroupMembersCmd.Flags().StringP("format", "F", "csv", "Output format (csv, json, yaml)")
}
