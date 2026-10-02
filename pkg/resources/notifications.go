package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

// Notification types reported by the account notifications API.
const (
	NotificationForecast             = "FORECAST"
	NotificationBudget               = "BUDGET"
	NotificationCost                 = "COST"
	NotificationBYOKRevoked          = "BYOK_REVOKED"
	NotificationBYOKActivated        = "BYOK_ACTIVATED"
	NotificationEnvironmentUpgrade   = "ENVIRONMENT_UPGRADE"
	NotificationEnvironmentDowngrade = "ENVIRONMENT_DOWNGRADE"
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
	NotificationEnvironmentUpgrade, NotificationEnvironmentDowngrade,
}

// ValidSeverities lists the accepted severity filters.
var ValidSeverities = []string{SeveritySevere, SeverityWarn, SeverityInfo}

// notificationPageSize is the page size requested from the v2 API. Its default
// is 20; 500 returns a typical account's whole history in one request.
const notificationPageSize = 500

// NotificationHandler handles account notifications: budget, cost, forecast,
// bring-your-own-key and environment upgrade/downgrade events.
//
// Uses GET /v2/accounts/{uuid}/notifications, which replaced the v1 POST
// (deprecated 2026-06-15, removed 2027-01-11). Requires `account-uac-read`.
type NotificationHandler struct {
	Client *client.Client

	// BaseURL overrides client.NotificationsV2BaseURL. Tests point it at a mock
	// server, since the production URL is absolute.
	BaseURL string
}

// NewNotificationHandler creates a new notification handler.
func NewNotificationHandler(c *client.Client) *NotificationHandler {
	return &NotificationHandler{Client: c, BaseURL: client.NotificationsV2BaseURL}
}

// ResourceName returns the resource name.
func (h *NotificationHandler) ResourceName() string { return "notification" }

// APIPath returns the notifications path for the configured account.
func (h *NotificationHandler) APIPath() string {
	base := h.BaseURL
	if base == "" {
		base = client.NotificationsV2BaseURL
	}
	return fmt.Sprintf("%s/%s/notifications", base, h.Client.AccountUUID())
}

// NotificationQuery filters the notification list. Zero values mean "no filter",
// which the API reads as "all".
type NotificationQuery struct {
	// StartDateTime and EndDateTime bound the window, in ISO 8601.
	StartDateTime string
	EndDateTime   string

	// Types filters by notification type; see ValidNotificationTypes.
	Types []string

	// Severities filters by severity; see ValidSeverities.
	Severities []string

	// Environments filters by environment ID.
	Environments []string

	// Capabilities filters by subscription capability key.
	Capabilities []string
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

// query builds the request parameters. List filters are repeated parameters
// (types=A&types=B): the v2 API rejects a comma-separated value with HTTP 400.
func (q NotificationQuery) query() url.Values {
	v := url.Values{}
	if q.StartDateTime != "" {
		v.Set("start-time", q.StartDateTime)
	}
	if q.EndDateTime != "" {
		v.Set("end-time", q.EndDateTime)
	}
	for _, t := range q.Types {
		v.Add("types", t)
	}
	for _, s := range q.Severities {
		v.Add("severities", s)
	}
	for _, e := range q.Environments {
		v.Add("environments", e)
	}
	for _, c := range q.Capabilities {
		v.Add("capabilities", c)
	}
	return v
}

// NotificationResult holds the matching notifications.
type NotificationResult struct {
	Records []map[string]any `json:"records" yaml:"records"`

	// TotalCount is the number of records returned. The v2 API documents a
	// totalRecordCount field but does not send it, so this is counted locally.
	TotalCount int `json:"totalCount" yaml:"totalCount"`

	// HasMore reports that the page limit was reached before the API ran out
	// of results. Query follows every page, so this is normally false.
	HasMore bool `json:"hasMore" yaml:"hasMore"`
}

// notificationPage is one page of the v2 response.
type notificationPage struct {
	Records     []map[string]any `json:"records"`
	HasNextPage bool             `json:"hasNextPage"`
	NextPageKey string           `json:"nextPageKey"`
}

// Query fetches every notification matching the filter, following pages.
//
// On the v2 API the filters go only on the first request; later pages are
// requested with page-key alone, which carries the original query.
func (h *NotificationHandler) Query(ctx context.Context, q NotificationQuery) (*NotificationResult, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	result := &NotificationResult{Records: []map[string]any{}}
	params := q.query()
	params.Set("page-size", fmt.Sprint(notificationPageSize))

	for page := 0; page < client.MaxPageRequests; page++ {
		body, err := h.Client.GetWithQuery(ctx, h.APIPath(), params)
		if err != nil {
			return nil, fmt.Errorf("failed to query notifications: %w", err)
		}

		var p notificationPage
		if err := json.Unmarshal(body, &p); err != nil {
			return nil, fmt.Errorf("failed to parse notifications response: %w", err)
		}
		result.Records = append(result.Records, p.Records...)

		if !p.HasNextPage || p.NextPageKey == "" {
			result.TotalCount = len(result.Records)
			return result, nil
		}
		params = url.Values{"page-key": {p.NextPageKey}}
	}

	result.TotalCount = len(result.Records)
	result.HasMore = true
	return result, nil
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
