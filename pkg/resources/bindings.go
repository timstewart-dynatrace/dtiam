package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jtimothystewart/dtiam/pkg/client"
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
//
// GET .../bindings/groups/{uuid} does not use the policyBindings envelope the
// other binding endpoints share. It returns {"policyUuids": [...]}, plus a
// "bindingsDetails" list with boundaries when ?details=true is passed.
// Verified against a live account.
func (h *BindingHandler) GetForGroup(ctx context.Context, groupID string) ([]map[string]any, error) {
	path := fmt.Sprintf("%s/groups/%s", h.Path, groupID)
	body, err := h.Client.Get(ctx, path, map[string]string{"details": "true"})
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.IsNotFound() {
			return []map[string]any{}, nil
		}
		return nil, h.handleError("get for group", err)
	}

	var response struct {
		PolicyUUIDs     []string `json:"policyUuids"`
		BindingsDetails []any    `json:"bindingsDetails"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.BindingsDetails) > 0 {
		return h.flattenBindingList(response.BindingsDetails, groupID), nil
	}

	// Without details, each policy UUID is one binding with no boundaries.
	result := make([]map[string]any, 0, len(response.PolicyUUIDs))
	for _, policyUUID := range response.PolicyUUIDs {
		result = append(result, h.bindingRow(policyUUID, groupID, []string{}))
	}
	return result, nil
}

// Create creates a new binding. Parameters is optional and can be nil.
//
// Uses POST .../bindings/{policy}/{group}, which appends a binding for that
// pair. The API has no POST on the level-wide .../bindings collection.
func (h *BindingHandler) Create(ctx context.Context, groupUUID, policyUUID string, boundaries []string, parameters map[string]string) (map[string]any, error) {
	data := map[string]any{}
	if len(boundaries) > 0 {
		data["boundaries"] = boundaries
	}
	if len(parameters) > 0 {
		data["parameters"] = parameters
	}

	path := fmt.Sprintf("%s/%s/%s", h.Path, policyUUID, groupUUID)
	body, err := h.Client.Post(ctx, path, data)
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

// Delete removes the binding between one group and one policy.
//
// Uses DELETE .../bindings/{policy}/{group}, which touches only that pair.
// The previous implementation read every binding at the level and PUT the
// edited set back -- an endpoint the API does not document, and a lost update
// for any binding changed by someone else in between.
func (h *BindingHandler) Delete(ctx context.Context, groupUUID, policyUUID string) error {
	path := fmt.Sprintf("%s/%s/%s", h.Path, policyUUID, groupUUID)
	if _, err := h.Client.Delete(ctx, path); err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.IsNotFound() {
			return fmt.Errorf("binding not found")
		}
		return h.handleError("delete", err)
	}
	return nil
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

// UpdateGroupBindings replaces the set of policies bound to a group. The API
// overwrites the group's bindings with exactly these policy UUIDs, and takes
// {"policyUuids": [...]} -- not the policyBindings envelope.
func (h *BindingHandler) UpdateGroupBindings(ctx context.Context, groupUUID string, policyUUIDs []string) error {
	path := fmt.Sprintf("%s/groups/%s", h.Path, groupUUID)
	_, err := h.Client.Put(ctx, path, map[string]any{"policyUuids": policyUUIDs})
	if err != nil {
		return h.handleError("update group bindings", err)
	}
	return nil
}

// AddBoundary adds a boundary to a binding, keeping its existing boundaries
// and parameters.
func (h *BindingHandler) AddBoundary(ctx context.Context, groupUUID, policyUUID, boundaryUUID string) error {
	return h.updateBoundaries(ctx, groupUUID, policyUUID, func(current []string) []string {
		for _, b := range current {
			if b == boundaryUUID {
				return current
			}
		}
		return append(current, boundaryUUID)
	})
}

// RemoveBoundary removes one boundary from a binding, keeping the others.
func (h *BindingHandler) RemoveBoundary(ctx context.Context, groupUUID, policyUUID, boundaryUUID string) error {
	return h.updateBoundaries(ctx, groupUUID, policyUUID, func(current []string) []string {
		kept := make([]string, 0, len(current))
		for _, b := range current {
			if b != boundaryUUID {
				kept = append(kept, b)
			}
		}
		return kept
	})
}

// updateBoundaries rewrites the boundary list of a single group/policy binding.
//
// GET .../bindings/{policy}/{group} wraps the binding in a
// {levelType, levelId, policyBindings: [...]} envelope, while PUT on the same
// path takes the bare {boundaries, parameters, metadata}. Reading boundaries
// from the top level of the GET response found none, so attach replaced every
// existing boundary with the new one and detach removed them all -- widening
// access. The current boundaries now come from the binding inside the envelope.
func (h *BindingHandler) updateBoundaries(ctx context.Context, groupUUID, policyUUID string, change func([]string) []string) error {
	raw, err := h.GetPolicyGroupBinding(ctx, policyUUID, groupUUID)
	if err != nil {
		return err
	}

	bindings, _ := raw["policyBindings"].([]any)
	switch len(bindings) {
	case 0:
		return fmt.Errorf("binding not found")
	case 1:
	default:
		// Parameterized policies can bind the same pair several times with
		// different parameters. Picking one would silently leave the others.
		return fmt.Errorf("group %s has %d bindings to policy %s (parameterized); "+
			"boundary changes for parameterized bindings are not supported", groupUUID, len(bindings), policyUUID)
	}

	binding, _ := bindings[0].(map[string]any)
	current := []string{}
	if existing, ok := binding["boundaries"].([]any); ok {
		for _, b := range existing {
			if s, ok := b.(string); ok {
				current = append(current, s)
			}
		}
	}

	body := map[string]any{"boundaries": change(current)}
	for _, field := range []string{"parameters", "metadata"} {
		if v, ok := binding[field]; ok {
			body[field] = v
		}
	}

	path := fmt.Sprintf("%s/%s/%s", h.Path, policyUUID, groupUUID)
	if _, err := h.Client.Put(ctx, path, body); err != nil {
		return h.handleError("update boundaries", err)
	}
	return nil
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
	return h.flattenBindingList(bindings, ""), nil
}

// flattenBindingList turns bindings of the form {policyUuid, groups, boundaries}
// into one row per policy/group pair. A non-empty onlyGroup keeps just that
// group's rows.
func (h *BindingHandler) flattenBindingList(bindings []any, onlyGroup string) []map[string]any {
	result := []map[string]any{}
	for _, b := range bindings {
		binding, ok := b.(map[string]any)
		if !ok {
			continue
		}

		policyUUID, _ := binding["policyUuid"].(string)
		boundaries := []string{}
		if bs, ok := binding["boundaries"].([]any); ok {
			for _, boundary := range bs {
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
			if !ok || (onlyGroup != "" && groupUUID != onlyGroup) {
				continue
			}
			result = append(result, h.bindingRow(policyUUID, groupUUID, boundaries))
		}
	}
	return result
}

// bindingRow builds one flattened binding row.
func (h *BindingHandler) bindingRow(policyUUID, groupUUID string, boundaries []string) map[string]any {
	return map[string]any{
		"policyUuid": policyUUID,
		"groupUuid":  groupUUID,
		"boundaries": boundaries,
		"levelType":  h.LevelType,
		"levelId":    h.LevelID,
	}
}
