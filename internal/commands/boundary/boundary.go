// Package boundary provides boundary management commands.
package boundary

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Cmd is the boundary command.
var Cmd = &cobra.Command{
	Use:   "boundary",
	Short: "Attach, detach, and inspect boundaries on policy bindings",
	Long: `Commands for attaching and detaching boundaries from policy bindings.

Boundaries restrict the scope of a policy binding to a subset of resources
such as management zones, app IDs, or schema IDs. Use these commands to
manage which boundaries are associated with specific group/policy bindings.`,
	Example: `  # Attach a boundary to a binding
  dtiam boundary attach --group GROUP_UUID --policy POLICY_UUID --boundary BOUNDARY_UUID

  # Detach a boundary from a binding
  dtiam boundary detach --group GROUP_UUID --policy POLICY_UUID --boundary BOUNDARY_UUID

  # List policies using a specific boundary
  dtiam boundary list-attached BOUNDARY_UUID`,
}

func init() {
	Cmd.AddCommand(attachCmd)
	Cmd.AddCommand(detachCmd)
	Cmd.AddCommand(listAttachedCmd)
	Cmd.AddCommand(createAppBoundaryCmd)
	Cmd.AddCommand(createSchemaBoundaryCmd)
}

// buildBoundaryQuery builds a boundary query string from a prefix, operator, and list of IDs.
func buildBoundaryQuery(prefix, operator string, ids []string) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = fmt.Sprintf("%q", id)
	}
	return fmt.Sprintf("%s %s (%s)", prefix, operator, strings.Join(quoted, ", "))
}
