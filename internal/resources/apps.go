package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtimothystewart/dtiam/internal/client"
)

// AppHandler handles App Engine Registry resources.
// Apps are retrieved from environment-specific endpoints.
// Requires an environment URL and the `app-engine:apps:run` scope.
type AppHandler struct {
	Client         *client.Client
	EnvironmentURL string
}

// NewAppHandler creates a new app handler.
// environmentURL can be an env-id (e.g., "abc12345") or full URL.
func NewAppHandler(c *client.Client, environmentURL string) *AppHandler {
	// Normalize the URL
	envURL := strings.TrimRight(environmentURL, "/")
	if !strings.HasPrefix(envURL, "http") {
		envURL = fmt.Sprintf("https://%s.apps.dynatrace.com", envURL)
	}

	return &AppHandler{
		Client:         c,
		EnvironmentURL: envURL,
	}
}

// ResourceName returns the resource name.
func (h *AppHandler) ResourceName() string {
	return "app"
}

// APIPath returns the full API path for apps.
func (h *AppHandler) APIPath() string {
	return fmt.Sprintf("%s/platform/app-engine/registry/v1/apps", h.EnvironmentURL)
}

// List lists all apps from the App Engine Registry.
func (h *AppHandler) List(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	body, err := h.Client.Get(ctx, h.APIPath(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		var items []map[string]any
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return items, nil
	}

	for _, key := range []string{"apps", "items"} {
		if items, ok := response[key]; ok {
			return toMapSlice(items)
		}
	}

	return []map[string]any{}, nil
}

// Get gets a single app by ID.
func (h *AppHandler) Get(ctx context.Context, appID string) (map[string]any, error) {
	path := fmt.Sprintf("%s/%s", h.APIPath(), appID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetIDs returns all app IDs (useful for policy statements).
func (h *AppHandler) GetIDs(ctx context.Context) ([]string, error) {
	apps, err := h.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(apps))
	for _, app := range apps {
		if id, ok := app["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
