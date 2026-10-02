// Package resources provides resource handlers for the Dynatrace IAM API.
package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
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

	// Pagination describes how to page through List results. Nil means the
	// endpoint returns its whole collection in a single response, which is true
	// for most Account Management endpoints.
	Pagination *client.PaginationConfig

	// NoSingleGet marks a collection that has no GET-by-ID endpoint, so Get
	// resolves the item from List instead. Groups, environments (v2) and
	// platform tokens are like this: the API answers {path}/{id} with 404 even
	// for an ID that exists, which made every by-ID lookup report "not found".
	NoSingleGet bool
}

// ResourceName returns the resource name.
func (h *BaseHandler) ResourceName() string {
	return h.Name
}

// APIPath returns the API path.
func (h *BaseHandler) APIPath() string {
	return h.Path
}

// List lists resources, following pagination to completion when the endpoint
// is paginated. Callers always receive the full collection.
func (h *BaseHandler) List(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	if h.Pagination == nil {
		body, err := h.Client.Get(ctx, h.Path, params)
		if err != nil {
			return nil, h.handleError("list", err)
		}
		return h.extractList(body)
	}

	var (
		all     []map[string]any
		pageKey string
	)

	for page := 1; page <= client.MaxPageRequests; page++ {
		body, err := h.Client.Get(ctx, h.Path, h.Pagination.Params(params, page, pageKey))
		if err != nil {
			return nil, h.handleError("list", err)
		}

		items, nextKey, total, err := h.extractPage(body)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)

		// An empty page always terminates: it is the only reliable end marker
		// shared by both paging styles.
		if len(items) == 0 {
			break
		}

		switch h.Pagination.Style {
		case client.PaginationPageKey:
			if nextKey == "" || nextKey == pageKey {
				return all, nil
			}
			pageKey = nextKey
		case client.PaginationPageNumber:
			// total is authoritative when present; otherwise rely on a short
			// final page, then on the empty page above.
			if total > 0 && len(all) >= total {
				return all, nil
			}
			if len(items) < h.Pagination.EffectivePageSize() {
				return all, nil
			}
		case client.PaginationNone:
			return all, nil
		}
	}

	return all, nil
}

// Get gets a single resource by ID.
func (h *BaseHandler) Get(ctx context.Context, id string) (map[string]any, error) {
	if h.NoSingleGet {
		return h.getFromList(ctx, id)
	}

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

// getFromList finds a resource by its ID field in the full collection. The
// "not found" error matches handleError's wording, which GetOrResolve relies on
// to fall back to a name search.
func (h *BaseHandler) getFromList(ctx context.Context, id string) (map[string]any, error) {
	items, err := h.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if itemID, ok := item[h.IDField].(string); ok && strings.EqualFold(itemID, id) {
			return item, nil
		}
	}
	return nil, fmt.Errorf("%s not found", h.Name)
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

	// Try common list keys. The Account Management API is not consistent about
	// this: "items" (groups, users), "results" (service users, platform tokens,
	// limits), "data" (environments, subscriptions), and "content" (boundaries)
	// all appear. Verified against a live account -- the documentation states
	// "items" for several endpoints that do not use it.
	keys := []string{h.ListKey, "items", "results", "data", "content", h.Name + "s", h.Name}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if items, ok := response[key]; ok {
			return toMapSlice(items)
		}
	}

	// Some endpoints return a single resource rather than a collection when
	// exactly one matches. Recognize that by the handler's own identity fields
	// and wrap it, instead of reporting an empty list.
	if h.looksLikeSingleResource(response) {
		return []map[string]any{response}, nil
	}

	// Return empty slice if no items found
	return []map[string]any{}, nil
}

// looksLikeSingleResource reports whether a response body is one resource rather
// than a collection envelope, judged by the handler's own ID and name fields.
func (h *BaseHandler) looksLikeSingleResource(response map[string]any) bool {
	for _, field := range []string{h.IDField, h.NameField} {
		if field == "" {
			continue
		}
		if _, ok := response[field]; ok {
			return true
		}
	}
	return false
}

// extractPage extracts one page of items plus the paging metadata needed to
// decide whether another request is required.
func (h *BaseHandler) extractPage(body []byte) (items []map[string]any, nextPageKey string, total int, err error) {
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		// A bare array is inherently a single, complete page.
		var direct []map[string]any
		if err := json.Unmarshal(body, &direct); err != nil {
			return nil, "", 0, fmt.Errorf("failed to parse response: %w", err)
		}
		return direct, "", 0, nil
	}

	p := h.Pagination

	// Resolve the items key, preferring the one the pagination config declares.
	keys := []string{}
	if p != nil && p.ItemsKey != "" {
		keys = append(keys, p.ItemsKey)
	}
	keys = append(keys, h.ListKey, "items", "results", "data", "content", h.Name+"s", h.Name)

	for _, key := range keys {
		if key == "" {
			continue
		}
		raw, ok := response[key]
		if !ok {
			continue
		}
		items, err = toMapSlice(raw)
		if err != nil {
			return nil, "", 0, err
		}
		break
	}

	// A single-resource response is a complete, one-element page.
	if len(items) == 0 && h.looksLikeSingleResource(response) {
		items = []map[string]any{response}
	}

	if p != nil {
		if p.NextKeyField != "" {
			if s, ok := response[p.NextKeyField].(string); ok {
				nextPageKey = s
			}
		}
		if p.TotalField != "" {
			if n, ok := toInt(response[p.TotalField]); ok {
				total = n
			}
		}
	}

	return items, nextPageKey, total, nil
}

// toInt converts a JSON number to an int. JSON numbers decode as float64, but
// accept the integer types too so the helper is safe for hand-built maps in tests.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
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

// apiError is a user-facing error that keeps the underlying *client.APIError
// reachable through errors.As, so callers can still branch on the status code.
type apiError struct {
	msg string
	err *client.APIError
}

func (e *apiError) Error() string { return e.msg }
func (e *apiError) Unwrap() error { return e.err }

// handleError maps API errors to user-friendly errors.
func (h *BaseHandler) handleError(operation string, err error) error {
	if apiErr, ok := err.(*client.APIError); ok {
		var msg string
		switch {
		case apiErr.IsNotFound():
			msg = fmt.Sprintf("%s not found", h.Name)
		case apiErr.IsPermissionDenied():
			msg = fmt.Sprintf("permission denied: %s", apiErr.Message)
		case apiErr.IsConflict():
			msg = fmt.Sprintf("conflict: %s", apiErr.Message)
		default:
			msg = fmt.Sprintf("failed to %s %s: %s", operation, h.Name, apiErr.Message)
		}
		return &apiError{msg: msg, err: apiErr}
	}
	return fmt.Errorf("failed to %s %s: %w", operation, h.Name, err)
}

// isNotAnID reports whether a failed GET-by-ID means "no resource with that
// ID" rather than a real failure: 404, or 400 for an identifier that is not
// even shaped like an ID (the policy and boundary endpoints answer a name with
// "Validation failed (uuid is expected)").
func isNotAnID(err error) bool {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsNotFound() || apiErr.StatusCode == http.StatusBadRequest
	}
	return strings.Contains(err.Error(), "not found")
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

	// If the identifier is not a known ID, search the list: it may be a name,
	// or the API may not support direct GET by ID at all.
	if isNotAnID(err) {
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
