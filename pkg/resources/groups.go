package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jtimothystewart/dtiam/pkg/client"
)

// GroupHandler handles group resources.
type GroupHandler struct {
	BaseHandler
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
func (h *GroupHandler) AddMember(ctx context.Context, groupID, userEmail string) error {
	path := fmt.Sprintf("/groups/%s/users", groupID)
	_, err := h.Client.Post(ctx, path, map[string]string{"email": userEmail})
	if err != nil {
		return h.handleError("add member", err)
	}
	return nil
}

// RemoveMember removes a user from a group.
func (h *GroupHandler) RemoveMember(ctx context.Context, groupID, userID string) error {
	path := fmt.Sprintf("/groups/%s/users/%s", groupID, userID)
	_, err := h.Client.Delete(ctx, path)
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

// GetPolicies gets the policy UUIDs bound to a group.
func (h *GroupHandler) GetPolicies(ctx context.Context, groupID string) ([]string, error) {
	path := fmt.Sprintf("/repo/account/%s/bindings/groups/%s", h.Client.AccountUUID(), groupID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		// Return empty list if no bindings found
		if apiErr, ok := err.(*client.APIError); ok && apiErr.IsNotFound() {
			return []string{}, nil
		}
		return nil, h.handleError("get policies", err)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract policy UUIDs from policyBindings
	var policies []string
	if bindings, ok := response["policyBindings"].([]any); ok {
		for _, binding := range bindings {
			if b, ok := binding.(map[string]any); ok {
				if policyUUID, ok := b["policyUuid"].(string); ok {
					policies = append(policies, policyUUID)
				}
			}
		}
	}

	return policies, nil
}

// Create creates a new group.
func (h *GroupHandler) Create(ctx context.Context, data map[string]any) (map[string]any, error) {
	// Validate required fields
	if _, ok := data["name"]; !ok {
		return nil, fmt.Errorf("name is required")
	}

	return h.BaseHandler.Create(ctx, data)
}
