// Package utils provides utility functions for the dtiam CLI.
package utils

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jtimothystewart/dtiam/internal/client"
	"github.com/jtimothystewart/dtiam/internal/resources"
)

// Permission represents a parsed permission from a policy statement.
type Permission struct {
	Effect      string `json:"effect" yaml:"effect"`
	Action      string `json:"action" yaml:"action"`
	Description string `json:"description" yaml:"description"`
	Conditions  string `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// PermissionSource tracks where a permission came from.
type PermissionSource struct {
	Group    string `json:"group,omitempty" yaml:"group,omitempty"`
	Policy   string `json:"policy" yaml:"policy"`
	Boundary string `json:"boundary,omitempty" yaml:"boundary,omitempty"`
}

// EffectivePermission represents an aggregated permission with its sources.
type EffectivePermission struct {
	Effect      string             `json:"effect" yaml:"effect"`
	Action      string             `json:"action" yaml:"action"`
	Description string             `json:"description" yaml:"description"`
	Sources     []PermissionSource `json:"sources" yaml:"sources"`
}

// Common permission patterns in Dynatrace IAM.
var permissionPatterns = map[string]string{
	"settings:objects:read":    "Read settings objects",
	"settings:objects:write":   "Write settings objects",
	"settings:schemas:read":    "Read settings schemas",
	"environment:roles:manage": "Manage environment roles",
	"account:users:read":       "Read account users",
	"account:users:write":      "Write account users",
	"account:groups:read":      "Read account groups",
	"account:groups:write":     "Write account groups",
	"account:policies:read":    "Read account policies",
	"account:policies:write":   "Write account policies",
}

// ParseStatementQuery parses a policy statement query into structured permissions.
func ParseStatementQuery(statement string) []Permission {
	var permissions []Permission

	// Split by semicolons for multiple statements
	statements := strings.Split(statement, ";")

	// Regex for ALLOW/DENY statements
	re := regexp.MustCompile(`(?i)(ALLOW|DENY)\s+([^\s]+(?:\s*,\s*[^\s]+)*)\s*(?:WHERE\s+(.+))?`)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		match := re.FindStringSubmatch(stmt)
		if match == nil {
			continue
		}

		effect := strings.ToUpper(match[1])
		actionsStr := match[2]
		conditions := ""
		if len(match) > 3 {
			conditions = strings.TrimSpace(match[3])
		}

		// Parse actions
		actions := strings.Split(actionsStr, ",")
		for _, action := range actions {
			action = strings.TrimSpace(action)
			if action == "" {
				continue
			}

			perm := Permission{
				Effect:      effect,
				Action:      action,
				Description: permissionPatterns[action],
			}
			if perm.Description == "" {
				perm.Description = action
			}
			if conditions != "" {
				perm.Conditions = conditions
			}
			permissions = append(permissions, perm)
		}
	}

	return permissions
}

// PermissionsCalculator calculates effective permissions for users and groups.
type PermissionsCalculator struct {
	client *client.Client
}

// NewPermissionsCalculator creates a new permissions calculator.
func NewPermissionsCalculator(c *client.Client) *PermissionsCalculator {
	return &PermissionsCalculator{client: c}
}

// UserEffectivePermissions represents the result of calculating user permissions.
type UserEffectivePermissions struct {
	User struct {
		UID   string `json:"uid" yaml:"uid"`
		Email string `json:"email" yaml:"email"`
	} `json:"user" yaml:"user"`
	Groups               []map[string]any      `json:"groups" yaml:"groups"`
	GroupCount           int                   `json:"group_count" yaml:"group_count"`
	Bindings             []map[string]any      `json:"bindings" yaml:"bindings"`
	BindingCount         int                   `json:"binding_count" yaml:"binding_count"`
	EffectivePermissions []EffectivePermission `json:"effective_permissions" yaml:"effective_permissions"`
	PermissionCount      int                   `json:"permission_count" yaml:"permission_count"`
	Error                string                `json:"error,omitempty" yaml:"error,omitempty"`
}

// GetUserEffectivePermissions calculates effective permissions for a user.
func (pc *PermissionsCalculator) GetUserEffectivePermissions(ctx context.Context, userID string) (*UserEffectivePermissions, error) {
	userHandler := resources.NewUserHandler(pc.client)
	policyHandler := resources.NewPolicyHandler(pc.client)
	bindingHandler := resources.NewBindingHandler(pc.client)

	result := &UserEffectivePermissions{}

	// Resolve user
	var user map[string]any
	var err error
	if strings.Contains(userID, "@") {
		user, err = userHandler.GetByEmail(ctx, userID)
	} else {
		user, err = userHandler.Get(ctx, userID)
	}

	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	result.User.UID = StringFrom(user, "uid")
	if email, ok := user["email"].(string); ok {
		result.User.Email = email
	}

	// Get user's groups
	groups, err := userHandler.GetGroups(ctx, result.User.UID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	result.Groups = groups
	result.GroupCount = len(groups)

	// Collect all policy bindings from all groups
	var allBindings []map[string]any
	uniquePermissions := make(map[string]*EffectivePermission)

	for _, group := range groups {
		groupID := StringFrom(group, "uuid")
		groupName := ""
		if name, ok := group["name"].(string); ok {
			groupName = name
		} else if name, ok := group["groupName"].(string); ok {
			groupName = name
		}

		// Get bindings for this group
		bindings, err := bindingHandler.GetForGroup(ctx, groupID)
		if err != nil {
			continue
		}

		for _, binding := range bindings {
			policyUUID := StringFrom(binding, "policyUuid")
			policy, err := policyHandler.Get(ctx, policyUUID)
			if err != nil || policy == nil {
				continue
			}

			statement := ""
			if s, ok := policy["statementQuery"].(string); ok {
				statement = s
			}
			permissions := ParseStatementQuery(statement)

			policyName := ""
			if name, ok := policy["name"].(string); ok {
				policyName = name
			}

			bindingInfo := map[string]any{
				"group_uuid":  groupID,
				"group_name":  groupName,
				"policy_uuid": policyUUID,
				"policy_name": policyName,
				"permissions": permissions,
			}
			if boundary, ok := binding["boundaryUuid"]; ok {
				bindingInfo["boundary"] = boundary
			}
			allBindings = append(allBindings, bindingInfo)

			// Aggregate permissions
			for _, perm := range permissions {
				key := fmt.Sprintf("%s:%s", perm.Effect, perm.Action)
				if existing, ok := uniquePermissions[key]; ok {
					existing.Sources = append(existing.Sources, PermissionSource{
						Group:  groupName,
						Policy: policyName,
					})
				} else {
					uniquePermissions[key] = &EffectivePermission{
						Effect:      perm.Effect,
						Action:      perm.Action,
						Description: perm.Description,
						Sources: []PermissionSource{{
							Group:  groupName,
							Policy: policyName,
						}},
					}
				}
			}
		}
	}

	result.Bindings = allBindings
	result.BindingCount = len(allBindings)

	for _, perm := range uniquePermissions {
		result.EffectivePermissions = append(result.EffectivePermissions, *perm)
	}
	result.PermissionCount = len(uniquePermissions)

	return result, nil
}

// GroupEffectivePermissions represents the result of calculating group permissions.
type GroupEffectivePermissions struct {
	Group struct {
		UUID string `json:"uuid" yaml:"uuid"`
		Name string `json:"name" yaml:"name"`
	} `json:"group" yaml:"group"`
	Bindings             []map[string]any      `json:"bindings" yaml:"bindings"`
	BindingCount         int                   `json:"binding_count" yaml:"binding_count"`
	EffectivePermissions []EffectivePermission `json:"effective_permissions" yaml:"effective_permissions"`
	PermissionCount      int                   `json:"permission_count" yaml:"permission_count"`
	Error                string                `json:"error,omitempty" yaml:"error,omitempty"`
}

// GetGroupEffectivePermissions calculates effective permissions for a group.
func (pc *PermissionsCalculator) GetGroupEffectivePermissions(ctx context.Context, groupID string) (*GroupEffectivePermissions, error) {
	groupHandler := resources.NewGroupHandler(pc.client)
	policyHandler := resources.NewPolicyHandler(pc.client)
	bindingHandler := resources.NewBindingHandler(pc.client)

	result := &GroupEffectivePermissions{}

	// Resolve group
	group, err := groupHandler.Resolve(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("group not found: %s", groupID)
	}

	result.Group.UUID = StringFrom(group, "uuid")
	if name, ok := group["name"].(string); ok {
		result.Group.Name = name
	}

	// Get bindings for this group
	bindings, err := bindingHandler.GetForGroup(ctx, result.Group.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group bindings: %w", err)
	}

	result.Bindings = bindings
	result.BindingCount = len(bindings)

	uniquePermissions := make(map[string]*EffectivePermission)

	for _, binding := range bindings {
		policyUUID := StringFrom(binding, "policyUuid")
		policy, err := policyHandler.Get(ctx, policyUUID)
		if err != nil || policy == nil {
			continue
		}

		statement := ""
		if s, ok := policy["statementQuery"].(string); ok {
			statement = s
		}
		permissions := ParseStatementQuery(statement)

		policyName := ""
		if name, ok := policy["name"].(string); ok {
			policyName = name
		}

		boundary := ""
		if b, ok := binding["boundaryUuid"].(string); ok {
			boundary = b
		}

		for _, perm := range permissions {
			key := fmt.Sprintf("%s:%s", perm.Effect, perm.Action)
			if existing, ok := uniquePermissions[key]; ok {
				existing.Sources = append(existing.Sources, PermissionSource{
					Policy:   policyName,
					Boundary: boundary,
				})
			} else {
				uniquePermissions[key] = &EffectivePermission{
					Effect:      perm.Effect,
					Action:      perm.Action,
					Description: perm.Description,
					Sources: []PermissionSource{{
						Policy:   policyName,
						Boundary: boundary,
					}},
				}
			}
		}
	}

	for _, perm := range uniquePermissions {
		result.EffectivePermissions = append(result.EffectivePermissions, *perm)
	}
	result.PermissionCount = len(uniquePermissions)

	return result, nil
}

// PermissionsMatrix generates permissions matrix for policies and groups.
type PermissionsMatrix struct {
	client *client.Client
}

// NewPermissionsMatrix creates a new permissions matrix generator.
func NewPermissionsMatrix(c *client.Client) *PermissionsMatrix {
	return &PermissionsMatrix{client: c}
}

// MatrixResult represents the result of generating a permissions matrix.
type MatrixResult struct {
	Permissions     []string         `json:"permissions" yaml:"permissions"`
	Policies        []string         `json:"policies,omitempty" yaml:"policies,omitempty"`
	Groups          []string         `json:"groups,omitempty" yaml:"groups,omitempty"`
	Matrix          []map[string]any `json:"matrix" yaml:"matrix"`
	PolicyCount     int              `json:"policy_count,omitempty" yaml:"policy_count,omitempty"`
	GroupCount      int              `json:"group_count,omitempty" yaml:"group_count,omitempty"`
	PermissionCount int              `json:"permission_count" yaml:"permission_count"`
}

// GeneratePolicyMatrix generates a matrix of policies and their permissions.
func (pm *PermissionsMatrix) GeneratePolicyMatrix(ctx context.Context) (*MatrixResult, error) {
	policyHandler := resources.NewPolicyHandler(pm.client)

	policies, err := policyHandler.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	// Collect all unique permissions
	allPermissions := make(map[string]bool)
	policyPermissions := make(map[string]struct {
		uuid        string
		permissions map[string]bool
	})

	for _, policy := range policies {
		policyName := StringFrom(policy, "name")
		policyUUID := StringFrom(policy, "uuid")

		// Get full policy details
		policyDetail, err := policyHandler.Get(ctx, policyUUID)
		if err != nil || policyDetail == nil {
			continue
		}

		statement := ""
		if s, ok := policyDetail["statementQuery"].(string); ok {
			statement = s
		}

		permissions := ParseStatementQuery(statement)
		permSet := make(map[string]bool)

		for _, perm := range permissions {
			permKey := fmt.Sprintf("%s:%s", perm.Effect, perm.Action)
			allPermissions[permKey] = true
			permSet[permKey] = true
		}

		policyPermissions[policyName] = struct {
			uuid        string
			permissions map[string]bool
		}{
			uuid:        policyUUID,
			permissions: permSet,
		}
	}

	// Build matrix
	var permissionList []string
	for perm := range allPermissions {
		permissionList = append(permissionList, perm)
	}

	var policyNames []string
	var matrix []map[string]any

	for policyName, data := range policyPermissions {
		policyNames = append(policyNames, policyName)
		row := map[string]any{
			"policy_name": policyName,
			"policy_uuid": data.uuid,
		}
		for _, perm := range permissionList {
			row[perm] = data.permissions[perm]
		}
		matrix = append(matrix, row)
	}

	return &MatrixResult{
		Permissions:     permissionList,
		Policies:        policyNames,
		Matrix:          matrix,
		PolicyCount:     len(policies),
		PermissionCount: len(permissionList),
	}, nil
}

// GenerateGroupMatrix generates a matrix of groups and their effective permissions.
func (pm *PermissionsMatrix) GenerateGroupMatrix(ctx context.Context) (*MatrixResult, error) {
	groupHandler := resources.NewGroupHandler(pm.client)
	bindingHandler := resources.NewBindingHandler(pm.client)
	policyHandler := resources.NewPolicyHandler(pm.client)

	groups, err := groupHandler.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	// Collect all unique permissions
	allPermissions := make(map[string]bool)
	groupPermissions := make(map[string]struct {
		uuid        string
		permissions map[string]bool
	})

	for _, group := range groups {
		groupName := StringFrom(group, "name")
		groupUUID := StringFrom(group, "uuid")

		bindings, err := bindingHandler.GetForGroup(ctx, groupUUID)
		if err != nil {
			continue
		}

		permSet := make(map[string]bool)

		for _, binding := range bindings {
			policyUUID := StringFrom(binding, "policyUuid")
			policy, err := policyHandler.Get(ctx, policyUUID)
			if err != nil || policy == nil {
				continue
			}

			statement := ""
			if s, ok := policy["statementQuery"].(string); ok {
				statement = s
			}

			permissions := ParseStatementQuery(statement)
			for _, perm := range permissions {
				permKey := fmt.Sprintf("%s:%s", perm.Effect, perm.Action)
				allPermissions[permKey] = true
				permSet[permKey] = true
			}
		}

		groupPermissions[groupName] = struct {
			uuid        string
			permissions map[string]bool
		}{
			uuid:        groupUUID,
			permissions: permSet,
		}
	}

	// Build matrix
	var permissionList []string
	for perm := range allPermissions {
		permissionList = append(permissionList, perm)
	}

	var groupNames []string
	var matrix []map[string]any

	for groupName, data := range groupPermissions {
		groupNames = append(groupNames, groupName)
		row := map[string]any{
			"group_name": groupName,
			"group_uuid": data.uuid,
		}
		for _, perm := range permissionList {
			row[perm] = data.permissions[perm]
		}
		matrix = append(matrix, row)
	}

	return &MatrixResult{
		Permissions:     permissionList,
		Groups:          groupNames,
		Matrix:          matrix,
		GroupCount:      len(groups),
		PermissionCount: len(permissionList),
	}, nil
}

// EffectivePermissionsAPI provides direct API access for effective permissions resolution.
type EffectivePermissionsAPI struct {
	client *client.Client
}

// NewEffectivePermissionsAPI creates a new effective permissions API client.
func NewEffectivePermissionsAPI(c *client.Client) *EffectivePermissionsAPI {
	return &EffectivePermissionsAPI{client: c}
}

// APIEffectivePermissions represents the result from the Dynatrace API.
type APIEffectivePermissions struct {
	EntityID             string           `json:"entityId" yaml:"entityId"`
	EntityType           string           `json:"entityType" yaml:"entityType"`
	LevelType            string           `json:"levelType" yaml:"levelType"`
	LevelID              string           `json:"levelId" yaml:"levelId"`
	EffectivePermissions []map[string]any `json:"effectivePermissions" yaml:"effectivePermissions"`
	Total                int              `json:"total" yaml:"total"`
	Error                string           `json:"error,omitempty" yaml:"error,omitempty"`
}

// GetEffectivePermissions gets effective permissions from the Dynatrace API.
func (api *EffectivePermissionsAPI) GetEffectivePermissions(ctx context.Context, entityID, entityType, levelType, levelID string, services []string) (*APIEffectivePermissions, error) {
	if levelID == "" {
		levelID = api.client.AccountUUID()
	}

	// Build the resolution API path
	path := fmt.Sprintf("%s/%s/%s/effectivepermissions", client.ResolutionBasePath, levelType, levelID)

	params := map[string]string{
		"entityId":   entityID,
		"entityType": entityType,
	}

	if len(services) > 0 {
		params["services"] = strings.Join(services, ",")
	}

	// Fetch all pages
	var allPermissions []map[string]any
	page := 1
	pageSize := 100

	for {
		params["page"] = fmt.Sprintf("%d", page)
		params["size"] = fmt.Sprintf("%d", pageSize)

		body, err := api.client.Get(ctx, path, params)
		if err != nil {
			return &APIEffectivePermissions{
				Error: err.Error(),
			}, nil
		}

		// Parse response
		var response struct {
			EffectivePermissions []map[string]any `json:"effectivePermissions"`
			Items                []map[string]any `json:"items"`
			Total                int              `json:"total"`
		}
		if err := client.ParseJSON(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		permissions := response.EffectivePermissions
		if permissions == nil {
			permissions = response.Items
		}

		allPermissions = append(allPermissions, permissions...)

		// Check if there are more pages
		if len(allPermissions) >= response.Total || len(permissions) == 0 {
			break
		}

		page++
	}

	return &APIEffectivePermissions{
		EntityID:             entityID,
		EntityType:           entityType,
		LevelType:            levelType,
		LevelID:              levelID,
		EffectivePermissions: allPermissions,
		Total:                len(allPermissions),
	}, nil
}

// GetUserEffectivePermissions gets effective permissions for a user via the API.
func (api *EffectivePermissionsAPI) GetUserEffectivePermissions(ctx context.Context, userID, levelType, levelID string, services []string) (*APIEffectivePermissions, error) {
	// Resolve user email to UID if needed
	if strings.Contains(userID, "@") {
		userHandler := resources.NewUserHandler(api.client)
		user, err := userHandler.GetByEmail(ctx, userID)
		if err != nil || user == nil {
			return &APIEffectivePermissions{
				Error: fmt.Sprintf("user not found: %s", userID),
			}, nil
		}
		userID = StringFrom(user, "uid")
	}

	return api.GetEffectivePermissions(ctx, userID, "user", levelType, levelID, services)
}

// GetGroupEffectivePermissions gets effective permissions for a group via the API.
func (api *EffectivePermissionsAPI) GetGroupEffectivePermissions(ctx context.Context, groupID, levelType, levelID string, services []string) (*APIEffectivePermissions, error) {
	// Resolve group name to UUID if needed
	groupHandler := resources.NewGroupHandler(api.client)
	group, err := groupHandler.Resolve(ctx, groupID)
	if err != nil {
		return &APIEffectivePermissions{
			Error: fmt.Sprintf("group not found: %s", groupID),
		}, nil
	}

	groupUUID := StringFrom(group, "uuid")
	return api.GetEffectivePermissions(ctx, groupUUID, "group", levelType, levelID, services)
}
