package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jtimothystewart/dtiam/internal/client"
)

// BindingHandler handles policy binding resources.
type BindingHandler struct {
	BaseHandler
	LevelType string
	LevelID   string
}

// NewBindingHandler creates a new binding handler for account-level bindings.
func NewBindingHandler(c *client.Client) *BindingHandler {
	return NewBindingHandlerWithLevel(c, "account", c.AccountUUID())
}

// NewBindingHandlerWithLevel creates a new binding handler for a specific level.
func NewBindingHandlerWithLevel(c *client.Client, levelType, levelID string) *BindingHandler {
	// Bindings use the /repo/ endpoint which is NOT under /accounts/{uuid}/
	path := fmt.Sprintf("%s/%s/%s/bindings", client.RepoBasePath, levelType, levelID)
	return &BindingHandler{
		BaseHandler: BaseHandler{
			Client:  c,
			Name:    "binding",
			Path:    path,
			ListKey: "policyBindings",
			IDField: "policyUuid",
		},
		LevelType: levelType,
		LevelID:   levelID,
	}
}

// List lists bindings, flattening the structure so each policy-group combination is separate.
func (h *BindingHandler) List(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	body, err := h.Client.Get(ctx, h.Path, params)
	if err != nil {
		return nil, h.handleError("list", err)
	}

	return h.flattenBindings(body)
}

// ListRaw returns the raw binding structure from the API.
func (h *BindingHandler) ListRaw(ctx context.Context, params map[string]string) (map[string]any, error) {
	body, err := h.Client.Get(ctx, h.Path, params)
	if err != nil {
		return nil, h.handleError("list", err)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}

// GetForGroup gets bindings for a specific group.
func (h *BindingHandler) GetForGroup(ctx context.Context, groupID string) ([]map[string]any, error) {
	path := fmt.Sprintf("%s/groups/%s", h.Path, groupID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.IsNotFound() {
			return []map[string]any{}, nil
		}
		return nil, h.handleError("get for group", err)
	}

	return h.flattenBindings(body)
}

// Create creates a new binding. Parameters is optional and can be nil.
func (h *BindingHandler) Create(ctx context.Context, groupUUID, policyUUID string, boundaries []string, parameters map[string]string) (map[string]any, error) {
	binding := map[string]any{
		"policyUuid": policyUUID,
		"groups":     []string{groupUUID},
	}
	if len(boundaries) > 0 {
		binding["boundaries"] = boundaries
	}
	if len(parameters) > 0 {
		binding["parameters"] = parameters
	}

	data := map[string]any{
		"policyBindings": []any{binding},
	}

	body, err := h.Client.Post(ctx, h.Path, data)
	if err != nil {
		return nil, h.handleError("create", err)
	}

	if len(body) == 0 {
		result := map[string]any{
			"groupUuid":  groupUUID,
			"policyUuid": policyUUID,
			"boundaries": boundaries,
			"levelType":  h.LevelType,
			"levelId":    h.LevelID,
		}
		if len(parameters) > 0 {
			result["parameters"] = parameters
		}
		return result, nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// Delete deletes a binding by removing the group from the policy's binding.
func (h *BindingHandler) Delete(ctx context.Context, groupUUID, policyUUID string) error {
	// Get current bindings
	raw, err := h.ListRaw(ctx, nil)
	if err != nil {
		return err
	}

	bindings, ok := raw["policyBindings"].([]any)
	if !ok {
		return fmt.Errorf("invalid bindings structure")
	}

	// Find and update the binding
	var found bool
	for i, b := range bindings {
		binding, ok := b.(map[string]any)
		if !ok {
			continue
		}

		if binding["policyUuid"] != policyUUID {
			continue
		}

		groups, ok := binding["groups"].([]any)
		if !ok {
			continue
		}

		// Remove the group
		newGroups := make([]string, 0, len(groups))
		for _, g := range groups {
			if gStr, ok := g.(string); ok && gStr != groupUUID {
				newGroups = append(newGroups, gStr)
			}
		}

		if len(newGroups) == len(groups) {
			continue // Group not found in this binding
		}

		found = true

		if len(newGroups) == 0 {
			// Remove the entire binding
			bindings = append(bindings[:i], bindings[i+1:]...)
		} else {
			binding["groups"] = newGroups
		}
		break
	}

	if !found {
		return fmt.Errorf("binding not found")
	}

	// Update bindings
	_, err = h.Client.Put(ctx, h.Path, map[string]any{"policyBindings": bindings})
	return err
}

// GetForPolicy gets bindings for a specific policy.
func (h *BindingHandler) GetForPolicy(ctx context.Context, policyUUID string) (map[string]any, error) {
	path := fmt.Sprintf("%s/%s", h.Path, policyUUID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		return nil, h.handleError("get for policy", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetPolicyGroupBinding gets a specific binding.
func (h *BindingHandler) GetPolicyGroupBinding(ctx context.Context, policyUUID, groupUUID string) (map[string]any, error) {
	path := fmt.Sprintf("%s/%s/%s", h.Path, policyUUID, groupUUID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		return nil, h.handleError("get binding", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetDescendants gets bindings from child levels.
func (h *BindingHandler) GetDescendants(ctx context.Context, policyUUID string) ([]map[string]any, error) {
	path := fmt.Sprintf("%s/descendants/%s", h.Path, policyUUID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.IsNotFound() {
			return []map[string]any{}, nil
		}
		return nil, h.handleError("get descendants", err)
	}

	return h.flattenBindings(body)
}

// UpdateGroupBindings replaces all bindings for a group.
func (h *BindingHandler) UpdateGroupBindings(ctx context.Context, groupUUID string, policyBindings []map[string]any) error {
	path := fmt.Sprintf("%s/groups/%s", h.Path, groupUUID)
	_, err := h.Client.Put(ctx, path, map[string]any{"policyBindings": policyBindings})
	if err != nil {
		return h.handleError("update group bindings", err)
	}
	return nil
}

// AddBoundary adds a boundary to a binding.
func (h *BindingHandler) AddBoundary(ctx context.Context, groupUUID, policyUUID, boundaryUUID string) error {
	binding, err := h.GetPolicyGroupBinding(ctx, policyUUID, groupUUID)
	if err != nil {
		return err
	}

	boundaries := []string{boundaryUUID}
	if existing, ok := binding["boundaries"].([]any); ok {
		for _, b := range existing {
			if bStr, ok := b.(string); ok {
				boundaries = append(boundaries, bStr)
			}
		}
	}

	binding["boundaries"] = boundaries
	path := fmt.Sprintf("%s/%s/%s", h.Path, policyUUID, groupUUID)
	_, err = h.Client.Put(ctx, path, binding)
	return err
}

// RemoveBoundary removes a boundary from a binding.
func (h *BindingHandler) RemoveBoundary(ctx context.Context, groupUUID, policyUUID, boundaryUUID string) error {
	binding, err := h.GetPolicyGroupBinding(ctx, policyUUID, groupUUID)
	if err != nil {
		return err
	}

	var newBoundaries []string
	if existing, ok := binding["boundaries"].([]any); ok {
		for _, b := range existing {
			if bStr, ok := b.(string); ok && bStr != boundaryUUID {
				newBoundaries = append(newBoundaries, bStr)
			}
		}
	}

	binding["boundaries"] = newBoundaries
	path := fmt.Sprintf("%s/%s/%s", h.Path, policyUUID, groupUUID)
	_, err = h.Client.Put(ctx, path, binding)
	return err
}

// flattenBindings flattens the policyBindings structure.
func (h *BindingHandler) flattenBindings(body []byte) ([]map[string]any, error) {
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	bindings, ok := response["policyBindings"].([]any)
	if !ok {
		return []map[string]any{}, nil
	}

	var result []map[string]any
	for _, b := range bindings {
		binding, ok := b.(map[string]any)
		if !ok {
			continue
		}

		policyUUID, _ := binding["policyUuid"].(string)
		boundaries := []string{}
		if b, ok := binding["boundaries"].([]any); ok {
			for _, boundary := range b {
				if bStr, ok := boundary.(string); ok {
					boundaries = append(boundaries, bStr)
				}
			}
		}

		groups, ok := binding["groups"].([]any)
		if !ok {
			continue
		}

		for _, g := range groups {
			groupUUID, ok := g.(string)
			if !ok {
				continue
			}

			result = append(result, map[string]any{
				"policyUuid": policyUUID,
				"groupUuid":  groupUUID,
				"boundaries": boundaries,
				"levelType":  h.LevelType,
				"levelId":    h.LevelID,
			})
		}
	}

	return result, nil
}
