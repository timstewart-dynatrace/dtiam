package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Identity is who an access token authenticates as, read from its claims.
type Identity struct {
	// Subject is the user or service user UID (the "sub" claim).
	Subject string `json:"subject" yaml:"subject"`
	// Email is the identity's email; for a service user it is
	// {uid}@service.sso.dynatrace.com.
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
	// ClientID is the OAuth client the token was issued to (the "aud" claim).
	ClientID string `json:"clientId,omitempty" yaml:"clientId,omitempty"`
	// Scopes the token carries.
	Scopes []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// IdentityFromToken reads the identity from a JWT access token without
// verifying its signature. It is for reporting who dtiam acts as, never for an
// access decision: the API verifies the token on every request.
func IdentityFromToken(token string) (*Identity, error) {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("the access token is not a JWT, so the identity cannot be read from it")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode access token claims: %w", err)
	}
	var claims struct {
		Sub      string `json:"sub"`
		Email    string `json:"email"`
		Username string `json:"preferred_username"`
		Aud      any    `json:"aud"`
		Scope    string `json:"scope"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse access token claims: %w", err)
	}
	if claims.Sub == "" {
		return nil, fmt.Errorf("the access token has no subject claim")
	}

	id := &Identity{Subject: claims.Sub, Email: claims.Email}
	if id.Email == "" && strings.Contains(claims.Username, "@") {
		id.Email = claims.Username
	}
	switch aud := claims.Aud.(type) {
	case string:
		id.ClientID = aud
	case []any:
		if len(aud) > 0 {
			id.ClientID, _ = aud[0].(string)
		}
	}
	if claims.Scope != "" {
		id.Scopes = strings.Fields(claims.Scope)
	}
	return id, nil
}
