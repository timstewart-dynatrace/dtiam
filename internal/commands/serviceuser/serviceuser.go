// Package serviceuser provides service user (OAuth client) management commands.
package serviceuser

import (
	"github.com/spf13/cobra"
)

// Cmd is the service-user command.
var Cmd = &cobra.Command{
	Use:     "service-user",
	Aliases: []string{"serviceuser"},
	Short:   "Service user (OAuth client) management commands",
	Long: `Commands for managing service users (OAuth clients).

Service users are non-human identities used for automation, CI/CD pipelines,
and API integrations. Use subcommands to list, create, update, delete, and
manage group memberships for service users.`,
	Example: `  # List all service users
  dtiam service-user list

  # Get details for a service user
  dtiam service-user get my-automation-user

  # Create a new service user
  dtiam service-user create --name "CI Pipeline" --description "CI/CD automation"

  # Add a service user to a group
  dtiam service-user add-to-group my-automation-user --group GROUP_UUID`,
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(addToGroupCmd)
	Cmd.AddCommand(removeFromGroupCmd)
	Cmd.AddCommand(listGroupsCmd)
}
