// Package doctor provides diagnostic health checks for the dtiam setup.
package doctor

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtimothystewart/dtiam/internal/auth"
	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/common"
	"github.com/jtimothystewart/dtiam/internal/config"
	"github.com/jtimothystewart/dtiam/internal/output"
	"github.com/jtimothystewart/dtiam/internal/resources"
	"github.com/jtimothystewart/dtiam/pkg/version"
)

// Check statuses, ordered by severity.
const (
	StatusOK   = "ok"
	StatusWarn = "warn"
	StatusFail = "fail"
	StatusSkip = "skip"
)

// CheckResult is the outcome of one diagnostic check.
type CheckResult struct {
	Name   string `json:"name" yaml:"name" table:"CHECK"`
	Status string `json:"status" yaml:"status" table:"STATUS"`
	Detail string `json:"detail" yaml:"detail" table:"DETAIL"`
}

// Cmd is the doctor command.
var Cmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check configuration, credentials, and API connectivity",
	Long: `Run diagnostic checks to verify dtiam is configured correctly and can reach
the Dynatrace Account Management API.

Checks performed:
  1. dtiam version
  2. Configuration file exists and parses
  3. A current context is selected
  4. Account UUID is resolvable
  5. Credentials are configured (OAuth2 or bearer token)
  6. OAuth scope set in use, and whether any are missing
  7. Token can be obtained from the SSO endpoint
  8. The account API answers an authenticated request

Checks 7 and 8 make network calls; everything before them is local. The command
exits non-zero if any check fails, so it is usable as a CI readiness gate.`,
	Example: `  # Run all checks
  dtiam doctor

  # Check a specific context
  dtiam doctor --context production

  # Local checks only, no network calls
  dtiam doctor --offline

  # Machine-readable output for CI
  dtiam doctor --plain`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		offline, _ := cmd.Flags().GetBool("offline")

		results := RunChecks(context.Background(), offline)

		printer := cli.GlobalState.NewPrinter()
		if err := printer.Print(resultsToMaps(results), DoctorColumns()); err != nil {
			return err
		}

		// A failing check is reported through the exit code so CI can gate on it,
		// but the table is printed first so the user sees every result.
		var failed []string
		for _, r := range results {
			if r.Status == StatusFail {
				failed = append(failed, r.Name)
			}
		}
		if len(failed) > 0 {
			return fmt.Errorf("%d check(s) failed: %s", len(failed), strings.Join(failed, ", "))
		}
		return nil
	},
}

func init() {
	Cmd.Flags().Bool("offline", false, "Skip the checks that make network calls")
}

// DoctorColumns returns the columns for doctor output.
func DoctorColumns() []output.Column {
	return []output.Column{
		{Key: "check", Header: "CHECK"},
		{Key: "status", Header: "STATUS"},
		{Key: "detail", Header: "DETAIL"},
	}
}

// RunChecks runs every diagnostic check in order and returns the results.
//
// Checks are ordered cheapest-first so a misconfiguration is reported before any
// network call is attempted, and later checks are skipped rather than failed when
// an earlier one makes them meaningless.
func RunChecks(ctx context.Context, offline bool) []CheckResult {
	results := []CheckResult{versionCheck()}

	cfg, cfgResult := configCheck()
	results = append(results, cfgResult)

	if cfg == nil {
		return append(results, skipRemaining(offline,
			"configuration could not be loaded")...)
	}

	results = append(results, contextCheck(cfg))

	clientID, clientSecret, accountUUID, bearerToken, useOAuth := config.GetEffectiveCredentials(cfg)

	results = append(results, accountCheck(accountUUID))
	results = append(results, credentialsCheck(clientID, clientSecret, bearerToken, useOAuth))
	results = append(results, scopesCheck(cfg, useOAuth))

	if accountUUID == "" || (useOAuth && (clientID == "" || clientSecret == "")) ||
		(!useOAuth && bearerToken == "") {
		return append(results, skipChecks(offline, "credentials are incomplete")...)
	}

	if offline {
		return append(results, skipChecks(true, "--offline")...)
	}

	results = append(results, tokenCheck(ctx, clientID, clientSecret, accountUUID, bearerToken, useOAuth))
	results = append(results, apiCheck(ctx))

	return results
}

// versionCheck reports the running version.
func versionCheck() CheckResult {
	return CheckResult{
		Name:   "dtiam version",
		Status: StatusOK,
		Detail: version.Version,
	}
}

// configCheck loads the configuration file.
func configCheck() (*config.Config, CheckResult) {
	path, pathErr := config.GetConfigPath()
	if pathErr != nil {
		path = "(unknown)"
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, CheckResult{
			Name:   "configuration file",
			Status: StatusFail,
			Detail: fmt.Sprintf("%s: %v", path, err),
		}
	}

	if !config.Exists() {
		// Environment variables alone are a legitimate setup, so this is a
		// warning rather than a failure.
		return cfg, CheckResult{
			Name:   "configuration file",
			Status: StatusWarn,
			Detail: fmt.Sprintf("no config at %s; relying on environment variables", path),
		}
	}

	return cfg, CheckResult{
		Name:   "configuration file",
		Status: StatusOK,
		Detail: path,
	}
}

// contextCheck verifies a current context is selected.
func contextCheck(cfg *config.Config) CheckResult {
	if cfg.CurrentContext == "" {
		return CheckResult{
			Name:   "current context",
			Status: StatusWarn,
			Detail: "none selected; run 'dtiam config use-context NAME'",
		}
	}
	if cfg.GetCurrentContext() == nil {
		return CheckResult{
			Name:   "current context",
			Status: StatusFail,
			Detail: fmt.Sprintf("%q is selected but not defined in the config", cfg.CurrentContext),
		}
	}
	return CheckResult{
		Name:   "current context",
		Status: StatusOK,
		Detail: cfg.CurrentContext,
	}
}

// accountCheck verifies an account UUID is resolvable.
func accountCheck(accountUUID string) CheckResult {
	if accountUUID == "" {
		return CheckResult{
			Name:   "account UUID",
			Status: StatusFail,
			Detail: "not set; run 'dtiam config set-context' or set DTIAM_ACCOUNT_UUID",
		}
	}
	return CheckResult{Name: "account UUID", Status: StatusOK, Detail: accountUUID}
}

// credentialsCheck verifies credentials are present and reports which kind.
func credentialsCheck(clientID, clientSecret, bearerToken string, useOAuth bool) CheckResult {
	if useOAuth {
		switch {
		case clientSecret == "":
			return CheckResult{
				Name:   "credentials",
				Status: StatusFail,
				Detail: "OAuth selected but no client secret; set DTIAM_CLIENT_SECRET",
			}
		case clientID == "":
			return CheckResult{
				Name:   "credentials",
				Status: StatusFail,
				Detail: "OAuth selected but no client ID could be derived from the secret",
			}
		default:
			return CheckResult{
				Name:   "credentials",
				Status: StatusOK,
				Detail: fmt.Sprintf("OAuth2 client %s", clientID),
			}
		}
	}

	if bearerToken == "" {
		return CheckResult{
			Name:   "credentials",
			Status: StatusFail,
			Detail: "none configured; set up OAuth credentials or DTIAM_BEARER_TOKEN",
		}
	}

	// A static token cannot refresh, so flag it rather than reporting plain OK.
	return CheckResult{
		Name:   "credentials",
		Status: StatusWarn,
		Detail: "static bearer token; does not auto-refresh and will fail when it expires",
	}
}

// scopesCheck reports the scope set in use and names any default scope missing
// from an override.
func scopesCheck(cfg *config.Config, useOAuth bool) CheckResult {
	if !useOAuth {
		return CheckResult{
			Name:   "OAuth scopes",
			Status: StatusSkip,
			Detail: "not applicable to bearer token auth",
		}
	}

	override := config.GetEffectiveScopes(cfg.GetCurrentCredential())
	if override == "" {
		return CheckResult{
			Name:   "OAuth scopes",
			Status: StatusOK,
			Detail: fmt.Sprintf("default set (%d scopes)", len(auth.DefaultScopeList)),
		}
	}

	// An override silently drops whatever it omits, which is the most common
	// cause of an unexpected 403, so name the missing scopes explicitly.
	have := map[string]bool{}
	for _, s := range strings.Fields(override) {
		have[s] = true
	}
	var missing []string
	for _, s := range auth.DefaultScopeList {
		if !have[s] {
			missing = append(missing, s)
		}
	}

	if len(missing) > 0 {
		return CheckResult{
			Name:   "OAuth scopes",
			Status: StatusWarn,
			Detail: fmt.Sprintf("override omits %d default scope(s): %s",
				len(missing), strings.Join(missing, ", ")),
		}
	}

	return CheckResult{
		Name:   "OAuth scopes",
		Status: StatusOK,
		Detail: fmt.Sprintf("override with %d scopes, covering the defaults", len(have)),
	}
}

// tokenCheck verifies a token can actually be obtained.
func tokenCheck(ctx context.Context, clientID, clientSecret, accountUUID, bearerToken string, useOAuth bool) CheckResult {
	if !useOAuth {
		return CheckResult{
			Name:   "token retrieval",
			Status: StatusSkip,
			Detail: "static bearer token requires no exchange",
		}
	}

	mgr := auth.NewOAuthTokenManager(auth.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AccountUUID:  accountUUID,
	})
	defer func() { _ = mgr.Close() }()

	if _, err := mgr.GetHeaders(); err != nil {
		return CheckResult{
			Name:   "token retrieval",
			Status: StatusFail,
			Detail: fmt.Sprintf("SSO token request failed: %v", err),
		}
	}

	return CheckResult{
		Name:   "token retrieval",
		Status: StatusOK,
		Detail: "obtained a token from sso.dynatrace.com",
	}
}

// apiCheck makes one authenticated request to confirm the account is reachable.
func apiCheck(ctx context.Context) CheckResult {
	c, err := common.CreateClient()
	if err != nil {
		return CheckResult{
			Name:   "account API",
			Status: StatusFail,
			Detail: err.Error(),
		}
	}
	defer func() { _ = c.Close() }()

	// Groups is the cheapest authenticated read that proves both connectivity
	// and that the token carries account-idm-read.
	groups, err := resources.NewGroupHandler(c).List(ctx, nil)
	if err != nil {
		return CheckResult{
			Name:   "account API",
			Status: StatusFail,
			Detail: fmt.Sprintf("GET /groups failed: %v", err),
		}
	}

	return CheckResult{
		Name:   "account API",
		Status: StatusOK,
		Detail: fmt.Sprintf("authenticated; %d group(s) visible", len(groups)),
	}
}

// skipChecks returns skip results for the two network checks.
func skipChecks(offline bool, reason string) []CheckResult {
	return []CheckResult{
		{Name: "token retrieval", Status: StatusSkip, Detail: "skipped: " + reason},
		{Name: "account API", Status: StatusSkip, Detail: "skipped: " + reason},
	}
}

// skipRemaining returns skip results for every check after the config check.
func skipRemaining(offline bool, reason string) []CheckResult {
	results := []CheckResult{
		{Name: "current context", Status: StatusSkip, Detail: "skipped: " + reason},
		{Name: "account UUID", Status: StatusSkip, Detail: "skipped: " + reason},
		{Name: "credentials", Status: StatusSkip, Detail: "skipped: " + reason},
		{Name: "OAuth scopes", Status: StatusSkip, Detail: "skipped: " + reason},
	}
	return append(results, skipChecks(offline, reason)...)
}

// resultsToMaps converts results for the printer.
func resultsToMaps(results []CheckResult) []map[string]any {
	out := make([]map[string]any, 0, len(results))
	for _, r := range results {
		out = append(out, map[string]any{
			"check":  r.Name,
			"status": r.Status,
			"detail": r.Detail,
		})
	}
	return out
}
