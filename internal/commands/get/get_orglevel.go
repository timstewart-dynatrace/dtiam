package get

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/pkg/auth"
	"github.com/jtimothystewart/dtiam/pkg/output"
	"github.com/jtimothystewart/dtiam/pkg/resources"
)

var (
	envUsersEnvironmentFlag  string
	envUsersSearchFlag       string
	envUsersUUIDFlag         string
	envGroupsEnvironmentFlag string
	envGroupsSearchFlag      string
)

func init() {
	envUsersCmd.Flags().StringVar(&envUsersEnvironmentFlag, "environment", "",
		"Environment ID or URL (defaults to DTIAM_ENVIRONMENT_URL)")
	envUsersCmd.Flags().StringVar(&envUsersSearchFlag, "search", "",
		"Partial email or name to search for")
	envUsersCmd.Flags().StringVar(&envUsersUUIDFlag, "uuid", "",
		"User UUID to look up")
	envUsersCmd.Flags().String("level", "environment",
		"Organizational level to search: account or environment")

	envGroupsCmd.Flags().StringVar(&envGroupsEnvironmentFlag, "environment", "",
		"Environment ID or URL (defaults to DTIAM_ENVIRONMENT_URL)")
	envGroupsCmd.Flags().StringVar(&envGroupsSearchFlag, "search", "",
		"Partial group name to search for")
	envGroupsCmd.Flags().String("level", "environment",
		"Organizational level to search: account or environment")
}

// resolveEnvironmentURL returns the explicit flag value, falling back to the
// environment URL from config or DTIAM_ENVIRONMENT_URL.
func resolveEnvironmentURL(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return cli.GlobalState.EnvironmentURL()
}

// environmentIDFromURL extracts the environment ID from an environment URL.
// "https://abc12345.apps.dynatrace.com" yields "abc12345"; a bare ID is
// returned unchanged.
func environmentIDFromURL(envURL string) string {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(envURL, "https://"), "http://")
	if host, _, found := strings.Cut(trimmed, "/"); found {
		trimmed = host
	}
	if sub, _, found := strings.Cut(trimmed, "."); found {
		return sub
	}
	return trimmed
}

// orgLevelTarget resolves the --level flag into a (levelType, levelID) pair.
//
// Account level is identified by the account UUID; environment level by the
// environment ID, which is derived from the environment URL so callers do not
// have to repeat it.
func orgLevelTarget(cmd *cobra.Command, accountUUID, envURL string) (levelType, levelID string, err error) {
	levelType, _ = cmd.Flags().GetString("level")
	if levelType == "" {
		levelType = resources.LevelEnvironment
	}
	if err := resources.ValidateLevelType(levelType); err != nil {
		return "", "", err
	}

	switch levelType {
	case resources.LevelAccount:
		if accountUUID == "" {
			return "", "", fmt.Errorf("account UUID is required for account-level lookups")
		}
		return levelType, accountUUID, nil
	default:
		id := environmentIDFromURL(envURL)
		if id == "" {
			return "", "", fmt.Errorf(
				"environment ID could not be determined; pass --environment or set DTIAM_ENVIRONMENT_URL")
		}
		return levelType, id, nil
	}
}

var envUsersCmd = &cobra.Command{
	Use:     "env-users",
	Aliases: []string{"env-user", "environment-users"},
	Short:   "Search users visible at an environment level",
	Long: `Search active users through the environment-level Platform IAM API.

This is a different API from "dtiam get users". That command lists the account's
user records; this one answers which users are actually visible and assigned at
an environment, served from the environment itself
(https://{env}.apps.dynatrace.com/platform/iam/v1) rather than from
api.dynatrace.com.

The API requires a search term or a user UUID -- it will not enumerate every
user. Requires the iam:users:read scope, which dtiam requests for this command
only, or an environment token (DTIAM_ENVIRONMENT_TOKEN).`,
	Example: `  # Find users matching a partial email
  dtiam get env-users --environment abc12345 --search alice

  # Look up one user by UUID
  dtiam get env-users --environment abc12345 --uuid 1234-5678

  # Use the configured environment URL
  dtiam get env-users --search alice

  # Search at the account level instead of a single environment
  dtiam get env-users --level account --search alice`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		envURL := resolveEnvironmentURL(envUsersEnvironmentFlag)

		c, err := common.CreateEnvironmentClient(auth.EnvironmentIAMScopes)
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewOrgLevelHandler(c, envURL)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		levelType, levelID, err := orgLevelTarget(cmd, c.AccountUUID(), envURL)
		if err != nil {
			return err
		}

		users, err := handler.ListUsers(ctx, levelType, levelID, envUsersSearchFlag, envUsersUUIDFlag)
		if err != nil {
			return err
		}

		return printer.Print(users, output.OrgLevelUserColumns())
	},
}

var envGroupsCmd = &cobra.Command{
	Use:     "env-groups",
	Aliases: []string{"env-group", "environment-groups"},
	Short:   "List groups visible at an environment level",
	Long: `List groups through the environment-level Platform IAM API.

Like "get env-users", this is served from the environment rather than from the
account API, and reports which groups are visible at that organizational level.
The API requires a search term of at least 3 characters (--search). Requires the
iam:groups:read scope, which dtiam requests for this command only, or an
environment token (DTIAM_ENVIRONMENT_TOKEN).`,
	Example: `  # Find groups visible in an environment by partial name
  dtiam get env-groups --environment abc12345 --search admin

  # Machine-friendly output
  dtiam get env-groups --environment abc12345 --plain`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		envURL := resolveEnvironmentURL(envGroupsEnvironmentFlag)

		c, err := common.CreateEnvironmentClient(auth.EnvironmentIAMScopes)
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewOrgLevelHandler(c, envURL)
		printer := cli.GlobalState.NewPrinter()
		ctx := context.Background()

		levelType, levelID, err := orgLevelTarget(cmd, c.AccountUUID(), envURL)
		if err != nil {
			return err
		}

		groups, err := handler.ListGroups(ctx, levelType, levelID, envGroupsSearchFlag)
		if err != nil {
			return err
		}

		return printer.Print(groups, output.OrgLevelGroupColumns())
	},
}
