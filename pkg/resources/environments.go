package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

// EnvironmentHandler handles environment resources.
type EnvironmentHandler struct {
	BaseHandler
}

// NewEnvironmentHandler creates a new environment handler.
func NewEnvironmentHandler(c *client.Client) *EnvironmentHandler {
	// Environment API uses a different base URL
	baseURL := fmt.Sprintf("%s/%s/environments", client.EnvBaseURL, c.AccountUUID())
	return &EnvironmentHandler{
		BaseHandler: BaseHandler{
			Client: c,
			Name:   "environment",
			Path:   baseURL,
			// The API responds with {data: [...]}, not {tenants: [...]}.
			// Verified against a live account.
			ListKey:   "data",
			IDField:   "id",
			NameField: "name",
			// env/v2 has no GET /environments/{id}; it answers 404 for every ID.
			NoSingleGet: true,
		},
	}
}

// GetByName gets an environment by name.
func (h *EnvironmentHandler) GetByName(ctx context.Context, name string) (map[string]any, error) {
	items, err := h.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if itemName, ok := item["name"].(string); ok {
			if strings.EqualFold(itemName, name) {
				return item, nil
			}
		}
	}

	return nil, nil
}
