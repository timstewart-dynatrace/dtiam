package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtimothystewart/dtiam/pkg/client"
)

// UserHandler handles user resources.
type UserHandler struct {
	BaseHandler
}

// NewUserHandler creates a new user handler.
func NewUserHandler(c *client.Client) *UserHandler {
	return &UserHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "user",
			Path:      "/users",
			ListKey:   "items",
			IDField:   "uid",
			NameField: "email",
		},
	}
}

// List lists users.
func (h *UserHandler) List(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	return h.BaseHandler.List(ctx, params)
}

// ListWithServiceUsers lists users including service users.
func (h *UserHandler) ListWithServiceUsers(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	if params == nil {
		params = make(map[string]string)
	}
	params["service-users"] = "true"
	return h.BaseHandler.List(ctx, params)
}

// GetByEmail gets a user by email address.
func (h *UserHandler) GetByEmail(ctx context.Context, email string) (map[string]any, error) {
	items, err := h.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if itemEmail, ok := item["email"].(string); ok {
			if strings.EqualFold(itemEmail, email) {
				return item, nil
			}
		}
	}

	return nil, nil
}

// GetByName gets a user by email (alias for GetByEmail).
func (h *UserHandler) GetByName(ctx context.Context, name string) (map[string]any, error) {
	return h.GetByEmail(ctx, name)
}

// Create creates a new user.
func (h *UserHandler) Create(ctx context.Context, email string, firstName, lastName *string, groups []string) (map[string]any, error) {
	data := map[string]any{
		"email": email,
	}

	if firstName != nil {
		data["name"] = *firstName
	}
	if lastName != nil {
		data["surname"] = *lastName
	}
	if len(groups) > 0 {
		data["groups"] = groups
	}

	return h.BaseHandler.Create(ctx, data)
}

// Delete deletes a user.
func (h *UserHandler) Delete(ctx context.Context, userID string) error {
	return h.BaseHandler.Delete(ctx, userID)
}

// GetGroups gets the groups a user belongs to.
func (h *UserHandler) GetGroups(ctx context.Context, userID string) ([]map[string]any, error) {
	path := fmt.Sprintf("/users/%s/groups", userID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		// Fall back to getting groups from user object
		user, err := h.Get(ctx, userID)
		if err != nil {
			return nil, h.handleError("get groups", err)
		}
		if groups, ok := user["groups"].([]any); ok {
			return toMapSlice(groups)
		}
		return []map[string]any{}, nil
	}

	return h.extractList(body)
}

// GetExpanded gets a user with expanded group information.
func (h *UserHandler) GetExpanded(ctx context.Context, userID string) (map[string]any, error) {
	user, err := h.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	groups, err := h.GetGroups(ctx, userID)
	if err == nil {
		user["groups"] = groups
		user["group_count"] = len(groups)
	}

	return user, nil
}

// ReplaceGroups replaces all group memberships for a user.
func (h *UserHandler) ReplaceGroups(ctx context.Context, email string, groupUUIDs []string) error {
	path := fmt.Sprintf("/users/%s/groups", email)
	_, err := h.Client.Put(ctx, path, groupUUIDs)
	if err != nil {
		return h.handleError("replace groups", err)
	}
	return nil
}

// RemoveFromGroups removes a user from specified groups.
func (h *UserHandler) RemoveFromGroups(ctx context.Context, email string, groupUUIDs []string) error {
	path := fmt.Sprintf("/users/%s/groups", email)
	_, err := h.Client.DeleteWithBody(ctx, path, groupUUIDs)
	if err != nil {
		return h.handleError("remove from groups", err)
	}
	return nil
}

// AddToGroups adds a user to specified groups.
func (h *UserHandler) AddToGroups(ctx context.Context, email string, groupUUIDs []string) error {
	path := fmt.Sprintf("/users/%s", email)
	_, err := h.Client.Post(ctx, path, groupUUIDs)
	if err != nil {
		return h.handleError("add to groups", err)
	}
	return nil
}

// extractList overrides the base to handle user-specific response formats.
func (h *UserHandler) extractList(body []byte) ([]map[string]any, error) {
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		var items []map[string]any
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return items, nil
	}

	// Try common list keys
	for _, key := range []string{"items", "users", "groups"} {
		if items, ok := response[key]; ok {
			return toMapSlice(items)
		}
	}

	return []map[string]any{}, nil
}
