package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func newTestSubscriptionHandler(t *testing.T, mux *http.ServeMux) *SubscriptionHandler {
	t.Helper()
	c := newTestClient(t, mux)
	return &SubscriptionHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "subscription",
			Path:      "/subscriptions",
			ListKey:   "items",
			IDField:   "uuid",
			NameField: "name",
		},
		baseURL: "",
	}
}

func subscriptionsResponse() map[string]any {
	return map[string]any{
		"items": []any{
			map[string]any{
				"uuid":   "sub1",
				"name":   "Enterprise",
				"status": "ACTIVE",
				"type":   "SaaS",
				"capabilities": []any{
					map[string]any{"key": "APM", "value": "1000"},
				},
			},
			map[string]any{
				"uuid":   "sub2",
				"name":   "Trial",
				"status": "EXPIRED",
				"type":   "SaaS",
			},
		},
	}
}

func TestSubscriptionHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(subscriptionsResponse())
	})

	h := newTestSubscriptionHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(items))
	}
}

func TestSubscriptionHandler_List_SubscriptionsKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"subscriptions": []any{
				map[string]any{"uuid": "sub1", "name": "Enterprise"},
			},
		})
	})

	h := newTestSubscriptionHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() returned %d items, want 1", len(items))
	}
}

func TestSubscriptionHandler_List_SingleItemResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid": "sub1",
			"name": "Enterprise",
		})
	})

	h := newTestSubscriptionHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() returned %d items, want 1", len(items))
	}
}

func TestSubscriptionHandler_List_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"message":"server error"}`))
	})

	h := newTestSubscriptionHandler(t, mux)
	_, err := h.List(context.Background(), nil)
	if err == nil {
		t.Fatal("List() expected error for 500, got nil")
	}
}

func TestSubscriptionHandler_Get_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions/sub1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid":   "sub1",
			"name":   "Enterprise",
			"status": "ACTIVE",
		})
	})

	h := newTestSubscriptionHandler(t, mux)
	item, err := h.Get(context.Background(), "sub1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["name"] != "Enterprise" {
		t.Errorf("Get() name = %v, want Enterprise", item["name"])
	}
}

func TestSubscriptionHandler_Get_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestSubscriptionHandler(t, mux)
	_, err := h.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("Get() expected error for 404, got nil")
	}
}

func TestSubscriptionHandler_GetByName_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(subscriptionsResponse())
	})

	h := newTestSubscriptionHandler(t, mux)
	item, err := h.GetByName(context.Background(), "enterprise")
	if err != nil {
		t.Fatalf("GetByName() error: %v", err)
	}
	if item == nil {
		t.Fatal("GetByName() returned nil")
	}
	if item["uuid"] != "sub1" {
		t.Errorf("GetByName() uuid = %v, want sub1", item["uuid"])
	}
}

func TestSubscriptionHandler_GetByName_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(subscriptionsResponse())
	})

	h := newTestSubscriptionHandler(t, mux)
	item, err := h.GetByName(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByName() error: %v", err)
	}
	if item != nil {
		t.Errorf("GetByName() should return nil for missing name, got %v", item)
	}
}

func TestSubscriptionHandler_GetSummary_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(subscriptionsResponse())
	})

	h := newTestSubscriptionHandler(t, mux)
	summary, err := h.GetSummary(context.Background())
	if err != nil {
		t.Fatalf("GetSummary() error: %v", err)
	}
	if summary["total_subscriptions"] != 2 {
		t.Errorf("GetSummary() total = %v, want 2", summary["total_subscriptions"])
	}
	if summary["active_subscriptions"] != 1 {
		t.Errorf("GetSummary() active = %v, want 1", summary["active_subscriptions"])
	}
}

func TestSubscriptionHandler_GetUsage_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions/sub1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid":   "sub1",
			"name":   "Enterprise",
			"status": "ACTIVE",
			"type":   "SaaS",
			"currentUsage": map[string]any{
				"hostUnits": float64(42),
			},
		})
	})

	h := newTestSubscriptionHandler(t, mux)
	result, err := h.GetUsage(context.Background(), "sub1")
	if err != nil {
		t.Fatalf("GetUsage() error: %v", err)
	}
	if result["subscription_uuid"] != "sub1" {
		t.Errorf("GetUsage() subscription_uuid = %v, want sub1", result["subscription_uuid"])
	}
	usage, ok := result["usage"].(map[string]any)
	if !ok {
		t.Fatal("GetUsage() usage field missing or wrong type")
	}
	if usage["hostUnits"] != float64(42) {
		t.Errorf("GetUsage() hostUnits = %v, want 42", usage["hostUnits"])
	}
}

func TestSubscriptionHandler_GetForecast_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions/forecast", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"forecast": "data",
		})
	})

	h := newTestSubscriptionHandler(t, mux)
	result, err := h.GetForecast(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetForecast() error: %v", err)
	}
	if result["forecast"] != "data" {
		t.Errorf("GetForecast() forecast = %v, want data", result["forecast"])
	}
}

func TestSubscriptionHandler_GetCapabilities_All(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{
					"uuid": "sub1",
					"name": "Enterprise",
					"capabilities": []any{
						map[string]any{"key": "APM", "enabled": true},
						map[string]any{"key": "RUM", "enabled": false},
					},
				},
				map[string]any{
					"uuid": "sub2",
					"name": "Trial",
					"capabilities": []any{
						map[string]any{"key": "LogAnalytics", "enabled": true},
					},
				},
			},
		})
	})

	h := newTestSubscriptionHandler(t, mux)
	caps, err := h.GetCapabilities(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetCapabilities() error: %v", err)
	}
	if len(caps) != 3 {
		t.Fatalf("GetCapabilities() returned %d capabilities, want 3", len(caps))
	}
	// Verify subscription name is attached
	if caps[0]["subscription"] != "Enterprise" {
		t.Errorf("GetCapabilities()[0] subscription = %v, want Enterprise", caps[0]["subscription"])
	}
	if caps[2]["subscription"] != "Trial" {
		t.Errorf("GetCapabilities()[2] subscription = %v, want Trial", caps[2]["subscription"])
	}
}

func TestSubscriptionHandler_GetCapabilities_SingleSubscription(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions/sub1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid": "sub1",
			"name": "Enterprise",
			"capabilities": []any{
				map[string]any{"key": "APM", "enabled": true},
			},
		})
	})

	h := newTestSubscriptionHandler(t, mux)
	uuid := "sub1"
	caps, err := h.GetCapabilities(context.Background(), &uuid)
	if err != nil {
		t.Fatalf("GetCapabilities() error: %v", err)
	}
	if len(caps) != 1 {
		t.Fatalf("GetCapabilities() returned %d capabilities, want 1", len(caps))
	}
	if caps[0]["key"] != "APM" {
		t.Errorf("GetCapabilities()[0] key = %v, want APM", caps[0]["key"])
	}
}

func TestSubscriptionHandler_GetCapabilities_NoCapabilities(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{
					"uuid": "sub1",
					"name": "Basic",
				},
			},
		})
	})

	h := newTestSubscriptionHandler(t, mux)
	caps, err := h.GetCapabilities(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetCapabilities() error: %v", err)
	}
	if len(caps) != 0 {
		t.Errorf("GetCapabilities() returned %d capabilities, want 0", len(caps))
	}
}

func TestSubscriptionHandler_GetForecast_WithUUID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions/sub1/forecast", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"forecast": "specific",
		})
	})

	h := newTestSubscriptionHandler(t, mux)
	uuid := "sub1"
	result, err := h.GetForecast(context.Background(), &uuid)
	if err != nil {
		t.Fatalf("GetForecast() error: %v", err)
	}
	if result["forecast"] != "specific" {
		t.Errorf("GetForecast() forecast = %v, want specific", result["forecast"])
	}
}
