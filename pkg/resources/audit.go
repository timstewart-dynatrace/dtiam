package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

// AuditHandler handles account audit log resources.
//
// The audit API is the record of who changed what in the account: group
// membership edits, policy binding changes, user provisioning, token creation.
// Requires the `account-audit-logs-read` scope.
//
// It does not follow the usual Account Management conventions: the base path is
// /audit/v1 rather than /iam/v1, the response is keyed `audits` rather than
// `items`, and it reports partial-result `warnings` alongside the data.
type AuditHandler struct {
	BaseHandler
}

// NewAuditHandler creates a new audit log handler.
func NewAuditHandler(c *client.Client) *AuditHandler {
	path := fmt.Sprintf("%s/%s", client.AuditBaseURL, c.AccountUUID())
	return &AuditHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "audit-log",
			Path:      path,
			ListKey:   "audits",
			IDField:   "eventId",
			NameField: "eventType",
		},
	}
}

// AuditQuery describes a window and filter over the audit log.
type AuditQuery struct {
	// StartTime and EndTime accept ISO-8601, Unix epoch milliseconds, or
	// relative timestamps such as "now-24h". Empty means the API default window.
	StartTime string
	EndTime   string

	// Filter is an audit filter expression, e.g. `eventType = "CREATE"`.
	Filter string

	// AddFields names extra fields to include beyond the default projection.
	AddFields []string

	// Limit caps the number of returned entries. Zero means the API default.
	Limit int

	// ScanLimitGigabyte and ResultSizeLimitMegabyte bound the server-side scan.
	// Zero means the API default. These guard against expensive queries on
	// accounts with large audit histories.
	ScanLimitGigabyte       int
	ResultSizeLimitMegabyte int
}

// params converts the query into API query parameters, omitting unset fields so
// the server applies its own defaults.
func (q AuditQuery) params() map[string]string {
	params := map[string]string{}
	if q.StartTime != "" {
		params["startTime"] = q.StartTime
	}
	if q.EndTime != "" {
		params["endTime"] = q.EndTime
	}
	if q.Filter != "" {
		params["filter"] = q.Filter
	}
	if len(q.AddFields) > 0 {
		params["addFields"] = joinNonEmpty(q.AddFields, ",")
	}
	if q.Limit > 0 {
		params["limit"] = fmt.Sprintf("%d", q.Limit)
	}
	if q.ScanLimitGigabyte > 0 {
		params["scanLimitGigabyte"] = fmt.Sprintf("%d", q.ScanLimitGigabyte)
	}
	if q.ResultSizeLimitMegabyte > 0 {
		params["resultSizeLimitMegabyte"] = fmt.Sprintf("%d", q.ResultSizeLimitMegabyte)
	}
	return params
}

// AuditResult holds audit entries plus any warnings the API reported.
//
// Warnings matter here: the audit API returns HTTP 200 with a partial result
// when a scan or result-size limit is hit, so discarding warnings would present
// a truncated audit trail as complete.
type AuditResult struct {
	Audits   []map[string]any `json:"audits" yaml:"audits"`
	Warnings []string         `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// Query fetches audit log entries for the given window and filter.
func (h *AuditHandler) Query(ctx context.Context, q AuditQuery) (*AuditResult, error) {
	body, err := h.Client.Get(ctx, h.Path, q.params())
	if err != nil {
		return nil, h.handleError("query", err)
	}

	var response struct {
		Audits   []map[string]any `json:"audits"`
		Warnings []struct {
			Message string `json:"message"`
			Warning string `json:"warning"`
		} `json:"warnings"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse audit response: %w", err)
	}

	result := &AuditResult{Audits: response.Audits}
	for _, w := range response.Warnings {
		// The API uses `message` in most responses and `warning` in some; take
		// whichever is populated rather than dropping the warning entirely.
		if w.Message != "" {
			result.Warnings = append(result.Warnings, w.Message)
		} else if w.Warning != "" {
			result.Warnings = append(result.Warnings, w.Warning)
		}
	}

	return result, nil
}

// List returns audit entries using the API's default window.
func (h *AuditHandler) List(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	body, err := h.Client.Get(ctx, h.Path, params)
	if err != nil {
		return nil, h.handleError("list", err)
	}
	return h.extractList(body)
}

// joinNonEmpty joins the non-empty elements of parts with sep.
func joinNonEmpty(parts []string, sep string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += sep
		}
		out += p
	}
	return out
}
