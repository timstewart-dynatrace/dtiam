package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtimothystewart/dtiam/internal/client"
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
		},
	}
}

// Get gets an environment by ID.
func (h *EnvironmentHandler) Get(ctx context.Context, id string) (map[string]any, error) {
	path := fmt.Sprintf("%s/%s", h.Path, id)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		return nil, h.handleError("get", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
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
