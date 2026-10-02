package group

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/prompt"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

var (
	grantPermissionNameFlag  string
	grantScopeFlag           string
	grantScopeTypeFlag       string
	grantReplaceFlag         bool
	revokePermissionNameFlag string
	revokeScopeFlag          string
	revokeScopeTypeFlag      string
	revokeForceFlag          bool
)

func init() {
	grantPermissionCmd.Flags().StringVar(&grantPermissionNameFlag, "permission", "",
		"Permission name to grant (see 'dtiam get available-permissions')")
	grantPermissionCmd.Flags().StringVar(&grantScopeFlag, "scope", "",
		"Scope value: account UUID, environment ID, or '{env-id}:{mz-id}'")
	grantPermissionCmd.Flags().StringVar(&grantScopeTypeFlag, "scope-type", "tenant",
		"Scope type: account, tenant, or management-zone")
	grantPermissionCmd.Flags().BoolVar(&grantReplaceFlag, "replace", false,
		"Replace all existing permissions instead of adding to them")
	_ = grantPermissionCmd.MarkFlagRequired("permission")
	_ = grantPermissionCmd.MarkFlagRequired("scope")

	revokePermissionCmd.Flags().StringVar(&revokePermissionNameFlag, "permission", "",
		"Permission name to revoke")
	revokePermissionCmd.Flags().StringVar(&revokeScopeFlag, "scope", "",
		"Scope value the permission was granted on")
	revokePermissionCmd.Flags().StringVar(&revokeScopeTypeFlag, "scope-type", "tenant",
		"Scope type: account, tenant, or management-zone")
	revokePermissionCmd.Flags().BoolVarP(&revokeForceFlag, "force", "f", false,
		"Skip confirmation")
	_ = revokePermissionCmd.MarkFlagRequired("permission")
	_ = revokePermissionCmd.MarkFlagRequired("scope")
}

var permissionsCmd = &cobra.Command{
	Use:     "permissions IDENTIFIER",
	Aliases: []string{"permission"},
	Short:   "List the permissions granted directly to a group",
	Long: `List the permissions granted directly to a group.

These are role-style grants that predate IAM policies and still coexist with
them. A group's effective access is the union of its policy bindings and these
direct grants, so reviewing only "dtiam group bindings" understates what a group
can actually do.

Use "dtiam get available-permissions" to see which permission names are valid.

Requires the account-idm-read OAuth scope.`,
	Example: `  # List a group's direct permission grants
  dtiam group permissions "Platform Admins"

  # By UUID, as JSON
  dtiam group permissions abc-123 -o json

  # Machine-friendly output
  dtiam group permissions "Platform Admins" --plain`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		ctx := context.Background()
		printer := cli.GlobalState.NewPrinter()

		groupUUID, err := resolveGroupUUID(ctx, c, args[0])
		if err != nil {
			return err
		}

		perms, err := resources.NewGroupPermissionHandler(c).List(ctx, groupUUID)
		if err != nil {
			return err
		}

		return printer.Print(permissionsToMaps(perms), output.GroupPermissionColumns())
	},
}

var grantPermissionCmd = &cobra.Command{
	Use:   "grant-permission IDENTIFIER",
	Short: "Grant a permission to a group",
	Long: `Grant a direct permission to a group.

By default the permission is added to the group's existing grants. With --replace
the group's permissions are replaced by exactly this one, which removes every
other direct grant -- that is destructive and requires confirmation.

Scope types:
  account         scope is the account UUID
  tenant          scope is the environment ID
  management-zone scope is "{environment-id}:{management-zone-id}"

Requires the account-idm-write OAuth scope.`,
	Example: `  # Grant environment viewer access
  dtiam group grant-permission "Dev Team" \
    --permission tenant-viewer --scope abc12345 --scope-type tenant

  # Grant account-level user management
  dtiam group grant-permission "Admins" \
    --permission account-user-management --scope $DTIAM_ACCOUNT_UUID --scope-type account

  # Scope to a single management zone
  dtiam group grant-permission "Dev Team" \
    --permission tenant-viewer --scope "abc12345:-1234567890" --scope-type management-zone

  # Preview without applying
  dtiam group grant-permission "Dev Team" \
    --permission tenant-viewer --scope abc12345 --dry-run

  # Replace all existing grants with just this one
  dtiam group grant-permission "Dev Team" \
    --permission tenant-viewer --scope abc12345 --replace`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		perm := resources.GroupPermission{
			PermissionName: grantPermissionNameFlag,
			Scope:          grantScopeFlag,
			ScopeType:      grantScopeTypeFlag,
		}

		// Validate before touching the network so a bad scope type fails fast
		// and --dry-run reports the same error a real run would.
		if err := resources.ValidatePermissions([]resources.GroupPermission{perm}); err != nil {
			return err
		}

		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			action := "grant"
			if grantReplaceFlag {
				action = "replace all permissions with"
			}
			printer.PrintWarning("Dry run: would %s %q on %s %q for group %q",
				action, perm.PermissionName, perm.ScopeType, perm.Scope, args[0])
			return printer.PrintAny(perm)
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		ctx := context.Background()
		groupUUID, err := resolveGroupUUID(ctx, c, args[0])
		if err != nil {
			return err
		}

		handler := resources.NewGroupPermissionHandler(c)

		if grantReplaceFlag {
			// --replace drops every other grant, so confirm like a delete.
			if !prompt.ConfirmDelete("all other permissions on group", args[0],
				cli.GlobalState.IsPlain()) {
				printer.PrintMessage("Aborted.")
				return nil
			}
			if err := handler.Replace(ctx, groupUUID, []resources.GroupPermission{perm}); err != nil {
				return err
			}
			printer.PrintSuccess("Replaced permissions on group %q with %q",
				args[0], perm.PermissionName)
			return nil
		}

		if err := handler.Grant(ctx, groupUUID, []resources.GroupPermission{perm}); err != nil {
			return err
		}
		printer.PrintSuccess("Granted %q on %s %q to group %q",
			perm.PermissionName, perm.ScopeType, perm.Scope, args[0])
		return nil
	},
}

var revokePermissionCmd = &cobra.Command{
	Use:   "revoke-permission IDENTIFIER",
	Short: "Revoke a permission from a group",
	Long: `Revoke a direct permission grant from a group.

The grant is identified by the combination of permission name, scope, and scope
type, because a grant has no identifier of its own. All three must match an
existing grant exactly.

Requires confirmation unless --force or --plain is set.
Requires the account-idm-write OAuth scope.`,
	Example: `  # Revoke environment viewer access
  dtiam group revoke-permission "Dev Team" \
    --permission tenant-viewer --scope abc12345 --scope-type tenant

  # Skip the confirmation prompt
  dtiam group revoke-permission "Dev Team" \
    --permission tenant-viewer --scope abc12345 --force

  # Preview the revocation
  dtiam group revoke-permission "Dev Team" \
    --permission tenant-viewer --scope abc12345 --dry-run

  # Machine-friendly (skips prompts)
  dtiam group revoke-permission "Dev Team" \
    --permission tenant-viewer --scope abc12345 --plain`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cli.GlobalState.NewPrinter()

		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Dry run: would revoke %q on %s %q from group %q",
				revokePermissionNameFlag, revokeScopeTypeFlag, revokeScopeFlag, args[0])
			return nil
		}

		target := fmt.Sprintf("%s on %s %s", revokePermissionNameFlag, revokeScopeTypeFlag, revokeScopeFlag)
		if !prompt.ConfirmDelete("permission", target, revokeForceFlag || cli.GlobalState.IsPlain()) {
			printer.PrintMessage("Aborted.")
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		ctx := context.Background()
		groupUUID, err := resolveGroupUUID(ctx, c, args[0])
		if err != nil {
			return err
		}

		if err := resources.NewGroupPermissionHandler(c).Revoke(
			ctx, groupUUID, revokePermissionNameFlag, revokeScopeFlag, revokeScopeTypeFlag); err != nil {
			return err
		}

		printer.PrintSuccess("Revoked %q from group %q", revokePermissionNameFlag, args[0])
		return nil
	},
}

// resolveGroupUUID resolves a group name or UUID to its UUID.
func resolveGroupUUID(ctx context.Context, c *client.Client, identifier string) (string, error) {
	group, err := resources.GetOrResolve(ctx, resources.NewGroupHandler(c), identifier)
	if err != nil {
		return "", err
	}
	if group == nil {
		return "", fmt.Errorf("group %q not found", identifier)
	}

	uuid, _ := group["uuid"].(string)
	if uuid == "" {
		return "", fmt.Errorf("group %q has no UUID", identifier)
	}
	return uuid, nil
}

// permissionsToMaps converts typed grants to maps for the printer.
func permissionsToMaps(perms []resources.GroupPermission) []map[string]any {
	out := make([]map[string]any, 0, len(perms))
	for _, p := range perms {
		out = append(out, map[string]any{
			"permissionName": p.PermissionName,
			"scope":          p.Scope,
			"scopeType":      p.ScopeType,
			"createdAt":      p.CreatedAt,
			"updatedAt":      p.UpdatedAt,
		})
	}
	return out
}
