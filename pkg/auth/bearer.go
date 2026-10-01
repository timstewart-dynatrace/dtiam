package auth

import "net/http"

// StaticTokenManager provides a static bearer token without refresh capability.
type StaticTokenManager struct {
	token       string
	accountUUID string
}

// NewStaticTokenManager creates a new static token manager.
func NewStaticTokenManager(token, accountUUID string) *StaticTokenManager {
	return &StaticTokenManager{
		token:       token,
		accountUUID: accountUUID,
	}
}

// GetHeaders returns HTTP headers with the static bearer token.
func (m *StaticTokenManager) GetHeaders() (http.Header, error) {
	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+m.token)
	headers.Set("Content-Type", "application/json")
	return headers, nil
}

// IsValid always returns true for static tokens (we cannot validate them).
func (m *StaticTokenManager) IsValid() bool {
	return m.token != ""
}

// Close cleans up resources.
func (m *StaticTokenManager) Close() error {
	return nil
}

// AccountUUID returns the account UUID.
func (m *StaticTokenManager) AccountUUID() string {
	return m.accountUUID
}
