package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/utils"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Export all IAM resources to files",
	Long: `Export all IAM resources to files.

Exports environments, groups, users, policies, bindings, and boundaries.
With --detailed flag, includes enriched data like user counts and memberships.

Examples:
  dtiam export all                          # Export all to CSV in current dir
  dtiam export all -o ./backup -f json      # Export as JSON to backup dir
  dtiam export all --detailed               # Include enriched data
  dtiam export all -i groups,policies       # Only export groups and policies`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		prefix, _ := cmd.Flags().GetString("prefix")
		include, _ := cmd.Flags().GetString("include")
		detailed, _ := cmd.Flags().GetBool("detailed")
		timestampDir, _ := cmd.Flags().GetBool("timestamp-dir")

		// Determine which exports to run
		allExports := []string{"environments", "groups", "users", "policies", "bindings", "boundaries"}
		var exportsToRun []string

		if include != "" {
			requested := make(map[string]bool)
			for _, e := range splitAndTrim(include, ",") {
				requested[e] = true
			}
			for _, e := range allExports {
				if requested[e] {
					exportsToRun = append(exportsToRun, e)
				}
			}
		} else {
			exportsToRun = allExports
		}

		if len(exportsToRun) == 0 {
			return fmt.Errorf("no valid exports specified")
		}

		// Create output directory
		exportDir := outputDir
		if timestampDir {
			timestamp := time.Now().Format("20060102_150405")
			exportDir = filepath.Join(outputDir, fmt.Sprintf("%s_export_%s", prefix, timestamp))
		}

		if err := os.MkdirAll(exportDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		ctx := context.Background()

		// File extension
		ext := format
		if ext == "csv" {
			ext = "csv"
		}

		type exportResult struct {
			resource string
			count    int
			path     string
		}
		var exportedFiles []exportResult

		// Export each resource type
		for _, resource := range exportsToRun {
			fmt.Printf("Exporting %s...\n", resource)

			var data []map[string]any
			var exportErr error

			switch resource {
			case "environments":
				handler := resources.NewEnvironmentHandler(c)
				data, exportErr = handler.List(ctx, nil)

			case "groups":
				handler := resources.NewGroupHandler(c)
				data, exportErr = handler.List(ctx, nil)

				if detailed && exportErr == nil {
					// Enrich with member counts
					for i, group := range data {
						groupID := utils.StringFrom(group, "uuid")
						members, err := handler.GetMembers(ctx, groupID)
						if err == nil {
							data[i]["member_count"] = len(members)
							var emails []string
							for _, m := range members {
								if email, ok := m["email"].(string); ok {
									emails = append(emails, email)
								}
							}
							data[i]["member_emails"] = emails
						}
					}
				}

			case "users":
				handler := resources.NewUserHandler(c)
				data, exportErr = handler.List(ctx, nil)

				if detailed && exportErr == nil {
					// Enrich with group memberships
					for i, user := range data {
						userID := utils.StringFrom(user, "uid")
						groups, err := handler.GetGroups(ctx, userID)
						if err == nil {
							data[i]["group_count"] = len(groups)
							var names []string
							for _, g := range groups {
								if name, ok := g["name"].(string); ok {
									names = append(names, name)
								}
							}
							data[i]["group_names"] = names
						}
					}
				}

			case "policies":
				handler := resources.NewPolicyHandler(c)
				data, exportErr = handler.List(ctx, nil)

				if detailed && exportErr == nil {
					// Get full policy details
					var detailedData []map[string]any
					for _, policy := range data {
						policyID := utils.StringFrom(policy, "uuid")
						detail, err := handler.Get(ctx, policyID)
						if err == nil && detail != nil {
							detailedData = append(detailedData, detail)
						} else {
							detailedData = append(detailedData, policy)
						}
					}
					data = detailedData
				}

			case "bindings":
				handler := resources.NewBindingHandler(c)
				data, exportErr = handler.List(ctx, nil)

				if detailed && exportErr == nil {
					// Enrich with group and policy names
					groupHandler := resources.NewGroupHandler(c)
					policyHandler := resources.NewPolicyHandler(c)

					for i, binding := range data {
						if groupUUID, ok := binding["groupUuid"].(string); ok {
							group, err := groupHandler.Get(ctx, groupUUID)
							if err == nil && group != nil {
								data[i]["group_name"] = group["name"]
							}
						}
						if policyUUID, ok := binding["policyUuid"].(string); ok {
							policy, err := policyHandler.Get(ctx, policyUUID)
							if err == nil && policy != nil {
								data[i]["policy_name"] = policy["name"]
							}
						}
					}
				}

			case "boundaries":
				handler := resources.NewBoundaryHandler(c)
				data, exportErr = handler.List(ctx, nil)

				if detailed && exportErr == nil {
					// Get full boundary details
					var detailedData []map[string]any
					for _, boundary := range data {
						boundaryID := utils.StringFrom(boundary, "uuid")
						detail, err := handler.Get(ctx, boundaryID)
						if err == nil && detail != nil {
							// Add attached policies
							attached, err := handler.GetAttachedPolicies(ctx, boundaryID)
							if err == nil {
								detail["attached_policies"] = attached
								detail["attached_policy_count"] = len(attached)
							}
							detailedData = append(detailedData, detail)
						} else {
							detailedData = append(detailedData, boundary)
						}
					}
					data = detailedData
				}
			}

			if exportErr != nil {
				fmt.Printf("  Warning: Failed to export %s: %v\n", resource, exportErr)
				continue
			}

			filePath := filepath.Join(exportDir, fmt.Sprintf("%s_%s.%s", prefix, resource, ext))
			if err := writeData(data, filePath, format); err != nil {
				fmt.Printf("  Warning: Failed to write %s: %v\n", resource, err)
				continue
			}

			exportedFiles = append(exportedFiles, exportResult{
				resource: resource,
				count:    len(data),
				path:     filePath,
			})
		}

		// Summary
		fmt.Println()
		fmt.Println("Export complete!")
		fmt.Printf("Output directory: %s\n", exportDir)
		fmt.Println()

		for _, result := range exportedFiles {
			fmt.Printf("  %s: %d records -> %s\n", result.resource, result.count, filepath.Base(result.path))
		}

		return nil
	},
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func init() {
	allCmd.Flags().StringP("output", "o", ".", "Output directory")
	allCmd.Flags().StringP("format", "f", "csv", "Output format (csv, json, yaml)")
	allCmd.Flags().StringP("prefix", "p", "dtiam", "File name prefix")
	allCmd.Flags().StringP("include", "i", "", "Comma-separated list of exports to include")
	allCmd.Flags().BoolP("detailed", "d", false, "Include detailed/enriched data")
	allCmd.Flags().Bool("timestamp-dir", true, "Create timestamped subdirectory")
}
