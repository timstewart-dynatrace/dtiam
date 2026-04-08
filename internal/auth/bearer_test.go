package auth

import (
	"testing"
)

func TestNewStaticTokenManager(t *testing.T) {
	token := "test-bearer-token"
	accountUUID := "test-account-uuid"

	mgr := NewStaticTokenManager(token, accountUUID)

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.token != token {
		t.Errorf("expected token %q, got %q", token, mgr.token)
	}
	if mgr.accountUUID != accountUUID {
		t.Errorf("expected accountUUID %q, got %q", accountUUID, mgr.accountUUID)
	}
}

func TestStaticTokenManager_GetHeaders(t *testing.T) {
	token := "my-secret-token"
	mgr := NewStaticTokenManager(token, "account-123")

	headers, err := mgr.GetHeaders()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	authHeader := headers.Get("Authorization")
	expected := "Bearer my-secret-token"
	if authHeader != expected {
		t.Errorf("expected Authorization %q, got %q", expected, authHeader)
	}

	contentType := headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", contentType)
	}
}

func TestStaticTokenManager_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{
			name:  "should return true when token is set",
			token: "valid-token",
			want:  true,
		},
		{
			name:  "should return false when token is empty",
			token: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewStaticTokenManager(tt.token, "account-123")
			if got := mgr.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStaticTokenManager_Close(t *testing.T) {
	mgr := NewStaticTokenManager("token", "account-123")

	err := mgr.Close()
	if err != nil {
		t.Errorf("expected nil error from Close(), got %v", err)
	}
}

func TestStaticTokenManager_AccountUUID(t *testing.T) {
	uuid := "my-account-uuid"
	mgr := NewStaticTokenManager("token", uuid)

	if got := mgr.AccountUUID(); got != uuid {
		t.Errorf("expected AccountUUID %q, got %q", uuid, got)
	}
}

func TestStaticTokenManager_ImplementsTokenProvider(t *testing.T) {
	var _ TokenProvider = (*StaticTokenManager)(nil)
}
