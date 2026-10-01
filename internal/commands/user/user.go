// Package user provides user management commands.
package user

import (
	"github.com/spf13/cobra"
)

// Cmd is the user command.
var Cmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long: `Commands for managing user operations such as group membership,
user creation, and group listing.

Use subcommands to add users to groups, remove them, replace their
group memberships, list their groups, or create new users.`,
	Example: `  # Add a user to groups
  dtiam user add-to-groups user@example.com --groups "My Group"

  # List groups a user belongs to
  dtiam user list-groups user@example.com

  # Create a new user
  dtiam user create newuser@example.com --first-name John --last-name Doe`,
}

func init() {
	Cmd.AddCommand(addToGroupsCmd)
	Cmd.AddCommand(removeFromGroupsCmd)
	Cmd.AddCommand(replaceGroupsCmd)
	Cmd.AddCommand(listGroupsCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(infoCmd)
}
