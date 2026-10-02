package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

// Scope types accepted by the permission management API.
const (
	// ScopeTypeAccount scopes a permission to the whole account; scope is the
	// account UUID.
	ScopeTypeAccount = "account"

	// ScopeTypeTenant scopes a permission to one environment; scope is the
	// environment ID.
	ScopeTypeTenant = "tenant"

	// ScopeTypeManagementZone scopes a permission to a management zone; scope is
	// "{environment-id}:{management-zone-id}".
	ScopeTypeManagementZone = "management-zone"
)

// ValidScopeTypes lists the accepted scopeType values.
var ValidScopeTypes = []string{ScopeTypeAccount, ScopeTypeTenant, ScopeTypeManagementZone}

// GroupPermission is a single permission grant on a group.
//
// This is the role-style permission model that predates IAM policies and still
// coexists with them: a group's effective access is the union of its
// policy bindings and these direct permission grants. Auditing only policies
// therefore understates what a group can do.
type GroupPermission struct {
	PermissionName string `json:"permissionName" yaml:"permissionName" table:"PERMISSION"`
	Scope          string `json:"scope" yaml:"scope" table:"SCOPE"`
	ScopeType      string `json:"scopeType" yaml:"scopeType" table:"SCOPE_TYPE"`
	CreatedAt      string `json:"createdAt,omitempty" yaml:"createdAt,omitempty" table:"CREATED,wide"`
	UpdatedAt      string `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty" table:"UPDATED,wide"`
}

// GroupPermissionHandler handles direct permission grants on groups.
//
// Reads require `account-idm-read`; writes require `account-idm-write`.
type GroupPermissionHandler struct {
	Client *client.Client
}

// NewGroupPermissionHandler creates a new group permission handler.
func NewGroupPermissionHandler(c *client.Client) *GroupPermissionHandler {
	return &GroupPermissionHandler{Client: c}
}

// ResourceName returns the resource name.
func (h *GroupPermissionHandler) ResourceName() string { return "group-permission" }

// path returns the permissions path for a group. It is relative to the client's
// account-scoped base URL.
func (h *GroupPermissionHandler) path(groupUUID string) string {
	return fmt.Sprintf("/groups/%s/permissions", groupUUID)
}

// List returns the permissions currently granted to a group.
func (h *GroupPermissionHandler) List(ctx context.Context, groupUUID string) ([]GroupPermission, error) {
	body, err := h.Client.Get(ctx, h.path(groupUUID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions for group %s: %w", groupUUID, err)
	}

	// The API returns {permissions: [...]}; tolerate a bare array too.
	var wrapped struct {
		Permissions []GroupPermission `json:"permissions"`
		Items       []GroupPermission `json:"items"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		if wrapped.Permissions != nil {
			return wrapped.Permissions, nil
		}
		if wrapped.Items != nil {
			return wrapped.Items, nil
		}
	}

	var direct []GroupPermission
	if err := json.Unmarshal(body, &direct); err != nil {
		return nil, fmt.Errorf("failed to parse permissions response: %w", err)
	}
	return direct, nil
}

// Grant adds permissions to a group, leaving existing grants in place.
func (h *GroupPermissionHandler) Grant(ctx context.Context, groupUUID string, perms []GroupPermission) error {
	if len(perms) == 0 {
		return fmt.Errorf("no permissions to grant")
	}
	if err := ValidatePermissions(perms); err != nil {
		return err
	}

	if _, err := h.Client.Post(ctx, h.path(groupUUID), perms); err != nil {
		return fmt.Errorf("failed to grant permissions to group %s: %w", groupUUID, err)
	}
	return nil
}

// Replace overwrites a group's permissions with exactly the given set.
//
// An empty slice is rejected rather than treated as "remove everything": PUT
// with an empty body would silently strip every grant, which is too destructive
// to infer from an omitted argument. Use Revoke to remove grants.
func (h *GroupPermissionHandler) Replace(ctx context.Context, groupUUID string, perms []GroupPermission) error {
	if len(perms) == 0 {
		return fmt.Errorf("refusing to replace permissions with an empty set; use revoke to remove grants")
	}
	if err := ValidatePermissions(perms); err != nil {
		return err
	}

	if _, err := h.Client.Put(ctx, h.path(groupUUID), perms); err != nil {
		return fmt.Errorf("failed to replace permissions for group %s: %w", groupUUID, err)
	}
	return nil
}

// Revoke removes a single permission grant from a group.
//
// The API identifies the grant to delete by query parameter rather than by a
// path segment or request body, since a grant has no identifier of its own.
func (h *GroupPermissionHandler) Revoke(ctx context.Context, groupUUID, permissionName, scope, scopeType string) error {
	if permissionName == "" {
		return fmt.Errorf("permission name is required")
	}
	if err := validateScopeType(scopeType); err != nil {
		return err
	}

	params := map[string]string{
		"permission-name": permissionName,
		"scope":           scope,
		"scope-type":      scopeType,
	}
	path := h.path(groupUUID) + "?" + encodeParams(params)

	if _, err := h.Client.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to revoke %s from group %s: %w", permissionName, groupUUID, err)
	}
	return nil
}

// encodeParams builds a URL-encoded query string with stable ordering.
func encodeParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values.Encode()
}

// ValidatePermissions checks every grant has the fields the API requires.
func ValidatePermissions(perms []GroupPermission) error {
	for i, p := range perms {
		if p.PermissionName == "" {
			return fmt.Errorf("permission %d: permissionName is required", i)
		}
		if err := validateScopeType(p.ScopeType); err != nil {
			return fmt.Errorf("permission %q: %w", p.PermissionName, err)
		}
		if p.Scope == "" {
			return fmt.Errorf("permission %q: scope is required for scopeType %q", p.PermissionName, p.ScopeType)
		}
		if p.ScopeType == ScopeTypeManagementZone && !strings.Contains(p.Scope, ":") {
			return fmt.Errorf(
				"permission %q: management-zone scope must be \"{environment-id}:{management-zone-id}\", got %q",
				p.PermissionName, p.Scope)
		}
	}
	return nil
}

// validateScopeType checks scopeType against the accepted values.
func validateScopeType(scopeType string) error {
	for _, valid := range ValidScopeTypes {
		if scopeType == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid scopeType %q, want one of %s", scopeType, strings.Join(ValidScopeTypes, ", "))
}
