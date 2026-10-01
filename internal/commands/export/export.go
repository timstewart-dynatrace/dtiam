// Package export provides comprehensive export commands for IAM resources.
package export

import (
	"github.com/spf13/cobra"
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
	Cmd.AddCommand(environmentsCmd)
	Cmd.AddCommand(usersCmd)
	Cmd.AddCommand(bindingsCmd)
	Cmd.AddCommand(boundariesCmd)
	Cmd.AddCommand(serviceUsersCmd)
}

// writeData writes data to a file in the specified format.
