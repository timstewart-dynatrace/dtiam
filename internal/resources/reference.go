package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jtimothystewart/dtiam/internal/client"
)

// ReferenceHandler handles the reference data API, which reports what values the
// account accepts rather than what it currently contains.
//
// Requires the `account-env-read` scope. The permissions list it returns is the
// authoritative set of valid `permissionName` values for the permission
// management API, so it is the right way to validate a grant before attempting
// it instead of relying on a hardcoded list.
type ReferenceHandler struct {
	Client *client.Client

	// BaseURL is the reference API root. It defaults to client.RefBaseURL and is
	// overridable so tests can point at a local server; this endpoint is not
	// account-scoped, so it cannot inherit the client's base URL.
	BaseURL string
}

// NewReferenceHandler creates a new reference data handler.
func NewReferenceHandler(c *client.Client) *ReferenceHandler {
	return &ReferenceHandler{Client: c, BaseURL: client.RefBaseURL}
}

// ResourceName returns the resource name.
func (h *ReferenceHandler) ResourceName() string { return "permission" }

// APIPath returns the reference permissions path.
func (h *ReferenceHandler) APIPath() string {
	base := h.BaseURL
	if base == "" {
		base = client.RefBaseURL
	}
	return base + "/permissions"
}

// ListPermissions returns every permission that can be assigned to a group.
//
// The response is a bare JSON array of {id, description}, not the {count, items}
// envelope the rest of the Account Management API uses.
func (h *ReferenceHandler) ListPermissions(ctx context.Context) ([]map[string]any, error) {
	body, err := h.Client.Get(ctx, h.APIPath(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list available permissions: %w", err)
	}

	var items []map[string]any
	if err := json.Unmarshal(body, &items); err == nil {
		return items, nil
	}

	// Tolerate a wrapped envelope in case the API gains one.
	var wrapped map[string]any
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse permissions response: %w", err)
	}
	for _, key := range []string{"permissions", "items", "results"} {
		if raw, ok := wrapped[key]; ok {
			return toMapSlice(raw)
		}
	}

	return []map[string]any{}, nil
}

// PermissionIDs returns just the permission identifiers, for validation.
func (h *ReferenceHandler) PermissionIDs(ctx context.Context) ([]string, error) {
	items, err := h.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		if id, ok := item["id"].(string); ok && id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
