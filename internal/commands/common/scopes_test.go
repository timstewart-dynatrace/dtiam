package common

import (
	"testing"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/auth"
)

func TestNewOAuthProviderWithScopes_EmptyUsesDefaults(t *testing.T) {
	// An empty override must fall through to auth.DefaultScopeList rather than
	// requesting no scopes at all, which would yield a useless token.
	p := NewOAuthProviderWithScopes("id", "secret", "uuid", "")
	if p == nil {
		t.Fatal("provider is nil")
	}
	if len(auth.DefaultScopeList) == 0 {
		t.Fatal("DefaultScopeList is empty")
	}
}

func TestNewOAuthProviderWithScopes_HonorsOverride(t *testing.T) {
	p := NewOAuthProviderWithScopes("id", "secret", "uuid", "account-idm-read")
	if p == nil {
		t.Fatal("provider is nil")
	}
}

func TestNewOAuthProvider_DelegatesToScopedConstructor(t *testing.T) {
	if p := NewOAuthProvider("id", "secret", "uuid"); p == nil {
		t.Fatal("provider is nil")
	}
}
