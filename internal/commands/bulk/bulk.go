// Package bulk provides bulk operations for IAM resources.
package bulk

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/prompt"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/internal/utils"
)

// Cmd is the bulk command.
var Cmd = &cobra.Command{
	Use:   "bulk",
	Short: "Bulk operations",
	Long:  "Commands for bulk IAM operations from files.",
}

func init() {
	Cmd.AddCommand(addUsersToGroupCmd)
	Cmd.AddCommand(removeUsersFromGroupCmd)
	Cmd.AddCommand(createGroupsCmd)
	Cmd.AddCommand(createBindingsCmd)
	Cmd.AddCommand(exportGroupMembersCmd)
}

// loadInputFile loads data from a JSON, YAML, or CSV file.
func loadInputFile(filePath string) ([]map[string]string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	switch ext {
	case ".json":
		var data []map[string]string
		// Try as array first
		if err := json.Unmarshal(content, &data); err != nil {
			// Try as single object
			var single map[string]string
			if err := json.Unmarshal(content, &single); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %w", err)
			}
			data = []map[string]string{single}
		}
		return data, nil

	case ".yaml", ".yml":
		var data []map[string]string
		// Try as array first
		if err := yaml.Unmarshal(content, &data); err != nil {
			// Try as single object
			var single map[string]string
			if err := yaml.Unmarshal(content, &single); err != nil {
				return nil, fmt.Errorf("failed to parse YAML: %w", err)
			}
			data = []map[string]string{single}
		}
		return data, nil

	case ".csv":
		reader := csv.NewReader(strings.NewReader(string(content)))
		// Skip comments
		reader.Comment = '#'
		records, err := reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV: %w", err)
		}
		if len(records) < 2 {
			return nil, fmt.Errorf("CSV file must have a header row and at least one data row")
		}
		// First row is header
		headers := records[0]
		var data []map[string]string
		for _, row := range records[1:] {
			record := make(map[string]string)
			for i, value := range row {
				if i < len(headers) {
					record[headers[i]] = value
				}
			}
			data = append(data, record)
		}
		return data, nil

	default:
		return nil, fmt.Errorf("unsupported file format: %s (use .json, .yaml, .yml, or .csv)", ext)
	}
}

// loadYAMLFile loads data from a YAML file with nested structure.
func loadYAMLFile(filePath string, key string) ([]map[string]any, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var wrapper map[string][]map[string]any
	if err := yaml.Unmarshal(content, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	data, ok := wrapper[key]
	if !ok {
		// Try without wrapper
		var direct []map[string]any
		if err := yaml.Unmarshal(content, &direct); err != nil {
			return nil, fmt.Errorf("YAML must have '%s' key or be a list", key)
		}
		return direct, nil
	}

	return data, nil
}

var addUsersToGroupCmd = &cobra.Command{
	Use:   "add-users-to-group",
	Short: "Add multiple users to a group from a file",
	Long: `Add multiple users to a group from a file.

The file can be JSON, YAML, or CSV format.

JSON/YAML example:
  [{"email": "user1@example.com"}, {"email": "user2@example.com"}]

CSV example:
  email
  user1@example.com
  user2@example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		groupID, _ := cmd.Flags().GetString("group")
		emailField, _ := cmd.Flags().GetString("email-field")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		if filePath == "" {
			return fmt.Errorf("--file is required")
		}
		if groupID == "" {
			return fmt.Errorf("--group is required")
		}

		// Load file
		records, err := loadInputFile(filePath)
		if err != nil {
			return err
		}

		if len(records) == 0 {
			fmt.Println("Warning: No records found in file.")
			return nil
		}

		// Dry run check
		if cli.GlobalState.IsDryRun() {
			printer := cli.GlobalState.NewPrinter()
			printer.PrintWarning("Would add %d users to group '%s'", len(records), groupID)
			for _, record := range records {
				if email := record[emailField]; email != "" {
					fmt.Fprintf(os.Stderr, "  - %s\n", email)
				}
			}
			return nil
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

		fmt.Printf("Adding users to group '%s' (%s)...\n", groupName, groupUUID)

		// Process additions
		var successCount, failCount int
		for _, record := range records {
			email := strings.TrimSpace(record[emailField])
			if email == "" {
				fmt.Printf("  Warning: Record missing '%s' field\n", emailField)
				continue
			}

			err := handler.AddMember(ctx, groupUUID, email)
			if err != nil {
				failCount++
				fmt.Printf("  Failed to add '%s': %v\n", email, err)
				if !continueOnError {
					return err
				}
			} else {
				successCount++
				if cli.GlobalState.IsVerbose() {
					fmt.Printf("  Added: %s\n", email)
				}
			}
		}

		fmt.Printf("\nSuccessfully added: %d users\n", successCount)
		if failCount > 0 {
			fmt.Printf("Failed: %d users\n", failCount)
		}

		return nil
	},
}

func init() {
	addUsersToGroupCmd.Flags().StringP("file", "f", "", "File with user emails (JSON, YAML, or CSV)")
	addUsersToGroupCmd.Flags().StringP("group", "g", "", "Group UUID or name")
	addUsersToGroupCmd.Flags().StringP("email-field", "e", "email", "Field name containing email addresses")
	addUsersToGroupCmd.Flags().Bool("continue-on-error", false, "Continue processing on errors")
}

var removeUsersFromGroupCmd = &cobra.Command{
	Use:   "remove-users-from-group",
	Short: "Remove multiple users from a group from a file",
	Long: `Remove multiple users from a group from a file.

The file can be JSON, YAML, or CSV format containing email addresses or user UIDs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		groupID, _ := cmd.Flags().GetString("group")
		userField, _ := cmd.Flags().GetString("user-field")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")
		force, _ := cmd.Flags().GetBool("force")

		if filePath == "" {
			return fmt.Errorf("--file is required")
		}
		if groupID == "" {
			return fmt.Errorf("--group is required")
		}

		// Load file
		records, err := loadInputFile(filePath)
		if err != nil {
			return err
		}

		if len(records) == 0 {
			fmt.Println("Warning: No records found in file.")
			return nil
		}

		// Dry run check
		if cli.GlobalState.IsDryRun() {
			printer := cli.GlobalState.NewPrinter()
			printer.PrintWarning("Would remove %d users from group '%s'", len(records), groupID)
			for _, record := range records {
				if user := record[userField]; user != "" {
					fmt.Fprintf(os.Stderr, "  - %s\n", user)
				}
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		groupHandler := resources.NewGroupHandler(c)
		userHandler := resources.NewUserHandler(c)
		ctx := context.Background()

		// Resolve group
		group, err := groupHandler.Resolve(ctx, groupID)
		if err != nil {
			return fmt.Errorf("group not found: %s", groupID)
		}
		groupUUID := utils.StringFrom(group, "uuid")
		groupName := utils.StringFrom(group, "name")

		// Resolve users to UIDs
		type userToRemove struct {
			uid     string
			display string
		}
		var usersToRemove []userToRemove

		for _, record := range records {
			userID := strings.TrimSpace(record[userField])
			if userID == "" {
				fmt.Printf("  Warning: Record missing '%s' field\n", userField)
				continue
			}

			// If it looks like email, resolve to UID
			if strings.Contains(userID, "@") {
				user, err := userHandler.GetByEmail(ctx, userID)
				if err != nil || user == nil {
					fmt.Printf("  Warning: User not found: %s\n", userID)
					continue
				}
				usersToRemove = append(usersToRemove, userToRemove{
					uid:     utils.StringFrom(user, "uid"),
					display: userID,
				})
			} else {
				usersToRemove = append(usersToRemove, userToRemove{
					uid:     userID,
					display: userID,
				})
			}
		}

		if len(usersToRemove) == 0 {
			return fmt.Errorf("no valid users found")
		}

		fmt.Printf("Found %d users to remove from group '%s'\n", len(usersToRemove), groupName)

		// Confirmation
		if !prompt.Confirm(
			fmt.Sprintf("Remove %d users from group '%s'?", len(usersToRemove), groupName),
			force || cli.GlobalState.IsPlain(),
		) {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}

		// Process removals
		var successCount, failCount int
		for _, user := range usersToRemove {
			err := groupHandler.RemoveMember(ctx, groupUUID, user.uid)
			if err != nil {
				failCount++
				fmt.Printf("  Failed to remove '%s': %v\n", user.display, err)
				if !continueOnError {
					return err
				}
			} else {
				successCount++
				if cli.GlobalState.IsVerbose() {
					fmt.Printf("  Removed: %s\n", user.display)
				}
			}
		}

		fmt.Printf("\nSuccessfully removed: %d users\n", successCount)
		if failCount > 0 {
			fmt.Printf("Failed: %d users\n", failCount)
		}

		return nil
	},
}

func init() {
	removeUsersFromGroupCmd.Flags().StringP("file", "f", "", "File with user emails/UIDs (JSON, YAML, or CSV)")
	removeUsersFromGroupCmd.Flags().StringP("group", "g", "", "Group UUID or name")
	removeUsersFromGroupCmd.Flags().StringP("user-field", "u", "email", "Field name containing email or UID")
	removeUsersFromGroupCmd.Flags().Bool("continue-on-error", false, "Continue processing on errors")
	removeUsersFromGroupCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

var createGroupsCmd = &cobra.Command{
	Use:   "create-groups",
	Short: "Create multiple groups from a file",
	Long: `Create multiple groups from a file.

JSON/YAML example:
  groups:
    - name: "Group A"
      description: "Description for Group A"
    - name: "Group B"
      description: "Description for Group B"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		if filePath == "" {
			return fmt.Errorf("--file is required")
		}

		// Load YAML file
		records, err := loadYAMLFile(filePath, "groups")
		if err != nil {
			return err
		}

		if len(records) == 0 {
			fmt.Println("Warning: No records found in file.")
			return nil
		}

		// Validate records have name
		var validGroups []map[string]any
		for _, record := range records {
			if _, ok := record["name"]; !ok {
				fmt.Printf("Warning: Record missing 'name' field: %v\n", record)
				continue
			}
			validGroups = append(validGroups, record)
		}

		if len(validGroups) == 0 {
			return fmt.Errorf("no valid group definitions found")
		}

		fmt.Printf("Found %d groups to create\n", len(validGroups))

		// Dry run check
		if cli.GlobalState.IsDryRun() {
			fmt.Println("Would create the following groups:")
			for _, group := range validGroups {
				fmt.Printf("  - %s\n", group["name"])
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewGroupHandler(c)
		ctx := context.Background()

		// Process creations
		var successCount, failCount int
		for _, groupDef := range validGroups {
			name := utils.StringFrom(groupDef, "name")
			data := map[string]any{
				"name": name,
			}
			if desc, ok := groupDef["description"].(string); ok && desc != "" {
				data["description"] = desc
			}

			_, err := handler.Create(ctx, data)
			if err != nil {
				failCount++
				fmt.Printf("  Failed to create '%s': %v\n", name, err)
				if !continueOnError {
					return err
				}
			} else {
				successCount++
				if cli.GlobalState.IsVerbose() {
					fmt.Printf("  Created: %s\n", name)
				}
			}
		}

		fmt.Printf("\nSuccessfully created: %d groups\n", successCount)
		if failCount > 0 {
			fmt.Printf("Failed: %d groups\n", failCount)
		}

		return nil
	},
}

func init() {
	createGroupsCmd.Flags().StringP("file", "f", "", "File with group definitions (JSON or YAML)")
	createGroupsCmd.Flags().Bool("continue-on-error", false, "Continue processing on errors")
}

var createBindingsCmd = &cobra.Command{
	Use:   "create-bindings",
	Short: "Create multiple policy bindings from a file",
	Long: `Create multiple policy bindings from a file.

JSON/YAML example:
  bindings:
    - group: "group-uuid-or-name"
      policy: "policy-uuid-or-name"
      boundary: "optional-boundary-uuid"  # optional
    - group: "another-group"
      policy: "another-policy"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		if filePath == "" {
			return fmt.Errorf("--file is required")
		}

		// Load YAML file
		records, err := loadYAMLFile(filePath, "bindings")
		if err != nil {
			return err
		}

		if len(records) == 0 {
			fmt.Println("Warning: No records found in file.")
			return nil
		}

		// Validate records have group and policy
		var validBindings []map[string]any
		for _, record := range records {
			if _, ok := record["group"]; !ok {
				fmt.Printf("Warning: Record missing 'group' field: %v\n", record)
				continue
			}
			if _, ok := record["policy"]; !ok {
				fmt.Printf("Warning: Record missing 'policy' field: %v\n", record)
				continue
			}
			validBindings = append(validBindings, record)
		}

		if len(validBindings) == 0 {
			return fmt.Errorf("no valid binding definitions found")
		}

		fmt.Printf("Found %d bindings to create\n", len(validBindings))

		// Dry run check
		if cli.GlobalState.IsDryRun() {
			fmt.Println("Would create the following bindings:")
			for _, binding := range validBindings {
				fmt.Printf("  - Group: %s -> Policy: %s\n", binding["group"], binding["policy"])
			}
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		bindingHandler := resources.NewBindingHandler(c)
		groupHandler := resources.NewGroupHandler(c)
		policyHandler := resources.NewPolicyHandler(c)
		ctx := context.Background()

		// Process creations
		var successCount, failCount int
		for _, bindingDef := range validBindings {
			groupID := utils.StringFrom(bindingDef, "group")
			policyID := utils.StringFrom(bindingDef, "policy")

			// Resolve group
			group, err := groupHandler.Resolve(ctx, groupID)
			if err != nil {
				failCount++
				fmt.Printf("  Failed: Group not found: %s\n", groupID)
				if !continueOnError {
					return fmt.Errorf("group not found: %s", groupID)
				}
				continue
			}
			groupUUID := utils.StringFrom(group, "uuid")

			// Resolve policy
			policy, err := policyHandler.Resolve(ctx, policyID)
			if err != nil {
				failCount++
				fmt.Printf("  Failed: Policy not found: %s\n", policyID)
				if !continueOnError {
					return fmt.Errorf("policy not found: %s", policyID)
				}
				continue
			}
			policyUUID := utils.StringFrom(policy, "uuid")

			// Optional boundary
			var boundaries []string
			if boundaryID, ok := bindingDef["boundary"].(string); ok && boundaryID != "" {
				boundaries = []string{boundaryID}
			}

			_, err = bindingHandler.Create(ctx, groupUUID, policyUUID, boundaries, nil)
			if err != nil {
				failCount++
				fmt.Printf("  Failed to create binding %s -> %s: %v\n", groupID, policyID, err)
				if !continueOnError {
					return err
				}
			} else {
				successCount++
				if cli.GlobalState.IsVerbose() {
					fmt.Printf("  Created: %s -> %s\n", groupID, policyID)
				}
			}
		}

		fmt.Printf("\nSuccessfully created: %d bindings\n", successCount)
		if failCount > 0 {
			fmt.Printf("Failed: %d bindings\n", failCount)
		}

		return nil
	},
}

func init() {
	createBindingsCmd.Flags().StringP("file", "f", "", "File with binding definitions (JSON or YAML)")
	createBindingsCmd.Flags().Bool("continue-on-error", false, "Continue processing on errors")
}

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
