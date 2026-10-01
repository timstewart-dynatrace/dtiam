// Package create provides commands for creating resources.
package create

import (
	"github.com/spf13/cobra"
)

// Cmd is the create command.
var Cmd = &cobra.Command{
	Use:   "create",
	Short: "Create a resource",
	Long:  "Commands for creating IAM resources.",
}

func init() {
	Cmd.AddCommand(groupCmd)
	Cmd.AddCommand(policyCmd)
	Cmd.AddCommand(bindingCmd)
	Cmd.AddCommand(boundaryCmd)
	Cmd.AddCommand(tokenCmd)
}
