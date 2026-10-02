package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

// SubscriptionHandler handles subscription resources.
type SubscriptionHandler struct {
	BaseHandler
	baseURL string

	// V3BaseURL is the account-scoped root for Subscription API endpoints that
	// live on v3. Overridable so tests can target a local server; the cost
	// endpoint cannot inherit h.Path, which is pinned to v2.
	V3BaseURL string
}

// NewSubscriptionHandler creates a new subscription handler.
func NewSubscriptionHandler(c *client.Client) *SubscriptionHandler {
	// Subscription API uses a different base URL
	baseURL := fmt.Sprintf("%s/%s", client.SubBaseURL, c.AccountUUID())
	return &SubscriptionHandler{
		BaseHandler: BaseHandler{
			Client: c,
			Name:   "subscription",
			Path:   baseURL + "/subscriptions",
			// The API responds with {data: [...]}, not {items: [...]}.
			// Verified against a live account.
			ListKey:   "data",
			IDField:   "uuid",
			NameField: "name",
		},
		baseURL:   baseURL,
		V3BaseURL: fmt.Sprintf("%s/%s", client.SubV3BaseURL, c.AccountUUID()),
	}
}

// Get gets a subscription by UUID.
func (h *SubscriptionHandler) Get(ctx context.Context, subscriptionUUID string) (map[string]any, error) {
	path := fmt.Sprintf("%s/%s", h.Path, subscriptionUUID)
	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		return nil, h.handleError("get", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetByName gets a subscription by name.
func (h *SubscriptionHandler) GetByName(ctx context.Context, name string) (map[string]any, error) {
	items, err := h.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if itemName, ok := item["name"].(string); ok {
			if strings.EqualFold(itemName, name) {
				return item, nil
			}
		}
	}

	return nil, nil
}

// GetForecast gets the forecast for subscriptions.
func (h *SubscriptionHandler) GetForecast(ctx context.Context, subscriptionUUID *string) (map[string]any, error) {
	var path string
	if subscriptionUUID != nil && *subscriptionUUID != "" {
		path = fmt.Sprintf("%s/%s/forecast", h.Path, *subscriptionUUID)
	} else {
		path = fmt.Sprintf("%s/forecast", h.Path)
	}

	body, err := h.Client.Get(ctx, path, nil)
	if err != nil {
		return nil, h.handleError("get forecast", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetUsage gets usage information for a subscription.
func (h *SubscriptionHandler) GetUsage(ctx context.Context, subscriptionUUID string) (map[string]any, error) {
	sub, err := h.Get(ctx, subscriptionUUID)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"subscription_uuid": subscriptionUUID,
		"name":              sub["name"],
		"type":              sub["type"],
		"status":            sub["status"],
		"startTime":         sub["startTime"],
		"endTime":           sub["endTime"],
		"capabilities":      sub["capabilities"],
	}

	// Extract usage from currentUsage or usage field
	if usage, ok := sub["currentUsage"].(map[string]any); ok {
		result["usage"] = usage
	} else if usage, ok := sub["usage"].(map[string]any); ok {
		result["usage"] = usage
	}

	return result, nil
}

// GetSummary returns a summary of all subscriptions.
func (h *SubscriptionHandler) GetSummary(ctx context.Context) (map[string]any, error) {
	items, err := h.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	var active int
	for _, item := range items {
		if status, ok := item["status"].(string); ok {
			if strings.EqualFold(status, "active") {
				active++
			}
		}
	}

	return map[string]any{
		"total_subscriptions":  len(items),
		"active_subscriptions": active,
		"subscriptions":        items,
	}, nil
}

// GetCapabilities gets capabilities from subscriptions.
func (h *SubscriptionHandler) GetCapabilities(ctx context.Context, subscriptionUUID *string) ([]map[string]any, error) {
	var subscriptions []map[string]any

	if subscriptionUUID != nil && *subscriptionUUID != "" {
		sub, err := h.Get(ctx, *subscriptionUUID)
		if err != nil {
			return nil, err
		}
		subscriptions = []map[string]any{sub}
	} else {
		var err error
		subscriptions, err = h.List(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	var capabilities []map[string]any
	for _, sub := range subscriptions {
		subName, _ := sub["name"].(string)
		if caps, ok := sub["capabilities"].([]any); ok {
			for _, cap := range caps {
				if capMap, ok := cap.(map[string]any); ok {
					capMap["subscription"] = subName
					capabilities = append(capabilities, capMap)
				}
			}
		}
	}

	return capabilities, nil
}

// extractList handles subscription-specific response formats.

// EnvironmentUsage returns per-environment usage for a subscription over a
// window. Both bounds are required by the API, in "2021-05-01T15:11:00Z" form.
//
// Uses /sub/v3/.../environments/usage; the v2 endpoint was deprecated on
// 2026-06-15 and is removed on 2027-01-11. v3 pages its results, so every page
// is fetched and the result has the v2 shape: {"data": [...],
// "lastModifiedTime": ...}, each data entry being {environmentId, clusterId,
// usage: [...]}. Use FlattenEnvironmentData for one row per record.
//
// This is distinct from GetUsage, which only reports the usage totals embedded
// in the subscription object itself.
func (h *SubscriptionHandler) EnvironmentUsage(
	ctx context.Context, subscriptionUUID, startTime, endTime string,
	environmentIDs, capabilityKeys []string,
) (map[string]any, error) {
	return h.environmentData(ctx, "usage", subscriptionUUID, startTime, endTime, environmentIDs, capabilityKeys)
}

// EnvironmentCost returns per-environment cost for a subscription over a window,
// from /sub/v3/.../environments/cost, following every page. The result has the
// same shape as EnvironmentUsage with "cost" in place of "usage".
func (h *SubscriptionHandler) EnvironmentCost(
	ctx context.Context, subscriptionUUID, startTime, endTime string,
) (map[string]any, error) {
	return h.environmentData(ctx, "cost", subscriptionUUID, startTime, endTime, nil, nil)
}

// subscriptionV3PageSize is the largest page the v3 environment endpoints
// accept; anything above 50 is rejected with HTTP 400.
const subscriptionV3PageSize = 50

// environmentData fetches every page of /sub/v3/.../environments/{kind}.
//
// Unlike the v2 notifications API, these endpoints require the full query --
// startTime, endTime and filters -- on every page alongside page-key; a request
// carrying page-key alone is rejected ("'endTime' must be provided"). Verified
// against a live account.
func (h *SubscriptionHandler) environmentData(
	ctx context.Context, kind, subscriptionUUID, startTime, endTime string,
	environmentIDs, capabilityKeys []string,
) (map[string]any, error) {
	if subscriptionUUID == "" {
		return nil, fmt.Errorf("subscription UUID is required")
	}
	if startTime == "" || endTime == "" {
		return nil, fmt.Errorf("startTime and endTime are required for environment %s", kind)
	}

	v3Base := h.V3BaseURL
	if v3Base == "" {
		v3Base = fmt.Sprintf("%s/%s", client.SubV3BaseURL, h.Client.AccountUUID())
	}
	path := fmt.Sprintf("%s/subscriptions/%s/environments/%s", v3Base, subscriptionUUID, kind)

	params := map[string]string{
		"startTime": startTime,
		"endTime":   endTime,
		"page-size": fmt.Sprint(subscriptionV3PageSize),
	}
	if len(environmentIDs) > 0 {
		params["environmentIds"] = joinNonEmpty(environmentIDs, ",")
	}
	if len(capabilityKeys) > 0 {
		params["capabilityKeys"] = joinNonEmpty(capabilityKeys, ",")
	}

	data := []any{}
	var lastModified any
	for page := 0; page < client.MaxPageRequests; page++ {
		body, err := h.Client.Get(ctx, path, params)
		if err != nil {
			return nil, h.handleError("get environment "+kind, err)
		}

		var resp struct {
			Data             []any  `json:"data"`
			LastModifiedTime any    `json:"lastModifiedTime"`
			NextPageKey      string `json:"nextPageKey"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse environment %s response: %w", kind, err)
		}
		data = append(data, resp.Data...)
		if resp.LastModifiedTime != nil {
			lastModified = resp.LastModifiedTime
		}
		if resp.NextPageKey == "" || resp.NextPageKey == params["page-key"] {
			break
		}
		params["page-key"] = resp.NextPageKey
	}

	return map[string]any{"data": data, "lastModifiedTime": lastModified}, nil
}

// FlattenEnvironmentData turns an EnvironmentUsage or EnvironmentCost result
// into one row per record, each carrying its environmentId and clusterId.
// itemsKey is "usage" or "cost".
//
// The API nests records under each environment, and v3 also splits one
// environment's records across pages, so the same environmentId can appear in
// several entries. Rows are returned in API order.
func FlattenEnvironmentData(result map[string]any, itemsKey string) []map[string]any {
	rows := []map[string]any{}
	entries, _ := result["data"].([]any)
	for _, e := range entries {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		items, _ := entry[itemsKey].([]any)
		for _, it := range items {
			item, ok := it.(map[string]any)
			if !ok {
				continue
			}
			row := make(map[string]any, len(item)+2)
			for k, v := range item {
				row[k] = v
			}
			row["environmentId"] = entry["environmentId"]
			if cluster, ok := entry["clusterId"]; ok && cluster != nil {
				row["clusterId"] = cluster
			}
			rows = append(rows, row)
		}
	}
	return rows
}
