// Package auth provides identity commands: who dtiam acts as, and what that
// identity (or another) is allowed to do.
package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	pkgauth "github.com/timstewart-dynatrace/dtiam/v3/pkg/auth"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/utils"
)

// Cmd is the auth command.
var Cmd = &cobra.Command{
	Use:   "auth",
	Short: "Identity commands: whoami and can-i",
	Long: `Commands about the identity dtiam authenticates as.

"auth whoami" shows who that is; "auth can-i" asks the effective-permissions
API whether it -- or another user or group -- holds a permission.`,
	Example: `  # Who am I?
  dtiam auth whoami

  # Can this credential write policies?
  dtiam auth can-i iam:policies:write

  # Can a user read logs in an environment?
  dtiam auth can-i storage:logs:read --user alice@example.com --environment abc12345`,
}

func init() {
	Cmd.AddCommand(whoamiCmd)
	Cmd.AddCommand(canICmd)

	canICmd.Flags().String("user", "", "Check this user instead of the caller (email or UID)")
	canICmd.Flags().String("group", "", "Check this group instead of the caller (UUID or name)")
	canICmd.Flags().String("environment", "", "Check at this environment's level instead of the account's")
	canICmd.Flags().Bool("strict", false, "Treat a conditional grant as no")
	canICmd.MarkFlagsMutuallyExclusive("user", "group")
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the identity dtiam authenticates as",
	Long: `Show the user or service user dtiam acts as in the current context, read from
the access token, along with its groups, the OAuth client, and the context's
safety level.

With an OAuth client this is the service user behind the client. A static
bearer token that is not a JWT has no readable identity.`,
	Example: `  # Show the current identity
  dtiam auth whoami

  # Machine-readable
  dtiam auth whoami -o json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		id, err := callerIdentity(c.AccessToken)
		if err != nil {
			return err
		}

		info := map[string]any{
			"uid":          id.Subject,
			"email":        id.Email,
			"oauth_client": id.ClientID,
			"account":      c.AccountUUID(),
		}
		if cfg, err := config.Load(); err == nil {
			info["context"] = cfg.CurrentContext
			if ctx := cfg.GetCurrentContext(); ctx != nil {
				info["safety_level"] = ctx.EffectiveSafetyLevel()
			}
		}

		printer := cli.GlobalState.NewPrinter()
		lookup := id.Email
		if lookup == "" {
			lookup = id.Subject
		}
		user, err := resources.NewUserHandler(c).Get(context.Background(), lookup)
		if err != nil {
			printer.PrintWarning("Could not look up the identity's user record: %v", err)
		} else {
			for _, k := range []string{"name", "surname", "type", "userStatus"} {
				if v, ok := user[k]; ok {
					info[k] = v
				}
			}
			info["groups"] = groupNames(user["groups"])
		}

		return printer.PrintDetail(info)
	},
}

var canICmd = &cobra.Command{
	Use:   "can-i PERMISSION",
	Short: "Check whether an identity holds a permission",
	Long: `Ask the effective-permissions API whether the caller -- or --user / --group --
holds a permission, such as iam:policies:write or storage:logs:read.

Answers:
  yes          an unconditional grant applies
  conditional  granted only where conditions hold (a boundary, a policy
               parameter, a bound group); the conditions are listed
  no           no policy grants it, or a policy denies it

Exit code is 0 for yes and conditional, 1 for no. With --strict, conditional
exits 1 too. Checks the account level unless --environment is given.

Scope of the answer: this covers platform permissions granted by IAM policies
(storage:logs:read, iam:bindings:write, document:documents:write, ...). It does
not cover Account Management access through api.dynatrace.com -- managing users,
groups and policies -- which comes from account permissions attached to groups
(see "dtiam group permissions"). OAuth scopes such as account-idm-write are not
permissions and are rejected.`,
	Example: `  # Can this credential create groups?
  dtiam auth can-i account-idm-write

  # In a script
  if dtiam auth can-i iam:policies:write --plain >/dev/null; then echo allowed; fi

  # Another user, in one environment
  dtiam auth can-i storage:logs:read --user alice@example.com --environment abc12345

  # A group, counting conditional grants as no
  dtiam auth can-i iam:bindings:write --group "Platform Team" --strict`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		permission := strings.TrimSpace(args[0])
		if strings.Count(permission, ":") < 2 {
			return fmt.Errorf("%q is not a policy permission (expected service:resource:action, e.g. "+
				"storage:logs:read); account-level scopes and roles are not covered by the "+
				"effective-permissions API -- see 'dtiam group permissions'", permission)
		}
		user, _ := cmd.Flags().GetString("user")
		group, _ := cmd.Flags().GetString("group")
		environment, _ := cmd.Flags().GetString("environment")
		strict, _ := cmd.Flags().GetBool("strict")

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()
		ctx := context.Background()

		levelType, levelID := "account", c.AccountUUID()
		if environment != "" {
			levelType, levelID = "environment", environment
		}

		api := utils.NewEffectivePermissionsAPI(c)
		var result *utils.APIEffectivePermissions
		subject := ""
		switch {
		case group != "":
			subject = "group " + group
			result, err = api.GetGroupEffectivePermissions(ctx, group, levelType, levelID, nil)
		case user != "":
			subject = "user " + user
			result, err = api.GetUserEffectivePermissions(ctx, user, levelType, levelID, nil)
		default:
			id, idErr := callerIdentity(c.AccessToken)
			if idErr != nil {
				return fmt.Errorf("%w; pass --user or --group to check someone explicitly", idErr)
			}
			subject = "caller " + id.Subject
			result, err = api.GetEffectivePermissions(ctx, id.Subject, "user", levelType, levelID, nil)
		}
		if err != nil {
			return err
		}
		if result.Error != "" {
			return fmt.Errorf("failed to resolve effective permissions for %s: %s", subject, result.Error)
		}

		decision := utils.DecidePermission(result.EffectivePermissions, permission)
		out := map[string]any{
			"permission": permission,
			"decision":   decision.Decision,
			"subject":    subject,
			"level":      levelType + ":" + levelID,
		}
		if len(decision.Conditions) > 0 {
			out["conditions"] = decision.Conditions
		}
		if decision.Reason != "" {
			out["reason"] = decision.Reason
		}

		printer := cli.GlobalState.NewPrinter()
		if err := printer.PrintDetail(out); err != nil {
			return err
		}

		if decision.Decision == utils.DecisionNo || (strict && decision.Decision == utils.DecisionConditional) {
			// A "no" is an answer, not a failure: exit 1 without an error line.
			return cli.ErrSilentExit
		}
		return nil
	},
}

// callerIdentity reads the identity from the client's access token.
func callerIdentity(token func() (string, error)) (*pkgauth.Identity, error) {
	t, err := token()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain an access token: %w", err)
	}
	return pkgauth.IdentityFromToken(t)
}

// groupNames extracts group names (or UUIDs) from a user record's groups.
func groupNames(raw any) []string {
	list, _ := raw.([]any)
	names := make([]string, 0, len(list))
	for _, g := range list {
		m, ok := g.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["groupName"].(string)
		if name == "" {
			name, _ = m["name"].(string)
		}
		if name == "" {
			name, _ = m["uuid"].(string)
		}
		names = append(names, name)
	}
	return names
}
