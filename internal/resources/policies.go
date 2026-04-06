package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jtimothystewart/dtiam/internal/client"
)

// PolicyHandler handles policy resources.
type PolicyHandler struct {
	BaseHandler
	LevelType string
	LevelID   string
}

// NewPolicyHandler creates a new policy handler for account-level policies.
func NewPolicyHandler(c *client.Client) *PolicyHandler {
	return NewPolicyHandlerWithLevel(c, "account", c.AccountUUID())
}

// NewPolicyHandlerWithLevel creates a new policy handler for a specific level.
func NewPolicyHandlerWithLevel(c *client.Client, levelType, levelID string) *PolicyHandler {
	// Policies use the /repo/ endpoint which is NOT under /accounts/{uuid}/
	path := fmt.Sprintf("%s/%s/%s/policies", client.RepoBasePath, levelType, levelID)
	return &PolicyHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "policy",
			Path:      path,
			ListKey:   "policies",
			IDField:   "uuid",
			NameField: "name",
		},
		LevelType: levelType,
		LevelID:   levelID,
	}
}

// Create creates a new policy.
func (h *PolicyHandler) Create(ctx context.Context, data map[string]any) (map[string]any, error) {
	// Validate required fields
	if _, ok := data["name"]; !ok {
		return nil, fmt.Errorf("name is required")
	}
	if _, ok := data["statementQuery"]; !ok {
		return nil, fmt.Errorf("statementQuery is required")
	}

	return h.BaseHandler.Create(ctx, data)
}

// ListAllLevels lists policies from account, environments, and global levels.
func (h *PolicyHandler) ListAllLevels(ctx context.Context) ([]map[string]any, error) {
	var allPolicies []map[string]any

	// Account level
	accountHandler := NewPolicyHandlerWithLevel(h.Client, "account", h.Client.AccountUUID())
	accountPolicies, err := accountHandler.List(ctx, nil)
	if err == nil {
		for _, p := range accountPolicies {
			p["_level_type"] = "account"
			p["_level_id"] = h.Client.AccountUUID()
			allPolicies = append(allPolicies, p)
		}
	}

	// Global level
	globalHandler := NewPolicyHandlerWithLevel(h.Client, "global", "global")
	globalPolicies, err := globalHandler.List(ctx, nil)
	if err == nil {
		for _, p := range globalPolicies {
			p["_level_type"] = "global"
			p["_level_id"] = "global"
			allPolicies = append(allPolicies, p)
		}
	}

	return allPolicies, nil
}

// ListAggregate lists all policies including inherited ones.
func (h *PolicyHandler) ListAggregate(ctx context.Context) ([]map[string]any, error) {
	path := fmt.Sprintf("%s/aggregate", h.Path)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		return nil, h.handleError("list aggregate", err)
	}

	return h.extractList(body)
}

// Validate validates a policy definition.
func (h *PolicyHandler) Validate(ctx context.Context, data map[string]any) (*ValidationResult, error) {
	path := fmt.Sprintf("%s/validation", h.Path)
	body, err := h.Client.Post(ctx, path, data)

	result := &ValidationResult{
		Valid:  err == nil,
		Errors: []string{},
	}

	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok {
			result.StatusCode = apiErr.StatusCode
			result.Errors = append(result.Errors, apiErr.Message)
		} else {
			result.Errors = append(result.Errors, err.Error())
		}
		return result, nil
	}

	// Parse response
	var response map[string]any
	if err := json.Unmarshal(body, &response); err == nil {
		if valid, ok := response["valid"].(bool); ok {
			result.Valid = valid
		}
		if errors, ok := response["errors"].([]any); ok {
			for _, e := range errors {
				if errStr, ok := e.(string); ok {
					result.Errors = append(result.Errors, errStr)
				}
			}
		}
	}

	return result, nil
}

// ValidateUpdate validates an update to an existing policy.
func (h *PolicyHandler) ValidateUpdate(ctx context.Context, policyID string, data map[string]any) (*ValidationResult, error) {
	path := fmt.Sprintf("%s/validation/%s", h.Path, policyID)
	body, err := h.Client.Post(ctx, path, data)

	result := &ValidationResult{
		Valid:  err == nil,
		Errors: []string{},
	}

	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok {
			result.StatusCode = apiErr.StatusCode
			result.Errors = append(result.Errors, apiErr.Message)
		} else {
			result.Errors = append(result.Errors, err.Error())
		}
		return result, nil
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err == nil {
		if valid, ok := response["valid"].(bool); ok {
			result.Valid = valid
		}
	}

	return result, nil
}

// ValidationResult represents the result of a policy validation.
type ValidationResult struct {
	Valid      bool
	Errors     []string
	StatusCode int
}
