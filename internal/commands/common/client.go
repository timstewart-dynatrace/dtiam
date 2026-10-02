// Package common provides shared utilities for commands.
package common

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/auth"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
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
	return createClient(nil)
}

// CreateEnvironmentClient creates a client for an environment-served API (App
// Engine registry, Settings schemas, environment-level Platform IAM).
//
// A configured environment token (DTIAM_ENVIRONMENT_TOKEN or the credential's
// environment-token) is used as-is. Otherwise the account credentials are used:
// an OAuth client requests exactly the given scopes, which the account-level
// default set does not include, and a bearer token is sent unchanged.
func CreateEnvironmentClient(scopes []string) (*client.Client, error) {
	return createClient(scopes)
}

// createClient builds a client. A nil envScopes means an account-API client;
// a non-nil one means an environment-API client requesting those scopes.
func createClient(envScopes []string) (*client.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	clientID, clientSecret, accountUUID, bearerToken, useOAuth := config.GetEffectiveCredentials(cfg)
	cred := cfg.GetCurrentCredential()

	if accountUUID == "" {
		return nil, fmt.Errorf("no account UUID configured. Use 'dtiam config set-context' or set DTIAM_ACCOUNT_UUID")
	}

	var tokenProvider client.TokenProvider
	envToken := ""
	if envScopes != nil {
		envToken, err = config.ResolveEnvironmentToken(cfg)
		if err != nil {
			return nil, err
		}
	}

	switch {
	case envToken != "":
		tokenProvider = NewBearerProvider(envToken)
	case useOAuth:
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("OAuth credentials not configured. Use 'dtiam config set-credentials' or set DTIAM_CLIENT_ID and DTIAM_CLIENT_SECRET")
		}
		var scopes string
		if envScopes != nil {
			scopes = strings.Join(envScopes, " ")
		} else {
			// Honor a DTIAM_SCOPES or per-credential scope override. Without
			// this the configured value is parsed and then silently ignored.
			scopes = config.GetEffectiveScopes(cred)
			// A readonly context asks only for read scopes, so the token
			// cannot write even if a command slipped past the safety check.
			// An explicit override still wins: it is a deliberate choice.
			if scopes == "" {
				if ctx := cfg.GetCurrentContext(); ctx != nil && ctx.EffectiveSafetyLevel() == config.SafetyReadOnly {
					scopes = strings.Join(auth.ReadOnlyScopeList, " ")
				}
			}
		}
		tokenProvider = NewOAuthProviderWithScopes(clientID, clientSecret, accountUUID, scopes)
	case bearerToken != "":
		tokenProvider = NewBearerProvider(bearerToken)
	default:
		return nil, fmt.Errorf("no authentication configured. Set up OAuth credentials or use DTIAM_BEARER_TOKEN")
	}

	return client.New(client.Config{
		AccountUUID:   accountUUID,
		TokenProvider: tokenProvider,
		Verbose:       cli.GlobalState.IsVerbose(),
		APIHost:       config.GetEffectiveAPIURL(cred, ""),
	}), nil
}
