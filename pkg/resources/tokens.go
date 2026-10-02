package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

// TokenHandler handles platform token resources.
// Platform tokens provide API access credentials for automation.
// Requires the `platform-token:tokens:manage` scope.
type TokenHandler struct {
	BaseHandler
}

// NewTokenHandler creates a new platform token handler.
func NewTokenHandler(c *client.Client) *TokenHandler {
	return &TokenHandler{
		BaseHandler: BaseHandler{
			Client:  c,
			Name:    "platform-token",
			Path:    "/platform-tokens",
			ListKey: "results",
			// The API field is tokenId; "id" does not exist in the response, so
			// get/delete by ID could never resolve. Verified live.
			IDField:   "tokenId",
			NameField: "name",
			// The platform token API paginates and returns
			// {pageSize, pageNumber, total, results}.
			Pagination: client.PlatformTokenPagination(),
			// Tokens have only DELETE on /platform-tokens/{id}, no GET.
			NoSingleGet: true,
		},
	}
}

// PlatformTokenRequest describes a platform token to create. Every field is
// required by POST /platform-tokens.
type PlatformTokenRequest struct {
	Name string
	// Scopes are the permissions the token carries, e.g. "storage:logs:read".
	Scopes []string
	// Resources are the URNs the token may be used against:
	// "urn:dtaccount:{uuid}" or "urn:dtenvironment:{id}".
	Resources []string
	// Tags are free-form labels; may be empty but is always sent.
	Tags []string
	// ExpirationDate is RFC 3339.
	ExpirationDate string
	// UserUUID is the UID of the user or service user that owns the token.
	UserUUID string
}

// Create creates a new platform token. The token value is returned only once,
// in the response's "token" field, and cannot be retrieved later.
//
// Before 3.2.0 this sent {name, scopes, expiresIn}, which the API rejects: it
// requires name, scope, resource, tags, expirationDate and userUuid.
func (h *TokenHandler) Create(ctx context.Context, req PlatformTokenRequest) (map[string]any, error) {
	switch {
	case req.Name == "":
		return nil, fmt.Errorf("token name is required")
	case len(req.Scopes) == 0:
		return nil, fmt.Errorf("at least one scope is required")
	case len(req.Resources) == 0:
		return nil, fmt.Errorf("at least one resource is required")
	case req.UserUUID == "":
		return nil, fmt.Errorf("the owning user UUID is required")
	}
	if _, err := time.Parse(time.RFC3339, req.ExpirationDate); err != nil {
		return nil, fmt.Errorf("invalid expiration date %q: use RFC 3339, e.g. 2027-01-01T00:00:00Z", req.ExpirationDate)
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	return h.BaseHandler.Create(ctx, map[string]any{
		"name":           req.Name,
		"scope":          req.Scopes,
		"resource":       req.Resources,
		"tags":           tags,
		"expirationDate": req.ExpirationDate,
		"userUuid":       req.UserUUID,
	})
}

// AccountResource returns the platform token resource URN for an account.
func AccountResource(accountUUID string) string { return "urn:dtaccount:" + accountUUID }

// EnvironmentResource returns the platform token resource URN for an environment.
func EnvironmentResource(environmentID string) string { return "urn:dtenvironment:" + environmentID }

// ParseTokenLifetime converts a lifetime such as "30d", "12h" or "1y" into an
// RFC 3339 expiration date relative to now. Go durations ("90m") are accepted too.
func ParseTokenLifetime(lifetime string, now time.Time) (string, error) {
	lifetime = strings.TrimSpace(lifetime)
	if len(lifetime) >= 2 {
		n, err := strconv.Atoi(lifetime[:len(lifetime)-1])
		if err == nil && n > 0 {
			switch lifetime[len(lifetime)-1] {
			case 'd':
				return now.AddDate(0, 0, n).UTC().Format(time.RFC3339), nil
			case 'w':
				return now.AddDate(0, 0, 7*n).UTC().Format(time.RFC3339), nil
			case 'y':
				return now.AddDate(n, 0, 0).UTC().Format(time.RFC3339), nil
			}
		}
	}
	if d, err := time.ParseDuration(lifetime); err == nil && d > 0 {
		return now.Add(d).UTC().Format(time.RFC3339), nil
	}
	return "", fmt.Errorf("invalid lifetime %q: use e.g. 30d, 2w, 1y or 12h", lifetime)
}

// Platform token statuses accepted by SetStatus.
const (
	TokenStatusActive   = "ACTIVE"
	TokenStatusInactive = "INACTIVE"
)

// SetStatus activates or deactivates a platform token with
// PUT /platform-tokens/{id}/status. An inactive token stops authenticating but
// keeps its ID and settings, so it can be reactivated, unlike a deleted one.
func (h *TokenHandler) SetStatus(ctx context.Context, tokenID, status string) error {
	if status != TokenStatusActive && status != TokenStatusInactive {
		return fmt.Errorf("invalid token status %q, want %s or %s", status, TokenStatusActive, TokenStatusInactive)
	}
	path := fmt.Sprintf("%s/%s/status", h.Path, tokenID)
	if _, err := h.Client.Put(ctx, path, map[string]string{"status": status}); err != nil {
		return h.handleError("set status of", err)
	}
	return nil
}

// SetExpiration changes when a platform token expires, with
// PUT /platform-tokens/{id}/expiration-date. The date must be RFC 3339, e.g.
// 2027-01-01T00:00:00Z.
func (h *TokenHandler) SetExpiration(ctx context.Context, tokenID, expirationDate string) error {
	if _, err := time.Parse(time.RFC3339, expirationDate); err != nil {
		return fmt.Errorf("invalid expiration date %q: use RFC 3339, e.g. 2027-01-01T00:00:00Z", expirationDate)
	}
	path := fmt.Sprintf("%s/%s/expiration-date", h.Path, tokenID)
	if _, err := h.Client.Put(ctx, path, map[string]string{"expirationDate": expirationDate}); err != nil {
		return h.handleError("set expiration of", err)
	}
	return nil
}
