package resources

import (
	"context"
	"strings"

	"github.com/jtimothystewart/dtiam/internal/client"
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
			ListKey:   "items",
			IDField:   "uid",
			NameField: "name",
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

// Create creates a new service user.
func (h *ServiceUserHandler) Create(ctx context.Context, name string, description *string, groups []string) (map[string]any, error) {
	data := map[string]any{
		"name": name,
	}

	if description != nil {
		data["description"] = *description
	}
	if len(groups) > 0 {
		data["groups"] = groups
	}

	return h.BaseHandler.Create(ctx, data)
}

// Update updates a service user.
func (h *ServiceUserHandler) Update(ctx context.Context, userID string, name, description *string, groups []string) (map[string]any, error) {
	data := make(map[string]any)

	if name != nil {
		data["name"] = *name
	}
	if description != nil {
		data["description"] = *description
	}
	if groups != nil {
		data["groups"] = groups
	}

	return h.BaseHandler.Update(ctx, userID, data)
}

// GetGroups gets the groups a service user belongs to.
func (h *ServiceUserHandler) GetGroups(ctx context.Context, userID string) ([]map[string]any, error) {
	user, err := h.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	groups, ok := user["groups"].([]any)
	if !ok {
		return []map[string]any{}, nil
	}

	result := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		switch v := g.(type) {
		case string:
			// Group is a UUID, create minimal object
			result = append(result, map[string]any{"uuid": v})
		case map[string]any:
			result = append(result, v)
		}
	}

	return result, nil
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

// AddToGroup adds a service user to a group.
func (h *ServiceUserHandler) AddToGroup(ctx context.Context, userID, groupUUID string) error {
	user, err := h.Get(ctx, userID)
	if err != nil {
		return err
	}

	// Get current groups
	var currentGroups []string
	if groups, ok := user["groups"].([]any); ok {
		for _, g := range groups {
			switch v := g.(type) {
			case string:
				currentGroups = append(currentGroups, v)
			case map[string]any:
				if uuid, ok := v["uuid"].(string); ok {
					currentGroups = append(currentGroups, uuid)
				}
			}
		}
	}

	// Add new group
	currentGroups = append(currentGroups, groupUUID)

	// Update
	_, err = h.Update(ctx, userID, nil, nil, currentGroups)
	return err
}

// RemoveFromGroup removes a service user from a group.
func (h *ServiceUserHandler) RemoveFromGroup(ctx context.Context, userID, groupUUID string) error {
	user, err := h.Get(ctx, userID)
	if err != nil {
		return err
	}

	// Get current groups and filter out the one to remove
	var newGroups []string
	if groups, ok := user["groups"].([]any); ok {
		for _, g := range groups {
			var uuid string
			switch v := g.(type) {
			case string:
				uuid = v
			case map[string]any:
				if u, ok := v["uuid"].(string); ok {
					uuid = u
				}
			}
			if uuid != "" && uuid != groupUUID {
				newGroups = append(newGroups, uuid)
			}
		}
	}

	// Update
	_, err = h.Update(ctx, userID, nil, nil, newGroups)
	return err
}

