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
			Client:    c,
			Name:      "environment",
			Path:      baseURL,
			ListKey:   "tenants",
			IDField:   "id",
			NameField: "name",
		},
	}
}

// List lists environments.
func (h *EnvironmentHandler) List(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	body, err := h.Client.Get(ctx, h.Path, params)
	if err != nil {
		return nil, h.handleError("list", err)
	}

	return h.extractList(body)
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

// extractList overrides the base to handle environment-specific response formats.
func (h *EnvironmentHandler) extractList(body []byte) ([]map[string]any, error) {
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		var items []map[string]any
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return items, nil
	}

	// Try common list keys for environments
	for _, key := range []string{"tenants", "environments", "items"} {
		if items, ok := response[key]; ok {
			return toMapSlice(items)
		}
	}

	return []map[string]any{}, nil
}
