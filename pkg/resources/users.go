package resources

import (
	"context"
	"fmt"
	"net/url"
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

// Get gets a user by email or UID.
//
// The API addresses users by email only: GET /users/{email} returns the user
// with its group memberships, and GET /users/{uid} is rejected with 400. A UID
// is therefore resolved to an email from the user list first.
func (h *UserHandler) Get(ctx context.Context, user string) (map[string]any, error) {
	email, err := h.resolveEmail(ctx, user)
	if err != nil {
		return nil, err
	}
	return h.BaseHandler.Get(ctx, email)
}

// resolveEmail returns the email for a user given by email or UID.
func (h *UserHandler) resolveEmail(ctx context.Context, user string) (string, error) {
	if strings.Contains(user, "@") {
		return user, nil
	}
	// Include service users: they are group members too, and the plain user
	// list leaves them out.
	items, err := h.ListWithServiceUsers(ctx, nil)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if uid, _ := item["uid"].(string); strings.EqualFold(uid, user) {
			if email, _ := item["email"].(string); email != "" {
				return email, nil
			}
		}
	}
	return "", fmt.Errorf("user not found")
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

// Delete removes a user from the account. The user may be given by email or
// UID; DELETE /users/{email} is the only form the API accepts.
func (h *UserHandler) Delete(ctx context.Context, user string) error {
	email, err := h.resolveEmail(ctx, user)
	if err != nil {
		return err
	}
	return h.BaseHandler.Delete(ctx, email)
}

// GetGroups gets the groups a user belongs to. They are part of the
// GET /users/{email} response; there is no /users/{id}/groups GET.
func (h *UserHandler) GetGroups(ctx context.Context, user string) ([]map[string]any, error) {
	record, err := h.Get(ctx, user)
	if err != nil {
		return nil, h.handleError("get groups", err)
	}
	groups, ok := record["groups"].([]any)
	if !ok {
		return []map[string]any{}, nil
	}
	return toMapSlice(groups)
}

// GetExpanded gets a user with a group count added. The groups themselves are
// already part of the GET /users/{email} response.
func (h *UserHandler) GetExpanded(ctx context.Context, user string) (map[string]any, error) {
	record, err := h.Get(ctx, user)
	if err != nil {
		return nil, err
	}

	groups := []map[string]any{}
	if raw, ok := record["groups"].([]any); ok {
		if converted, err := toMapSlice(raw); err == nil {
			groups = converted
		}
	}
	record["groups"] = groups
	record["group_count"] = len(groups)

	return record, nil
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

// RemoveFromGroups removes a user from specified groups. The API takes the
// groups as repeated group-uuid query parameters, not as a request body.
func (h *UserHandler) RemoveFromGroups(ctx context.Context, email string, groupUUIDs []string) error {
	path := fmt.Sprintf("/users/%s/groups", email)
	query := url.Values{}
	for _, g := range groupUUIDs {
		query.Add("group-uuid", g)
	}
	_, err := h.Client.DeleteWithQuery(ctx, path, query)
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
