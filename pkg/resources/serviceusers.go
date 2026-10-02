package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

// ServiceUserHandler handles service user (OAuth client) resources.
type ServiceUserHandler struct {
	BaseHandler
}

// NewServiceUserHandler creates a new service user handler.
func NewServiceUserHandler(c *client.Client) *ServiceUserHandler {
	return &ServiceUserHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "service-user",
			Path:      "/service-users",
			ListKey:   "results",
			IDField:   "uid",
			NameField: "name",
			// The service user API paginates and returns {results, nextPageKey,
			// totalCount} -- not the {count, items} shape used elsewhere.
			Pagination: client.ServiceUserPagination(),
		},
	}
}

// GetByName gets a service user by name.
func (h *ServiceUserHandler) GetByName(ctx context.Context, name string) (map[string]any, error) {
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

// Create creates a new service user, then adds it to the given groups.
//
// POST /service-users accepts only name and description; group membership is
// managed through the user endpoints using the service user's email, as it is
// for any other user.
func (h *ServiceUserHandler) Create(ctx context.Context, name string, description *string, groups []string) (map[string]any, error) {
	data := map[string]any{
		"name": name,
	}
	if description != nil {
		data["description"] = *description
	}

	created, err := h.BaseHandler.Create(ctx, data)
	if err != nil {
		return nil, err
	}

	if len(groups) > 0 {
		email, err := h.emailFor(ctx, created)
		if err != nil {
			return created, fmt.Errorf("service user created but not added to groups: %w", err)
		}
		if err := NewUserHandler(h.Client).AddToGroups(ctx, email, groups); err != nil {
			return created, fmt.Errorf("service user created but not added to groups: %w", err)
		}
		created["groups"] = groups
	}

	return created, nil
}

// Update updates a service user's name and description, and replaces its
// group memberships when groups is non-nil.
//
// PUT /service-users/{uid} requires name and accepts only name and
// description, so an unchanged field is filled from the current record.
func (h *ServiceUserHandler) Update(ctx context.Context, userID string, name, description *string, groups []string) (map[string]any, error) {
	current, err := h.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"name":        current["name"],
		"description": current["description"],
	}
	if name != nil {
		data["name"] = *name
	}
	if description != nil {
		data["description"] = *description
	}

	updated := current
	if name != nil || description != nil {
		updated, err = h.BaseHandler.Update(ctx, userID, data)
		if err != nil {
			return nil, err
		}
	}

	if groups != nil {
		email, err := h.emailFor(ctx, current)
		if err != nil {
			return nil, err
		}
		if err := NewUserHandler(h.Client).ReplaceGroups(ctx, email, groups); err != nil {
			return nil, err
		}
		updated["groups"] = groups
	}

	return updated, nil
}

// GetGroups gets the groups a service user belongs to. GET /service-users/{uid}
// does not include them; GET /users/{email} does, for service users as well.
func (h *ServiceUserHandler) GetGroups(ctx context.Context, userID string) ([]map[string]any, error) {
	email, err := h.emailForID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return NewUserHandler(h.Client).GetGroups(ctx, email)
}

// GetExpanded gets a service user with expanded group information.
func (h *ServiceUserHandler) GetExpanded(ctx context.Context, userID string) (map[string]any, error) {
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

// AddToGroup adds a service user to a group, leaving other memberships alone.
func (h *ServiceUserHandler) AddToGroup(ctx context.Context, userID, groupUUID string) error {
	email, err := h.emailForID(ctx, userID)
	if err != nil {
		return err
	}
	return NewUserHandler(h.Client).AddToGroups(ctx, email, []string{groupUUID})
}

// RemoveFromGroup removes a service user from one group.
func (h *ServiceUserHandler) RemoveFromGroup(ctx context.Context, userID, groupUUID string) error {
	email, err := h.emailForID(ctx, userID)
	if err != nil {
		return err
	}
	return NewUserHandler(h.Client).RemoveFromGroups(ctx, email, []string{groupUUID})
}

// emailForID returns the email of the service user with the given UID.
func (h *ServiceUserHandler) emailForID(ctx context.Context, userID string) (string, error) {
	user, err := h.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	return h.emailFor(ctx, user)
}

// emailFor returns a service user's email, the identifier the user endpoints
// require. Service user emails have the form {uid}@service.sso.dynatrace.com.
func (h *ServiceUserHandler) emailFor(ctx context.Context, user map[string]any) (string, error) {
	if email, _ := user["email"].(string); email != "" {
		return email, nil
	}
	uid, _ := user["uid"].(string)
	if uid == "" {
		return "", fmt.Errorf("service user has neither email nor uid")
	}
	// A create response may carry only the uid; fetch the full record.
	full, err := h.Get(ctx, uid)
	if err != nil {
		return "", err
	}
	if email, _ := full["email"].(string); email != "" {
		return email, nil
	}
	return "", fmt.Errorf("service user %s has no email", uid)
}
