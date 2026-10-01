package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtimothystewart/dtiam/internal/client"
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
// This is distinct from GetUsage, which only reports the usage totals embedded
// in the subscription object itself.
func (h *SubscriptionHandler) EnvironmentUsage(
	ctx context.Context, subscriptionUUID, startTime, endTime string,
	environmentIDs, capabilityKeys []string,
) (map[string]any, error) {
	if subscriptionUUID == "" {
		return nil, fmt.Errorf("subscription UUID is required")
	}
	if startTime == "" || endTime == "" {
		return nil, fmt.Errorf("startTime and endTime are required for environment usage")
	}

	params := map[string]string{"startTime": startTime, "endTime": endTime}
	if len(environmentIDs) > 0 {
		params["environmentIds"] = joinNonEmpty(environmentIDs, ",")
	}
	if len(capabilityKeys) > 0 {
		params["capabilityKeys"] = joinNonEmpty(capabilityKeys, ",")
	}

	path := fmt.Sprintf("%s/%s/environments/usage", h.Path, subscriptionUUID)
	body, err := h.Client.Get(ctx, path, params)
	if err != nil {
		return nil, h.handleError("get environment usage", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse environment usage response: %w", err)
	}
	return result, nil
}

// EnvironmentCost returns per-environment cost for a subscription over a window.
//
// Note the version skew: this endpoint lives on sub/v3 while subscription
// listing, usage and forecast remain on sub/v2, so the path is built from
// SubV3BaseURL rather than from h.Path.
func (h *SubscriptionHandler) EnvironmentCost(
	ctx context.Context, subscriptionUUID, startTime, endTime string,
) (map[string]any, error) {
	if subscriptionUUID == "" {
		return nil, fmt.Errorf("subscription UUID is required")
	}
	if startTime == "" || endTime == "" {
		return nil, fmt.Errorf("startTime and endTime are required for environment cost")
	}

	v3Base := h.V3BaseURL
	if v3Base == "" {
		v3Base = fmt.Sprintf("%s/%s", client.SubV3BaseURL, h.Client.AccountUUID())
	}
	path := fmt.Sprintf("%s/subscriptions/%s/environments/cost", v3Base, subscriptionUUID)

	body, err := h.Client.Get(ctx, path, map[string]string{"startTime": startTime, "endTime": endTime})
	if err != nil {
		return nil, h.handleError("get environment cost", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse environment cost response: %w", err)
	}
	return result, nil
}
