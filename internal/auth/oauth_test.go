package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newMockSSOServer(t *testing.T, statusCode int, response tokenResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", contentType)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
}

func TestNewOAuthTokenManager(t *testing.T) {
	config := OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		AccountUUID:  "test-account-uuid",
	}

	mgr := NewOAuthTokenManager(config)

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.clientID != config.ClientID {
		t.Errorf("expected clientID %q, got %q", config.ClientID, mgr.clientID)
	}
	if mgr.clientSecret != config.ClientSecret {
		t.Errorf("expected clientSecret %q, got %q", config.ClientSecret, mgr.clientSecret)
	}
	if mgr.accountUUID != config.AccountUUID {
		t.Errorf("expected accountUUID %q, got %q", config.AccountUUID, mgr.accountUUID)
	}
	if mgr.scopes != defaultScopes {
		t.Errorf("expected default scopes, got %q", mgr.scopes)
	}
	if mgr.tokenURL != DynatraceTokenURL {
		t.Errorf("expected default token URL %q, got %q", DynatraceTokenURL, mgr.tokenURL)
	}
}

func TestNewOAuthTokenManager_CustomConfig(t *testing.T) {
	customScopes := "custom-scope"
	customURL := "https://custom.example.com/token"

	config := OAuthConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		AccountUUID:  "uuid",
		Scopes:       customScopes,
		TokenURL:     customURL,
	}

	mgr := NewOAuthTokenManager(config)

	if mgr.scopes != customScopes {
		t.Errorf("expected scopes %q, got %q", customScopes, mgr.scopes)
	}
	if mgr.tokenURL != customURL {
		t.Errorf("expected tokenURL %q, got %q", customURL, mgr.tokenURL)
	}
}

func TestOAuthTokenManager_GetHeaders(t *testing.T) {
	mockToken := "mock-access-token-12345"
	server := newMockSSOServer(t, http.StatusOK, tokenResponse{
		AccessToken: mockToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	})
	defer server.Close()

	mgr := NewOAuthTokenManager(OAuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AccountUUID:  "test-uuid",
		TokenURL:     server.URL,
	})

	headers, err := mgr.GetHeaders()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	authHeader := headers.Get("Authorization")
	expected := "Bearer " + mockToken
	if authHeader != expected {
		t.Errorf("expected Authorization %q, got %q", expected, authHeader)
	}

	contentType := headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", contentType)
	}
}

func TestOAuthTokenManager_TokenRefresh(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := tokenResponse{
			AccessToken: "token-" + string(rune('0'+callCount)),
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	mgr := NewOAuthTokenManager(OAuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AccountUUID:  "test-uuid",
		TokenURL:     server.URL,
	})

	// First call should fetch a token
	_, err := mgr.GetHeaders()
	if err != nil {
		t.Fatalf("first GetHeaders failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 server call, got %d", callCount)
	}

	// Second call should use cached token (not expired)
	_, err = mgr.GetHeaders()
	if err != nil {
		t.Fatalf("second GetHeaders failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 server call (cached), got %d", callCount)
	}

	// Expire the token manually
	mgr.mu.Lock()
	mgr.expiresAt = time.Now().Add(-1 * time.Minute)
	mgr.mu.Unlock()

	// Third call should refresh the token
	_, err = mgr.GetHeaders()
	if err != nil {
		t.Fatalf("third GetHeaders failed: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 server calls after expiry, got %d", callCount)
	}
}

func TestOAuthTokenManager_ExtractClientID(t *testing.T) {
	// The Dynatrace secret format is dt0s01.CLIENTID.SECRET
	// When ClientID is empty, it should be extractable from the secret format.
	// However, looking at the implementation, NewOAuthTokenManager does not
	// auto-extract. This test verifies that a manually extracted client ID works.
	secret := "dt0s01.MYCLIENTID.SECRETPART"
	parts := strings.SplitN(secret, ".", 3)

	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}

	extractedID := parts[0] + "." + parts[1]
	expected := "dt0s01.MYCLIENTID"
	if extractedID != expected {
		t.Errorf("expected extracted client ID %q, got %q", expected, extractedID)
	}

	// Verify the manager works with extracted credentials
	server := newMockSSOServer(t, http.StatusOK, tokenResponse{
		AccessToken: "token-for-extracted",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	})
	defer server.Close()

	mgr := NewOAuthTokenManager(OAuthConfig{
		ClientID:     extractedID,
		ClientSecret: secret,
		AccountUUID:  "test-uuid",
		TokenURL:     server.URL,
	})

	headers, err := mgr.GetHeaders()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if auth := headers.Get("Authorization"); auth != "Bearer token-for-extracted" {
		t.Errorf("unexpected Authorization header: %s", auth)
	}
}

func TestOAuthTokenManager_InvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"Client authentication failed"}`))
	}))
	defer server.Close()

	mgr := NewOAuthTokenManager(OAuthConfig{
		ClientID:     "bad-client",
		ClientSecret: "bad-secret",
		AccountUUID:  "test-uuid",
		TokenURL:     server.URL,
	})

	_, err := mgr.GetHeaders()
	if err == nil {
		t.Fatal("expected error for invalid credentials, got nil")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain status 401, got: %v", err)
	}
}

func TestOAuthTokenManager_Close(t *testing.T) {
	mgr := NewOAuthTokenManager(OAuthConfig{
		ClientID:     "test",
		ClientSecret: "test",
		AccountUUID:  "test",
	})

	err := mgr.Close()
	if err != nil {
		t.Errorf("expected nil error from Close(), got %v", err)
	}
}

func TestOAuthTokenManager_IsValid(t *testing.T) {
	mgr := NewOAuthTokenManager(OAuthConfig{
		ClientID:     "test",
		ClientSecret: "test",
		AccountUUID:  "test",
	})

	// No token fetched yet — should be invalid
	if mgr.IsValid() {
		t.Error("expected IsValid() to be false before any token fetch")
	}

	// Set a valid token manually
	mgr.mu.Lock()
	mgr.accessToken = "some-token"
	mgr.expiresAt = time.Now().Add(1 * time.Hour)
	mgr.mu.Unlock()

	if !mgr.IsValid() {
		t.Error("expected IsValid() to be true with valid token")
	}

	// Set an expired token
	mgr.mu.Lock()
	mgr.expiresAt = time.Now().Add(-1 * time.Minute)
	mgr.mu.Unlock()

	if mgr.IsValid() {
		t.Error("expected IsValid() to be false with expired token")
	}
}

func TestOAuthTokenManager_AccountUUID(t *testing.T) {
	uuid := "my-account-uuid"
	mgr := NewOAuthTokenManager(OAuthConfig{
		ClientID:     "test",
		ClientSecret: "test",
		AccountUUID:  uuid,
	})

	if got := mgr.AccountUUID(); got != uuid {
		t.Errorf("expected AccountUUID %q, got %q", uuid, got)
	}
}

func TestOAuthTokenManager_ImplementsTokenProvider(t *testing.T) {
	var _ TokenProvider = (*OAuthTokenManager)(nil)
}
