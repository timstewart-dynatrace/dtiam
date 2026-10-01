package bulk

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/prompt"
	"github.com/jtimothystewart/dtiam/pkg/resources"
	"github.com/jtimothystewart/dtiam/pkg/utils"
)

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
