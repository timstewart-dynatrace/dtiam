package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// DynatraceTokenURL is the OAuth2 token endpoint for Dynatrace SSO.
	DynatraceTokenURL = "https://sso.dynatrace.com/sso/oauth2/token"

	// tokenExpirationBuffer is the time before expiration to consider token invalid.
	tokenExpirationBuffer = 30 * time.Second
)

// DefaultScopeList is the set of OAuth scopes dtiam requests by default.
// Each scope maps to the API groups dtiam calls. A missing scope causes the
// corresponding endpoint to fail with HTTP 403, so this list must stay in sync
// with the endpoints the resource handlers use.
var DefaultScopeList = []string{
	// Identity resources: users, groups, service users, account limits.
	"account-idm-read",
	"account-idm-write",
	// Environment resources: environment list, reference data.
	"account-env-read",
	// Usage and consumption: DPS subscriptions, forecast, cost, notifications.
	"account-uac-read",
	// Account audit log resources.
	"account-audit-logs-read",
	// Platform token management.
	"platform-token:tokens:manage",
	// Policies, bindings, and boundaries.
	"iam-policies-management",
	"iam:policies:read",
	"iam:policies:write",
	"iam:bindings:read",
	"iam:bindings:write",
	// Server-side effective permission resolution.
	"iam:effective-permissions:read",
}

// defaultScopes is the space-separated scope string sent to the token endpoint.
var defaultScopes = strings.Join(DefaultScopeList, " ")

// OAuthTokenManager manages OAuth2 tokens with automatic refresh.
type OAuthTokenManager struct {
	clientID     string
	clientSecret string
	accountUUID  string
	scopes       string
	tokenURL     string

	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time

	httpClient *http.Client
}

// OAuthConfig holds OAuth configuration options.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	AccountUUID  string
	Scopes       string
	TokenURL     string
	HTTPClient   *http.Client
}

// tokenResponse represents the OAuth token endpoint response.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// NewOAuthTokenManager creates a new OAuth token manager.
func NewOAuthTokenManager(config OAuthConfig) *OAuthTokenManager {
	scopes := config.Scopes
	if scopes == "" {
		scopes = defaultScopes
	}

	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = DynatraceTokenURL
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &OAuthTokenManager{
		clientID:     config.ClientID,
		clientSecret: config.ClientSecret,
		accountUUID:  config.AccountUUID,
		scopes:       scopes,
		tokenURL:     tokenURL,
		httpClient:   httpClient,
	}
}

// GetHeaders returns HTTP headers with a valid Authorization token.
func (m *OAuthTokenManager) GetHeaders() (http.Header, error) {
	token, err := m.getToken(false)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("Content-Type", "application/json")
	return headers, nil
}

// IsValid checks if the current token is valid.
func (m *OAuthTokenManager) IsValid() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.accessToken == "" {
		return false
	}

	return time.Now().Before(m.expiresAt.Add(-tokenExpirationBuffer))
}

// Close cleans up resources.
func (m *OAuthTokenManager) Close() error {
	return nil
}

// getToken returns a valid access token, refreshing if necessary.
func (m *OAuthTokenManager) getToken(forceRefresh bool) (string, error) {
	m.mu.RLock()
	valid := m.accessToken != "" && time.Now().Before(m.expiresAt.Add(-tokenExpirationBuffer))
	m.mu.RUnlock()

	if valid && !forceRefresh {
		m.mu.RLock()
		token := m.accessToken
		m.mu.RUnlock()
		return token, nil
	}

	return m.refreshToken()
}

// refreshToken fetches a new token from the OAuth endpoint.
func (m *OAuthTokenManager) refreshToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring lock
	if m.accessToken != "" && time.Now().Before(m.expiresAt.Add(-tokenExpirationBuffer)) {
		return m.accessToken, nil
	}

	// Build token request
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", m.clientID)
	data.Set("client_secret", m.clientSecret)
	data.Set("scope", m.scopes)
	data.Set("resource", fmt.Sprintf("urn:dtaccount:%s", m.accountUUID))

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		m.tokenURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	m.accessToken = tokenResp.AccessToken
	m.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return m.accessToken, nil
}

// AccountUUID returns the account UUID.
func (m *OAuthTokenManager) AccountUUID() string {
	return m.accountUUID
}
