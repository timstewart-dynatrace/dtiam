// Package bulk provides bulk operations for IAM resources.
package bulk

import (
	"github.com/spf13/cobra"
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
	Cmd.AddCommand(createGroupsWithPoliciesCmd)
}

// loadInputFile loads data from a JSON, YAML, or CSV file.
