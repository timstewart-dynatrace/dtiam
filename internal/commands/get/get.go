// Package get provides commands for listing and retrieving resources.
package get

import (
	"github.com/spf13/cobra"
)

// Cmd is the get command.
var Cmd = &cobra.Command{
	Use:   "get",
	Short: "List or retrieve resources",
	Long:  "Commands for listing and retrieving IAM resources.",
}

func init() {
	Cmd.AddCommand(groupsCmd)
	Cmd.AddCommand(usersCmd)
	Cmd.AddCommand(policiesCmd)
	Cmd.AddCommand(bindingsCmd)
	Cmd.AddCommand(environmentsCmd)
	Cmd.AddCommand(boundariesCmd)
	Cmd.AddCommand(tokensCmd)
	Cmd.AddCommand(appsCmd)
	Cmd.AddCommand(schemasCmd)
	Cmd.AddCommand(auditLogsCmd)
	Cmd.AddCommand(availablePermissionsCmd)
	Cmd.AddCommand(envUsersCmd)
	Cmd.AddCommand(envGroupsCmd)
}
