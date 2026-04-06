// Package export provides comprehensive export commands for IAM resources.
package export

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

// Cmd is the export command.
var Cmd = &cobra.Command{
	Use:   "export",
	Short: "Export IAM resources",
	Long:  "Commands for exporting IAM resources to files.",
}

func init() {
	Cmd.AddCommand(allCmd)
	Cmd.AddCommand(groupCmd)
	Cmd.AddCommand(policyCmd)
}

// writeData writes data to a file in the specified format.
func writeData(data []map[string]any, path, format string) error {
	var content []byte
	var err error

	switch format {
	case "json":
		content, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	case "yaml":
		content, err = yaml.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
	case "csv":
		if len(data) == 0 {
			return os.WriteFile(path, []byte(""), 0644)
		}

		// Flatten nested dicts for CSV
		var flatData []map[string]string
		var fields []string

		// Collect all fields
		fieldSet := make(map[string]bool)
		for _, item := range data {
			for k := range item {
				fieldSet[k] = true
			}
		}
		for f := range fieldSet {
			fields = append(fields, f)
		}

		// Flatten data
		for _, item := range data {
			flatItem := make(map[string]string)
			for k, v := range item {
				switch val := v.(type) {
				case []any, map[string]any:
					jsonBytes, _ := json.Marshal(val)
					flatItem[k] = string(jsonBytes)
				default:
					flatItem[k] = fmt.Sprintf("%v", val)
				}
			}
			flatData = append(flatData, flatItem)
		}

		// Write CSV
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Header
		if err := writer.Write(fields); err != nil {
			return err
		}

		// Rows
		for _, item := range flatData {
			var row []string
			for _, f := range fields {
				row = append(row, item[f])
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}

		return nil
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	return os.WriteFile(path, content, 0644)
}

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
			// Export as template
			exportData = map[string]any{
				"description": fmt.Sprintf("Template from policy: %s", policyName),
				"kind":        "Policy",
				"template": map[string]any{
					"name":           "{{ policy_name }}",
					"description":    "{{ description | default('') }}",
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
