package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtimothystewart/dtiam/internal/client"
)

// SchemaHandler handles Settings 2.0 schema resources.
// Schemas are retrieved from environment-specific endpoints.
// Requires an environment URL and a settings.read scope.
type SchemaHandler struct {
	Client         *client.Client
	EnvironmentURL string
}

// NewSchemaHandler creates a new schema handler.
// environmentURL can be an env-id (e.g., "abc12345") or full URL.
func NewSchemaHandler(c *client.Client, environmentURL string) *SchemaHandler {
	envURL := strings.TrimRight(environmentURL, "/")
	if !strings.HasPrefix(envURL, "http") {
		envURL = fmt.Sprintf("https://%s.live.dynatrace.com", envURL)
	}

	return &SchemaHandler{
		Client:         c,
		EnvironmentURL: envURL,
	}
}

// ResourceName returns the resource name.
func (h *SchemaHandler) ResourceName() string {
	return "schema"
}

// APIPath returns the full API path for schemas.
func (h *SchemaHandler) APIPath() string {
	return fmt.Sprintf("%s/api/v2/settings/schemas", h.EnvironmentURL)
}

// List lists all settings schemas from the Environment API.
func (h *SchemaHandler) List(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	body, err := h.Client.Get(ctx, h.APIPath(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		var items []map[string]any
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return items, nil
	}

	for _, key := range []string{"items", "schemas"} {
		if items, ok := response[key]; ok {
			return toMapSlice(items)
		}
	}

	return []map[string]any{}, nil
}

// Get gets a single schema by ID.
func (h *SchemaHandler) Get(ctx context.Context, schemaID string) (map[string]any, error) {
	path := fmt.Sprintf("%s/%s", h.APIPath(), schemaID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetIDs returns all schema IDs.
func (h *SchemaHandler) GetIDs(ctx context.Context) ([]string, error) {
	schemas, err := h.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(schemas))
	for _, schema := range schemas {
		if id, ok := schema["schemaId"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// GetBuiltinIDs returns only builtin schema IDs (starting with "builtin:").
func (h *SchemaHandler) GetBuiltinIDs(ctx context.Context) ([]string, error) {
	all, err := h.GetIDs(ctx)
	if err != nil {
		return nil, err
	}

	builtin := make([]string, 0, len(all))
	for _, id := range all {
		if strings.HasPrefix(id, "builtin:") {
			builtin = append(builtin, id)
		}
	}
	return builtin, nil
}

// Search searches schemas by ID or display name pattern (case-insensitive).
func (h *SchemaHandler) Search(ctx context.Context, pattern string) ([]map[string]any, error) {
	schemas, err := h.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	patternLower := strings.ToLower(pattern)
	var results []map[string]any
	for _, s := range schemas {
		id := strings.ToLower(stringFrom(s, "schemaId"))
		name := strings.ToLower(stringFrom(s, "displayName"))
		if strings.Contains(id, patternLower) || strings.Contains(name, patternLower) {
			results = append(results, s)
		}
	}
	return results, nil
}
