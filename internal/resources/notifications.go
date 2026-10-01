package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtimothystewart/dtiam/internal/client"
)

// Notification types reported by the account notifications API.
const (
	NotificationForecast      = "FORECAST"
	NotificationBudget        = "BUDGET"
	NotificationCost          = "COST"
	NotificationBYOKRevoked   = "BYOK_REVOKED"
	NotificationBYOKActivated = "BYOK_ACTIVATED"
)

// Notification severities.
const (
	SeveritySevere = "SEVERE"
	SeverityWarn   = "WARN"
	SeverityInfo   = "INFO"
)

// ValidNotificationTypes lists the accepted notification type filters.
var ValidNotificationTypes = []string{
	NotificationForecast, NotificationBudget, NotificationCost,
	NotificationBYOKRevoked, NotificationBYOKActivated,
}

// ValidSeverities lists the accepted severity filters.
var ValidSeverities = []string{SeveritySevere, SeverityWarn, SeverityInfo}

// NotificationHandler handles account notifications: budget, cost, forecast and
// bring-your-own-key events.
//
// Requires the `account-uac-read` scope. Unusually for a read operation this is
// a POST, because the filter is sent as a request body rather than as query
// parameters.
type NotificationHandler struct {
	Client *client.Client

	// BaseURL is the notifications API root. Defaults to
	// client.NotificationsBaseURL; overridable for tests, since the unprefixed
	// /v1 path cannot be reached through the client's account-scoped base URL.
	BaseURL string
}

// NewNotificationHandler creates a new notification handler.
func NewNotificationHandler(c *client.Client) *NotificationHandler {
	return &NotificationHandler{Client: c, BaseURL: client.NotificationsBaseURL}
}

// ResourceName returns the resource name.
func (h *NotificationHandler) ResourceName() string { return "notification" }

// APIPath returns the notifications path for the configured account.
func (h *NotificationHandler) APIPath() string {
	base := h.BaseURL
	if base == "" {
		base = client.NotificationsBaseURL
	}
	return fmt.Sprintf("%s/%s/notifications", base, h.Client.AccountUUID())
}

// NotificationQuery filters the notification list. Zero values mean "no filter",
// which the API reads as "all".
type NotificationQuery struct {
	// StartDateTime and EndDateTime bound the window, in ISO-8601.
	StartDateTime string
	EndDateTime   string

	// Types restricts results to these notification types.
	Types []string

	// Severities restricts results to these severities.
	Severities []string
}

// Validate checks the filter values against the accepted enums, so a typo fails
// with a clear message instead of silently matching nothing server-side.
func (q NotificationQuery) Validate() error {
	for _, t := range q.Types {
		if !containsString(ValidNotificationTypes, t) {
			return fmt.Errorf("invalid notification type %q, want one of %s",
				t, strings.Join(ValidNotificationTypes, ", "))
		}
	}
	for _, s := range q.Severities {
		if !containsString(ValidSeverities, s) {
			return fmt.Errorf("invalid severity %q, want one of %s",
				s, strings.Join(ValidSeverities, ", "))
		}
	}
	return nil
}

// body builds the request payload, omitting empty filters.
func (q NotificationQuery) body() map[string]any {
	payload := map[string]any{}
	if q.StartDateTime != "" {
		payload["startDateTime"] = q.StartDateTime
	}
	if q.EndDateTime != "" {
		payload["endDateTime"] = q.EndDateTime
	}
	if len(q.Types) > 0 {
		payload["types"] = q.Types
	}
	if len(q.Severities) > 0 {
		payload["severities"] = q.Severities
	}
	return payload
}

// NotificationResult holds a page of notifications plus its paging metadata.
type NotificationResult struct {
	Records    []map[string]any `json:"records" yaml:"records"`
	TotalCount int              `json:"totalCount" yaml:"totalCount"`
	HasMore    bool             `json:"hasMore" yaml:"hasMore"`
}

// Query fetches notifications matching the filter.
func (h *NotificationHandler) Query(ctx context.Context, q NotificationQuery) (*NotificationResult, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	body, err := h.Client.Post(ctx, h.APIPath(), q.body())
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}

	var result NotificationResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse notifications response: %w", err)
	}
	return &result, nil
}

// containsString reports whether needle is in haystack.
func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
