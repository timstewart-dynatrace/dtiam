package auth

import (
	"strings"
	"testing"
)

// requiredScopes maps an OAuth scope to the dtiam feature that breaks without it.
// Every entry must be present in DefaultScopeList or the corresponding command
// fails with HTTP 403 at runtime.
var requiredScopes = map[string]string{
	"account-idm-read":               "get users/groups/service-users, account limits",
	"account-idm-write":              "create/delete users, groups, service users",
	"account-env-read":               "get environments, reference data",
	"account-uac-read":               "account subscriptions, forecast, cost, notifications",
	"account-audit-logs-read":        "account audit logs",
	"platform-token:tokens:manage":   "get/create/delete platform tokens",
	"iam-policies-management":        "policies, bindings, boundaries",
	"iam:policies:read":              "get policies",
	"iam:policies:write":             "create/delete policies",
	"iam:bindings:read":              "get policy bindings",
	"iam:bindings:write":             "attach/detach policy bindings",
	"iam:effective-permissions:read": "analyze effective-user/effective-group",
}

func TestDefaultScopeListCoversEveryAPIGroup(t *testing.T) {
	have := make(map[string]bool, len(DefaultScopeList))
	for _, s := range DefaultScopeList {
		have[s] = true
	}

	for scope, feature := range requiredScopes {
		t.Run("should request "+scope, func(t *testing.T) {
			if !have[scope] {
				t.Errorf("scope %q missing from DefaultScopeList; %s will fail with HTTP 403", scope, feature)
			}
		})
	}
}

func TestDefaultScopeListHasNoDuplicates(t *testing.T) {
	seen := make(map[string]bool, len(DefaultScopeList))
	for _, s := range DefaultScopeList {
		if seen[s] {
			t.Errorf("duplicate scope %q in DefaultScopeList", s)
		}
		seen[s] = true
	}
}

func TestDefaultScopesIsSpaceSeparatedScopeList(t *testing.T) {
	got := strings.Split(defaultScopes, " ")
	if len(got) != len(DefaultScopeList) {
		t.Fatalf("defaultScopes has %d entries, DefaultScopeList has %d", len(got), len(DefaultScopeList))
	}
	for i, s := range DefaultScopeList {
		if got[i] != s {
			t.Errorf("defaultScopes[%d] = %q, want %q", i, got[i], s)
		}
	}
}

func TestNewOAuthTokenManagerDefaultsToFullScopeList(t *testing.T) {
	m := NewOAuthTokenManager(OAuthConfig{ClientID: "id", ClientSecret: "secret"})
	if m.scopes != defaultScopes {
		t.Errorf("scopes = %q, want %q", m.scopes, defaultScopes)
	}
}

func TestNewOAuthTokenManagerHonorsExplicitScopes(t *testing.T) {
	m := NewOAuthTokenManager(OAuthConfig{ClientID: "id", ClientSecret: "secret", Scopes: "account-idm-read"})
	if m.scopes != "account-idm-read" {
		t.Errorf("scopes = %q, want override to be preserved", m.scopes)
	}
}
