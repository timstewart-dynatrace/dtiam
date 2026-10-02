package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtimothystewart/dtiam/pkg/client"
)

// GroupHandler handles group resources.
type GroupHandler struct {
	BaseHandler

	// bindingsPath overrides the account-level bindings path. Tests set it to
	// reach a mock server, since the production path is an absolute URL.
	bindingsPath string
}

// NewGroupHandler creates a new group handler.
func NewGroupHandler(c *client.Client) *GroupHandler {
	return &GroupHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "group",
			Path:      "/groups",
			ListKey:   "items",
			IDField:   "uuid",
			NameField: "name",
			// GET /groups/{uuid} does not exist (only PUT and DELETE do).
			NoSingleGet: true,
		},
	}
}

// GetMembers gets the members of a group.
func (h *GroupHandler) GetMembers(ctx context.Context, groupID string) ([]map[string]any, error) {
	path := fmt.Sprintf("/groups/%s/users", groupID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		return nil, h.handleError("get members", err)
	}

	return h.extractList(body)
}

// GetMemberCount gets the number of members in a group.
func (h *GroupHandler) GetMemberCount(ctx context.Context, groupID string) (int, error) {
	path := fmt.Sprintf("/groups/%s/users", groupID)
	body, err := h.Client.Get(ctx, path, map[string]string{"count": "true"})
	if err != nil {
		return 0, h.handleError("get member count", err)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Try different count field names
	for _, key := range []string{"count", "totalCount", "total"} {
		if count, ok := response[key].(float64); ok {
			return int(count), nil
		}
	}

	// Fall back to listing and counting
	members, err := h.GetMembers(ctx, groupID)
	if err != nil {
		return 0, err
	}
	return len(members), nil
}

// AddMember adds a user to a group.
//
// Membership is managed through the user, not the group: POST /users/{email}
// with a list of group UUIDs adds the user to those groups and leaves existing
// memberships alone. The API has no POST on /groups/{uuid}/users.
func (h *GroupHandler) AddMember(ctx context.Context, groupID, userEmail string) error {
	path := fmt.Sprintf("/users/%s", userEmail)
	_, err := h.Client.Post(ctx, path, []string{groupID})
	if err != nil {
		return h.handleError("add member", err)
	}
	return nil
}

// RemoveMember removes a user from a group. The user may be given by email or
// UID; the API addresses users by email, so a UID is resolved first.
func (h *GroupHandler) RemoveMember(ctx context.Context, groupID, user string) error {
	email, err := NewUserHandler(h.Client).resolveEmail(ctx, user)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/users/%s/groups", email)
	_, err = h.Client.DeleteWithParams(ctx, path, map[string]string{"group-uuid": groupID})
	if err != nil {
		return h.handleError("remove member", err)
	}
	return nil
}

// GetExpanded gets a group with expanded member and policy information.
func (h *GroupHandler) GetExpanded(ctx context.Context, groupID string) (map[string]any, error) {
	group, err := h.Get(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// Get members
	members, err := h.GetMembers(ctx, groupID)
	if err == nil {
		group["members"] = members
		group["member_count"] = len(members)
	}

	// Get policies (via bindings)
	policies, err := h.GetPolicies(ctx, groupID)
	if err == nil {
		group["policy_uuids"] = policies
		group["policy_count"] = len(policies)
	}

	return group, nil
}

// GetPolicies gets the UUIDs of the policies bound to a group at account level.
func (h *GroupHandler) GetPolicies(ctx context.Context, groupID string) ([]string, error) {
	bindings, err := h.bindingHandler().GetForGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(bindings))
	policies := make([]string, 0, len(bindings))
	for _, b := range bindings {
		policyUUID, _ := b["policyUuid"].(string)
		if policyUUID == "" || seen[policyUUID] {
			continue
		}
		seen[policyUUID] = true
		policies = append(policies, policyUUID)
	}
	return policies, nil
}

// bindingHandler returns the account-level binding handler.
func (h *GroupHandler) bindingHandler() *BindingHandler {
	b := NewBindingHandler(h.Client)
	if h.bindingsPath != "" {
		b.Path = h.bindingsPath
	}
	return b
}

// Create creates a new group.
//
// POST /groups takes and returns an array of groups; a bare object is rejected
// with HTTP 500 ("payload.map is not a function"). Verified against a live
// account. The single created group is returned.
func (h *GroupHandler) Create(ctx context.Context, data map[string]any) (map[string]any, error) {
	// Validate required fields
	if _, ok := data["name"]; !ok {
		return nil, fmt.Errorf("name is required")
	}

	body, err := h.Client.Post(ctx, h.Path, []map[string]any{data})
	if err != nil {
		return nil, h.handleError("create", err)
	}
	if len(body) == 0 {
		return data, nil
	}

	var created []map[string]any
	if err := json.Unmarshal(body, &created); err != nil {
		// Tolerate a single-object response should the API ever return one.
		var single map[string]any
		if err := json.Unmarshal(body, &single); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return single, nil
	}
	if len(created) == 0 {
		return data, nil
	}
	return created[0], nil
}

// Update edits a group's name and description. The API replaces the group
// record, so fields absent from data keep their current values.
func (h *GroupHandler) Update(ctx context.Context, groupID string, data map[string]any) (map[string]any, error) {
	current, err := h.Get(ctx, groupID)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"uuid":        groupID,
		"name":        current["name"],
		"description": current["description"],
	}
	for _, field := range []string{"name", "description", "federatedAttributeValues"} {
		if v, ok := data[field]; ok {
			body[field] = v
		}
	}
	if name, _ := body["name"].(string); strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}

	return h.BaseHandler.Update(ctx, groupID, body)
}
