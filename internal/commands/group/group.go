// Package group provides advanced group management commands.
package group

import (
	"github.com/spf13/cobra"
)

// Cmd is the group command.
var Cmd = &cobra.Command{
	Use:   "group",
	Short: "Advanced group management commands",
	Long: `Commands for advanced group operations such as listing members,
adding and removing members, and viewing policy bindings for a group.

Groups can be identified by UUID or name in all subcommands.`,
	Example: `  # List members of a group
  dtiam group members "Production Team"

  # Add a user to a group
  dtiam group add-member "Production Team" --email user@example.com

  # View policy bindings for a group
  dtiam group bindings "Production Team"`,
}

func init() {
	Cmd.AddCommand(membersCmd)
	Cmd.AddCommand(addMemberCmd)
	Cmd.AddCommand(removeMemberCmd)
	Cmd.AddCommand(bindingsCmd)
	Cmd.AddCommand(cloneCmd)
	Cmd.AddCommand(setupCmd)
	Cmd.AddCommand(permissionsCmd)
	Cmd.AddCommand(grantPermissionCmd)
	Cmd.AddCommand(revokePermissionCmd)
	Cmd.AddCommand(updateCmd)
}
