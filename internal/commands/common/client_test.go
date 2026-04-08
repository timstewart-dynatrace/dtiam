package common

import (
	"testing"
)

func TestCreateClient_DoesNotPanic(t *testing.T) {
	// CreateClient should never panic regardless of config state.
	// It may succeed (if config/env vars provide credentials) or
	// return an error (if not configured). Either is acceptable.
	client, err := CreateClient()
	if err != nil {
		// Expected when no credentials configured -- verify error is descriptive
		if err.Error() == "" {
			t.Error("CreateClient error should have a descriptive message")
		}
		return
	}
	// If it succeeded, we got a valid client -- close it
	if client != nil {
		client.Close()
	}
}

func TestNewBearerProvider_IsValid(t *testing.T) {
	provider := NewBearerProvider("test-token")
	if provider == nil {
		t.Fatal("NewBearerProvider should return a non-nil provider")
	}
	if !provider.IsValid() {
		t.Error("bearer provider with a token should be valid")
	}
}

func TestNewOAuthProvider_ReturnsProvider(t *testing.T) {
	provider := NewOAuthProvider("client-id", "client-secret", "account-uuid")
	if provider == nil {
		t.Fatal("NewOAuthProvider should return a non-nil provider")
	}
}
