// Package delete provides commands for deleting resources.
package delete

import (
	"github.com/spf13/cobra"
)

// Cmd is the delete command.
var Cmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a resource",
	Long:  "Commands for deleting IAM resources. All delete operations require confirmation unless --force is set.",
}

func init() {
	Cmd.AddCommand(groupCmd)
	Cmd.AddCommand(policyCmd)
	Cmd.AddCommand(bindingCmd)
	Cmd.AddCommand(boundaryCmd)
	Cmd.AddCommand(userCmd)
	Cmd.AddCommand(serviceUserCmd)
	Cmd.AddCommand(tokenCmd)
}
