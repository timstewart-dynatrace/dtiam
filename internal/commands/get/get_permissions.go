package get

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

var availablePermissionsNameFlag string

func init() {
	availablePermissionsCmd.Flags().StringVar(&availablePermissionsNameFlag, "name", "",
		"Filter by permission ID or description (case-insensitive substring)")
}

var availablePermissionsCmd = &cobra.Command{
	Use:     "available-permissions",
	Aliases: []string{"available-permission", "permission-reference"},
	Short:   "List every permission the account can grant",
	Long: `List all permission names that can be granted to a group.

This is reference data describing what the account accepts, not what it
currently has assigned. It is the authoritative source for the permission names
used by "dtiam group grant-permission", so it is the right way to check a name
before attempting a grant.

To see the permissions actually granted to a group, use
"dtiam group permissions GROUP" instead.

Requires the account-env-read OAuth scope.`,
	Example: `  # List every grantable permission
  dtiam get available-permissions

  # Find the tenant-level permissions
  dtiam get available-permissions --name tenant

  # Machine-friendly output
  dtiam get available-permissions --plain`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewReferenceHandler(c)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		perms, err := handler.ListPermissions(ctx)
		if err != nil {
			return err
		}

		if availablePermissionsNameFlag != "" {
			needle := strings.ToLower(availablePermissionsNameFlag)
			filtered := make([]map[string]any, 0, len(perms))
			for _, p := range perms {
				id, _ := p["id"].(string)
				desc, _ := p["description"].(string)
				if strings.Contains(strings.ToLower(id), needle) ||
					strings.Contains(strings.ToLower(desc), needle) {
					filtered = append(filtered, p)
				}
			}
			perms = filtered
		}

		return printer.Print(perms, output.ReferencePermissionColumns())
	},
}
