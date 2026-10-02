package resources

import (
	"context"

	"github.com/jtimothystewart/dtiam/pkg/client"
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
