package create

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/common"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/output"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/resources"
)

var boundaryCmd = &cobra.Command{
	Use:   "boundary",
	Short: "Create a new boundary",
	Example: `  # Create a boundary with management zones
  dtiam create boundary --name "Production" --zone "Production" --zone "Staging"

  # Create a boundary with a custom query
  dtiam create boundary --name "Apps Only" --query 'shared:app-id IN ("dynatrace.dashboards")'

  # Create a boundary with a description
  dtiam create boundary --name "Production" --zone "Production" --description "Production boundary"

  # Dry run to preview
  dtiam create boundary --name "Production" --zone "Production" --dry-run

  # Machine-friendly output
  dtiam create boundary --name "Production" --zone "Production" -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		zones, _ := cmd.Flags().GetStringSlice("zone")
		query, _ := cmd.Flags().GetString("query")
		description, _ := cmd.Flags().GetString("description")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if len(zones) == 0 && query == "" {
			return fmt.Errorf("either --zone or --query is required")
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create boundary: %s", name)
			return nil
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		handler := resources.NewBoundaryHandler(c)
		ctx := context.Background()

		var queryPtr, descPtr *string
		if query != "" {
			queryPtr = &query
		}
		if description != "" {
			descPtr = &description
		}

		boundary, err := handler.Create(ctx, name, zones, queryPtr, descPtr)
		if err != nil {
			return err
		}

		printer.PrintSuccess("Boundary created successfully")
		return printer.Print([]map[string]any{boundary}, output.BoundaryColumns())
	},
}

func init() {
	boundaryCmd.Flags().StringP("name", "n", "", "Boundary name (required)")
	boundaryCmd.Flags().StringSliceP("zone", "z", nil, "Management zone names")
	boundaryCmd.Flags().StringP("query", "q", "", "Boundary query")
	boundaryCmd.Flags().StringP("description", "d", "", "Boundary description")
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Create a new platform token",
	Long: `Create a new platform token. The token value is only returned once during
creation and cannot be retrieved later -- save it immediately.

A token is owned by a user or service user (--user), carries one or more scopes
(--scopes), and is valid against one or more resources: the account by default,
or specific environments (--environment, or --resource with a full URN).

The owner must be the identity dtiam authenticates as: the API refuses (HTTP 403)
to mint a token for any other user or service user. For an OAuth client, that is
the service user behind the client.

Expiration defaults to 30 days. Give a lifetime with --expires-in (30d, 2w, 1y,
12h) or an exact time with --expires-at (RFC 3339).

Requires the platform-token:tokens:manage OAuth scope.`,
	Example: `  # Account-wide token owned by a user, valid for 30 days
  dtiam create token --name "CI Token" --user ci-bot@example.com \
    --scopes account-idm-read,iam-policies-management

  # Token for one environment, expiring at a fixed time
  dtiam create token --name "Logs reader" --user 1a2b3c4d-... \
    --scopes storage:logs:read --environment abc12345 --expires-at 2027-01-01T00:00:00Z

  # Dry run
  dtiam create token --name "CI Token" --user ci-bot@example.com --scopes account-idm-read --dry-run

  # Machine-friendly output (the token value is in the "token" field)
  dtiam create token --name "CI Token" --user ci-bot@example.com --scopes account-idm-read -o json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()
		name, _ := flags.GetString("name")
		scopesStr, _ := flags.GetString("scopes")
		user, _ := flags.GetString("user")
		expiresIn, _ := flags.GetString("expires-in")
		expiresAt, _ := flags.GetString("expires-at")
		resourcesFlag, _ := flags.GetStringSlice("resource")
		envs, _ := flags.GetStringSlice("environment")
		tags, _ := flags.GetStringSlice("tag")

		scopes := splitAndTrim(scopesStr)
		if len(scopes) == 0 {
			return fmt.Errorf("--scopes is required: a token must carry at least one scope")
		}

		// Resolve the expiration before anything else, so --dry-run shows the
		// real date and a bad value fails without a network call.
		if expiresIn != "" && expiresAt != "" {
			return fmt.Errorf("--expires-in and --expires-at are mutually exclusive")
		}
		expiration := expiresAt
		if expiration == "" {
			lifetime := expiresIn
			if lifetime == "" {
				lifetime = "30d"
			}
			var err error
			if expiration, err = resources.ParseTokenLifetime(lifetime, time.Now()); err != nil {
				return err
			}
		} else if _, err := time.Parse(time.RFC3339, expiration); err != nil {
			return fmt.Errorf("invalid --expires-at %q: use RFC 3339, e.g. 2027-01-01T00:00:00Z", expiration)
		}

		printer := cli.GlobalState.NewPrinter()
		if cli.GlobalState.IsDryRun() {
			printer.PrintWarning("Would create platform token %q", name)
			return printer.PrintAny(map[string]any{
				"name": name, "owner": user, "scope": scopes, "expirationDate": expiration,
				"resource": tokenResources(resourcesFlag, envs, "CURRENT_ACCOUNT"), "tags": tags,
			})
		}

		c, err := common.CreateClient()
		if err != nil {
			return err
		}
		defer c.Close()

		ctx := context.Background()
		userUID, err := resolveUserUID(ctx, c, user)
		if err != nil {
			return err
		}

		token, err := resources.NewTokenHandler(c).Create(ctx, resources.PlatformTokenRequest{
			Name:           name,
			Scopes:         scopes,
			Resources:      tokenResources(resourcesFlag, envs, c.AccountUUID()),
			Tags:           tags,
			ExpirationDate: expiration,
			UserUUID:       userUID,
		})
		if err != nil {
			return err
		}

		printer.PrintSuccess("Platform token created successfully")

		// Show the token value prominently -- it cannot be retrieved later.
		// Under --plain it is in the structured output instead.
		if tokenValue, ok := token["token"].(string); ok && tokenValue != "" && !cli.GlobalState.IsPlain() {
			printer.PrintWarning("Save this token now -- it cannot be retrieved later:")
			printer.PrintMessage("%s", tokenValue)
		}

		return printer.Print([]map[string]any{token}, output.CreatedTokenColumns())
	},
}

// tokenResources builds the token's resource URNs: explicit --resource values,
// then one per --environment, defaulting to the account.
func tokenResources(explicit, environments []string, accountUUID string) []string {
	out := append([]string{}, explicit...)
	for _, env := range environments {
		out = append(out, resources.EnvironmentResource(env))
	}
	if len(out) == 0 {
		out = append(out, resources.AccountResource(accountUUID))
	}
	return out
}

// resolveUserUID turns an email into the user's UID; anything else is taken to
// be a UID already. Service users resolve too, by their service email.
func resolveUserUID(ctx context.Context, c *client.Client, user string) (string, error) {
	if !strings.Contains(user, "@") {
		return user, nil
	}
	record, err := resources.NewUserHandler(c).Get(ctx, user)
	if err != nil {
		return "", fmt.Errorf("could not find token owner %q: %w", user, err)
	}
	uid, _ := record["uid"].(string)
	if uid == "" {
		return "", fmt.Errorf("token owner %q has no UID", user)
	}
	return uid, nil
}

func init() {
	tokenCmd.Flags().StringP("name", "n", "", "Token name (required)")
	tokenCmd.Flags().String("scopes", "", "Comma-separated scopes (required)")
	tokenCmd.Flags().String("user", "", "Owning user or service user, by email or UID (required)")
	tokenCmd.Flags().String("expires-in", "", "Lifetime, e.g. 30d, 2w, 1y, 12h (default 30d)")
	tokenCmd.Flags().String("expires-at", "", "Exact expiration, RFC 3339 (e.g. 2027-01-01T00:00:00Z)")
	tokenCmd.Flags().StringSlice("environment", nil, "Limit the token to these environment IDs (repeatable)")
	tokenCmd.Flags().StringSlice("resource", nil, "Resource URN, e.g. urn:dtaccount:UUID (repeatable)")
	tokenCmd.Flags().StringSlice("tag", nil, "Tag to attach (repeatable)")
	_ = tokenCmd.MarkFlagRequired("name")
	_ = tokenCmd.MarkFlagRequired("scopes")
	_ = tokenCmd.MarkFlagRequired("user")
}

// splitAndTrim splits a comma-separated string and trims whitespace.
func splitAndTrim(s string) []string {
	parts := make([]string, 0)
	for _, p := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
