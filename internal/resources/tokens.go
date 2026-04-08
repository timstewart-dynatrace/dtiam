package resources

import (
	"context"
	"fmt"

	"github.com/jtimothystewart/dtiam/internal/client"
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
			Client:    c,
			Name:      "platform-token",
			Path:      "/platform-tokens",
			ListKey:   "items",
			IDField:   "id",
			NameField: "name",
		},
	}
}

// Create creates a new platform token.
// The token value is only returned once during creation and cannot be retrieved later.
func (h *TokenHandler) Create(ctx context.Context, name string, scopes []string, expiresIn string) (map[string]any, error) {
	data := map[string]any{
		"name": name,
	}

	if len(scopes) > 0 {
		data["scopes"] = scopes
	}
	if expiresIn != "" {
		data["expiresIn"] = expiresIn
	}

	return h.BaseHandler.Create(ctx, data)
}

// Get gets a platform token by ID.
// Falls back to searching the list if the direct endpoint returns 404.
func (h *TokenHandler) Get(ctx context.Context, tokenID string) (map[string]any, error) {
	result, err := h.BaseHandler.Get(ctx, tokenID)
	if err == nil {
		return result, nil
	}

	// Fall back to searching the list
	if apiErr, ok := err.(*client.APIError); ok && apiErr.IsNotFound() {
		tokens, listErr := h.List(ctx, nil)
		if listErr != nil {
			return nil, listErr
		}
		for _, token := range tokens {
			if id, ok := token["id"].(string); ok && id == tokenID {
				return token, nil
			}
		}
		return nil, fmt.Errorf("platform-token not found: %s", tokenID)
	}

	return nil, err
}
