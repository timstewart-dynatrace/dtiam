// Package common provides shared utilities for commands.
package common

import (
	"fmt"
	"net/http"

	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/pkg/auth"
	"github.com/jtimothystewart/dtiam/pkg/client"
	"github.com/jtimothystewart/dtiam/pkg/config"
)

// tokenProviderAdapter adapts auth.TokenProvider to client.TokenProvider
type tokenProviderAdapter struct {
	provider auth.TokenProvider
}

func (a *tokenProviderAdapter) GetHeaders() (http.Header, error) {
	return a.provider.GetHeaders()
}

func (a *tokenProviderAdapter) IsValid() bool {
	return a.provider.IsValid()
}

func (a *tokenProviderAdapter) Close() error {
	return a.provider.Close()
}

// NewOAuthProvider creates a new OAuth token provider using the default scopes.
func NewOAuthProvider(clientID, clientSecret, accountUUID string) client.TokenProvider {
	return NewOAuthProviderWithScopes(clientID, clientSecret, accountUUID, "")
}

// NewOAuthProviderWithScopes creates an OAuth token provider with an explicit
// scope override. An empty scopes string uses auth.DefaultScopeList.
func NewOAuthProviderWithScopes(clientID, clientSecret, accountUUID, scopes string) client.TokenProvider {
	return &tokenProviderAdapter{
		provider: auth.NewOAuthTokenManager(auth.OAuthConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			AccountUUID:  accountUUID,
			Scopes:       scopes,
		}),
	}
}

// NewBearerProvider creates a new static bearer token provider.
func NewBearerProvider(token string) client.TokenProvider {
	return &tokenProviderAdapter{
		provider: auth.NewStaticTokenManager(token, ""),
	}
}

// CreateClient creates an API client from the current configuration.
func CreateClient() (*client.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	clientID, clientSecret, accountUUID, bearerToken, useOAuth := config.GetEffectiveCredentials(cfg)

	if accountUUID == "" {
		return nil, fmt.Errorf("no account UUID configured. Use 'dtiam config set-context' or set DTIAM_ACCOUNT_UUID")
	}

	var tokenProvider client.TokenProvider
	if useOAuth {
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("OAuth credentials not configured. Use 'dtiam config set-credentials' or set DTIAM_CLIENT_ID and DTIAM_CLIENT_SECRET")
		}
		// Honor a DTIAM_SCOPES or per-credential scope override. Without this
		// the configured value is parsed and then silently ignored.
		scopes := config.GetEffectiveScopes(cfg.GetCurrentCredential())
		tokenProvider = NewOAuthProviderWithScopes(clientID, clientSecret, accountUUID, scopes)
	} else if bearerToken != "" {
		tokenProvider = NewBearerProvider(bearerToken)
	} else {
		return nil, fmt.Errorf("no authentication configured. Set up OAuth credentials or use DTIAM_BEARER_TOKEN")
	}

	return client.New(client.Config{
		AccountUUID:   accountUUID,
		TokenProvider: tokenProvider,
		Verbose:       cli.GlobalState.IsVerbose(),
	}), nil
}
