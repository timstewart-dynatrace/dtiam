package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtimothystewart/dtiam/pkg/client"
)

// Organizational level types for the environment-level Platform IAM API.
const (
	// LevelAccount scopes a lookup to an account; the level ID is the account UUID.
	LevelAccount = "account"

	// LevelEnvironment scopes a lookup to one environment; the level ID is the
	// environment ID.
	LevelEnvironment = "environment"
)

// OrgLevelHandler handles the environment-level Platform IAM API.
//
// This is a different API from the account-level Account Management API the rest
// of dtiam uses. It is served from the environment
// (https://{env}.apps.dynatrace.com/platform/iam/v1) rather than from
// api.dynatrace.com, and answers "who can actually see this environment"
// instead of "who is in this account". Requires the `iam:users:read` scope,
// which is granted on the environment, not the account.
//
// Because it is environment-served, it needs an environment URL and will return
// a clear error when one is not configured rather than failing at request time.
type OrgLevelHandler struct {
	Client *client.Client

	// EnvironmentURL is the environment base URL, e.g.
	// https://abc12345.apps.dynatrace.com.
	EnvironmentURL string
}

// NewOrgLevelHandler creates a handler for the environment-level IAM API.
//
// envURL accepts either a full URL or a bare environment ID, which is expanded
// to https://{id}.apps.dynatrace.com. The .apps host is required here: this API
// is not served from .live.
func NewOrgLevelHandler(c *client.Client, envURL string) *OrgLevelHandler {
	if envURL != "" && !strings.HasPrefix(envURL, "http://") && !strings.HasPrefix(envURL, "https://") {
		envURL = fmt.Sprintf("https://%s.apps.dynatrace.com", envURL)
	}
	return &OrgLevelHandler{
		Client:         c,
		EnvironmentURL: strings.TrimSuffix(envURL, "/"),
	}
}

// ResourceName returns the resource name.
func (h *OrgLevelHandler) ResourceName() string { return "organizational-level" }

// basePath returns the organizational-levels path for a level, or an error when
// no environment URL is configured.
func (h *OrgLevelHandler) basePath(levelType, levelID string) (string, error) {
	if h.EnvironmentURL == "" {
		return "", fmt.Errorf(
			"environment URL is required for the environment-level IAM API; " +
				"set DTIAM_ENVIRONMENT_URL or configure environment-url on the credential")
	}
	if err := ValidateLevelType(levelType); err != nil {
		return "", err
	}
	if levelID == "" {
		return "", fmt.Errorf("level ID is required")
	}

	return fmt.Sprintf("%s/platform/iam/v1/organizational-levels/%s/%s",
		h.EnvironmentURL, levelType, levelID), nil
}

// ValidateLevelType checks levelType against the accepted values.
func ValidateLevelType(levelType string) error {
	switch levelType {
	case LevelAccount, LevelEnvironment:
		return nil
	default:
		return fmt.Errorf("invalid level type %q, want %q or %q", levelType, LevelAccount, LevelEnvironment)
	}
}

// ListUsers returns the active users visible at an organizational level.
//
// The API requires at least one of partialString or uuid; passing neither is
// rejected server-side, so this rejects it up front with a clearer message.
// When both are given, results match either criterion.
func (h *OrgLevelHandler) ListUsers(
	ctx context.Context, levelType, levelID, partialString, uuid string,
) ([]map[string]any, error) {
	base, err := h.basePath(levelType, levelID)
	if err != nil {
		return nil, err
	}
	if partialString == "" && uuid == "" {
		return nil, fmt.Errorf("a search term or user UUID is required to list users at an organizational level")
	}

	params := map[string]string{}
	if partialString != "" {
		params["partialString"] = partialString
	}
	if uuid != "" {
		params["uuid"] = uuid
	}

	handler := &BaseHandler{
		Client:     h.Client,
		Name:       "user",
		Path:       base + "/users",
		ListKey:    "results",
		IDField:    "uuid",
		NameField:  "email",
		Pagination: client.OrganizationalLevelPagination(),
	}
	return handler.List(ctx, params)
}

// GetUser returns one active user at an organizational level.
func (h *OrgLevelHandler) GetUser(ctx context.Context, levelType, levelID, userUUID string) (map[string]any, error) {
	base, err := h.basePath(levelType, levelID)
	if err != nil {
		return nil, err
	}
	if userUUID == "" {
		return nil, fmt.Errorf("user UUID is required")
	}

	body, err := h.Client.Get(ctx, fmt.Sprintf("%s/users/%s", base, userUUID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s at %s level: %w", userUUID, levelType, err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}
	return result, nil
}

// ListGroups returns the groups visible at an organizational level.
func (h *OrgLevelHandler) ListGroups(
	ctx context.Context, levelType, levelID, partialString string,
) ([]map[string]any, error) {
	base, err := h.basePath(levelType, levelID)
	if err != nil {
		return nil, err
	}

	// Groups filter on partialGroupName, not the partialString users take, and
	// the API rejects a request without it (or a uuid): "Mandatory query param
	// partialGroupName or uuid was not provided". Minimum length is 3.
	if len(strings.TrimSpace(partialString)) < 3 {
		return nil, fmt.Errorf("a search term of at least 3 characters is required to list groups at an organizational level")
	}
	params := map[string]string{"partialGroupName": partialString}

	handler := &BaseHandler{
		Client:     h.Client,
		Name:       "group",
		Path:       base + "/groups",
		ListKey:    "results",
		IDField:    "uuid",
		NameField:  "name",
		Pagination: client.OrganizationalLevelPagination(),
	}
	return handler.List(ctx, params)
}

// ListServiceUsers returns the service users visible at an organizational level.
func (h *OrgLevelHandler) ListServiceUsers(
	ctx context.Context, levelType, levelID, partialString string,
) ([]map[string]any, error) {
	base, err := h.basePath(levelType, levelID)
	if err != nil {
		return nil, err
	}

	params := map[string]string{}
	if partialString != "" {
		params["partialString"] = partialString
	}

	handler := &BaseHandler{
		Client:     h.Client,
		Name:       "service-user",
		Path:       base + "/service-users",
		ListKey:    "results",
		IDField:    "uid",
		NameField:  "name",
		Pagination: client.OrganizationalLevelPagination(),
	}
	return handler.List(ctx, params)
}
