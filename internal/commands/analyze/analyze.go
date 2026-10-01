// Package analyze provides analysis commands for IAM resources.
package analyze

import (
	"github.com/spf13/cobra"
)

// Cmd is the analyze command.
var Cmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze IAM permissions",
	Long:  "Commands for analyzing IAM permissions, effective permissions, and policy compliance.",
}

func init() {
	Cmd.AddCommand(userPermissionsCmd)
	Cmd.AddCommand(groupPermissionsCmd)
	Cmd.AddCommand(permissionsMatrixCmd)
	Cmd.AddCommand(policyCmd)
	Cmd.AddCommand(leastPrivilegeCmd)
	Cmd.AddCommand(effectiveUserCmd)
	Cmd.AddCommand(effectiveGroupCmd)
}
