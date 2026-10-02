package auth

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func jwt(t *testing.T, claims map[string]any) string {
	t.Helper()
	b, _ := json.Marshal(claims)
	return "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(b) + ".sig"
}

func TestIdentityFromToken(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantSub   string
		wantEmail string
		wantAud   string
		wantErr   bool
	}{
		{
			name:    "should read sub, email and a string audience",
			token:   jwt(t, map[string]any{"sub": "u1", "email": "a@b.c", "aud": "dt0s02.X", "scope": "s1 s2"}),
			wantSub: "u1", wantEmail: "a@b.c", wantAud: "dt0s02.X",
		},
		{
			name:    "should accept a Bearer prefix and a list audience",
			token:   "Bearer " + jwt(t, map[string]any{"sub": "u1", "aud": []any{"dt0s02.Y"}}),
			wantSub: "u1", wantAud: "dt0s02.Y",
		},
		{
			name:    "should fall back to preferred_username for the email",
			token:   jwt(t, map[string]any{"sub": "u1", "preferred_username": "x@service.sso.dynatrace.com"}),
			wantSub: "u1", wantEmail: "x@service.sso.dynatrace.com",
		},
		{name: "should reject a non-JWT token", token: "dt0c01.ABC.DEF", wantErr: true},
		{name: "should reject a token without a subject", token: jwt(t, map[string]any{"email": "a@b.c"}), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := IdentityFromToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id.Subject != tt.wantSub || id.Email != tt.wantEmail || id.ClientID != tt.wantAud {
				t.Errorf("got %+v", id)
			}
		})
	}
}
