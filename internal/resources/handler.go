// Package resources provides resource handlers for the Dynatrace IAM API.
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtimothystewart/dtiam/internal/client"
)

// Handler is the base interface for all resource handlers.
type Handler interface {
	// ResourceName returns the human-readable resource name.
	ResourceName() string
	// APIPath returns the base API path for this resource.
	APIPath() string
}

// Lister can list resources.
type Lister interface {
	Handler
	List(ctx context.Context, params map[string]string) ([]map[string]any, error)
}

// Getter can get a single resource.
type Getter interface {
	Handler
	Get(ctx context.Context, id string) (map[string]any, error)
}

// NameGetter can get a resource by name.
type NameGetter interface {
	Handler
	GetByName(ctx context.Context, name string) (map[string]any, error)
}

// Creator can create resources.
type Creator interface {
	Handler
	Create(ctx context.Context, data map[string]any) (map[string]any, error)
}

// Updater can update resources.
type Updater interface {
	Handler
	Update(ctx context.Context, id string, data map[string]any) (map[string]any, error)
}

// Deleter can delete resources.
type Deleter interface {
	Handler
	Delete(ctx context.Context, id string) error
}

// ExistsChecker can check if a resource exists.
type ExistsChecker interface {
	Handler
	Exists(ctx context.Context, id string) bool
}

// CRUDHandler combines all CRUD operations.
type CRUDHandler interface {
	Lister
	Getter
	NameGetter
	Creator
	Updater
	Deleter
	ExistsChecker
}

// BaseHandler provides common functionality for resource handlers.
type BaseHandler struct {
	Client    *client.Client
	Name      string
	Path      string
	ListKey   string
	IDField   string
	NameField string
}

// ResourceName returns the resource name.
func (h *BaseHandler) ResourceName() string {
	return h.Name
}

// APIPath returns the API path.
func (h *BaseHandler) APIPath() string {
	return h.Path
}

// List lists resources.
func (h *BaseHandler) List(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	body, err := h.Client.Get(ctx, h.Path, params)
	if err != nil {
		return nil, h.handleError("list", err)
	}

	return h.extractList(body)
}

// Get gets a single resource by ID.
func (h *BaseHandler) Get(ctx context.Context, id string) (map[string]any, error) {
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

// GetByName gets a resource by name (client-side search).
func (h *BaseHandler) GetByName(ctx context.Context, name string) (map[string]any, error) {
	items, err := h.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	nameField := h.NameField
	if nameField == "" {
		nameField = "name"
	}

	for _, item := range items {
		if itemName, ok := item[nameField].(string); ok {
			if strings.EqualFold(itemName, name) {
				return item, nil
			}
		}
	}

	return nil, nil
}

// Create creates a new resource.
func (h *BaseHandler) Create(ctx context.Context, data map[string]any) (map[string]any, error) {
	body, err := h.Client.Post(ctx, h.Path, data)
	if err != nil {
		return nil, h.handleError("create", err)
	}

	if len(body) == 0 {
		return data, nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// Update updates an existing resource.
func (h *BaseHandler) Update(ctx context.Context, id string, data map[string]any) (map[string]any, error) {
	path := fmt.Sprintf("%s/%s", h.Path, id)
	body, err := h.Client.Put(ctx, path, data)
	if err != nil {
		return nil, h.handleError("update", err)
	}

	if len(body) == 0 {
		return data, nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// Delete deletes a resource.
func (h *BaseHandler) Delete(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s/%s", h.Path, id)
	_, err := h.Client.Delete(ctx, path)
	if err != nil {
		return h.handleError("delete", err)
	}
	return nil
}

// Exists checks if a resource exists.
func (h *BaseHandler) Exists(ctx context.Context, id string) bool {
	_, err := h.Get(ctx, id)
	return err == nil
}

// extractList extracts a list from the API response.
func (h *BaseHandler) extractList(body []byte) ([]map[string]any, error) {
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		// Try parsing as array directly
		var items []map[string]any
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return items, nil
	}

	// Try common list keys
	keys := []string{h.ListKey, "items", h.Name + "s", h.Name}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if items, ok := response[key]; ok {
			return toMapSlice(items)
		}
	}

	// Return empty slice if no items found
	return []map[string]any{}, nil
}

// toMapSlice converts an interface to []map[string]any.
func toMapSlice(v any) ([]map[string]any, error) {
	switch items := v.(type) {
	case []any:
		result := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				result = append(result, m)
			}
		}
		return result, nil
	case []map[string]any:
		return items, nil
	default:
		return nil, fmt.Errorf("unexpected type %T for list", v)
	}
}

// handleError maps API errors to user-friendly errors.
func (h *BaseHandler) handleError(operation string, err error) error {
	if apiErr, ok := err.(*client.APIError); ok {
		switch {
		case apiErr.IsNotFound():
			return fmt.Errorf("%s not found", h.Name)
		case apiErr.IsPermissionDenied():
			return fmt.Errorf("permission denied: %s", apiErr.Message)
		case apiErr.IsConflict():
			return fmt.Errorf("conflict: %s", apiErr.Message)
		default:
			return fmt.Errorf("failed to %s %s: %s", operation, h.Name, apiErr.Message)
		}
	}
	return fmt.Errorf("failed to %s %s: %w", operation, h.Name, err)
}

// GetOrResolve gets a resource by ID or name.
// It first tries the direct GET endpoint, then falls back to searching the list.
func GetOrResolve(ctx context.Context, h interface {
	Getter
	NameGetter
	Lister
}, identifier string) (map[string]any, error) {
	// Try as ID first via direct API call
	result, err := h.Get(ctx, identifier)
	if err == nil {
		return result, nil
	}

	// If not found (or other error), search the list
	// Some APIs don't support direct GET by ID but do return the ID in list
	isNotFound := false
	if apiErr, ok := err.(*client.APIError); ok && apiErr.IsNotFound() {
		isNotFound = true
	} else if strings.Contains(err.Error(), "not found") {
		isNotFound = true
	}

	if isNotFound {
		// Search the list for the resource
		items, listErr := h.List(ctx, nil)
		if listErr != nil {
			return nil, listErr
		}

		// Search by UUID/ID first (common fields: uuid, uid, id)
		for _, item := range items {
			for _, idField := range []string{"uuid", "uid", "id"} {
				if id, ok := item[idField].(string); ok && strings.EqualFold(id, identifier) {
					return item, nil
				}
			}
		}

		// Then search by name
		for _, item := range items {
			if name, ok := item["name"].(string); ok && strings.EqualFold(name, identifier) {
				return item, nil
			}
		}

		// Not found in list either
		return nil, nil
	}

	// Return original error for non-404 errors
	return nil, err
}

// Resolve resolves an identifier to a resource (by ID or name).
func (h *BaseHandler) Resolve(ctx context.Context, identifier string) (map[string]any, error) {
	// Try as ID first
	result, err := h.Get(ctx, identifier)
	if err == nil && result != nil {
		return result, nil
	}

	// Try as name
	result, err = h.GetByName(ctx, identifier)
	if err == nil && result != nil {
		return result, nil
	}

	return nil, fmt.Errorf("%s not found: %s", h.Name, identifier)
}
