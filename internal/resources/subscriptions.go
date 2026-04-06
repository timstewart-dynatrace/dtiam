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
}

// NewSubscriptionHandler creates a new subscription handler.
func NewSubscriptionHandler(c *client.Client) *SubscriptionHandler {
	// Subscription API uses a different base URL
	baseURL := fmt.Sprintf("%s/%s", client.SubBaseURL, c.AccountUUID())
	return &SubscriptionHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "subscription",
			Path:      baseURL + "/subscriptions",
			ListKey:   "items",
			IDField:   "uuid",
			NameField: "name",
		},
		baseURL: baseURL,
	}
}

// List lists subscriptions.
func (h *SubscriptionHandler) List(ctx context.Context, params map[string]string) ([]map[string]any, error) {
	body, err := h.Client.Get(ctx, h.Path, params)
	if err != nil {
		return nil, h.handleError("list", err)
	}

	return h.extractList(body)
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
func (h *SubscriptionHandler) extractList(body []byte) ([]map[string]any, error) {
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		var items []map[string]any
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return items, nil
	}

	// Try common list keys
	for _, key := range []string{"items", "subscriptions"} {
		if items, ok := response[key]; ok {
			return toMapSlice(items)
		}
	}

	// Single item response - wrap in array
	if _, ok := response["uuid"]; ok {
		return []map[string]any{response}, nil
	}

	return []map[string]any{}, nil
}
