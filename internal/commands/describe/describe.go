// Package describe provides commands for displaying detailed resource information.
package describe

import (
	"github.com/spf13/cobra"
)

// Cmd is the describe command.
var Cmd = &cobra.Command{
	Use:   "describe",
	Short: "Show detailed information about a resource",
	Long: `Display detailed information about a specific IAM resource.

Unlike 'get' which lists resources in a table, 'describe' shows all fields
for a single resource including nested objects and metadata. The IDENTIFIER
can be a UUID or a human-readable name (or email for users).`,
}

func init() {
	Cmd.AddCommand(groupCmd)
	Cmd.AddCommand(userCmd)
	Cmd.AddCommand(policyCmd)
	Cmd.AddCommand(environmentCmd)
	Cmd.AddCommand(boundaryCmd)
	Cmd.AddCommand(serviceUserCmd)
}
