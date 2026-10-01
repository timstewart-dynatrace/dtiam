package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSubscriptionHandler_EnvironmentUsageRequiresWindow(t *testing.T) {
	// Both bounds are mandatory server-side; reject locally with a clear message.
	h := NewSubscriptionHandler(newTestClient(t, http.NewServeMux()))

	tests := []struct {
		name, sub, start, end string
	}{
		{name: "should reject a missing subscription UUID", sub: "", start: "s", end: "e"},
		{name: "should reject a missing start time", sub: "s1", start: "", end: "e"},
		{name: "should reject a missing end time", sub: "s1", start: "s", end: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := h.EnvironmentUsage(context.Background(), tt.sub, tt.start, tt.end, nil, nil); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestSubscriptionHandler_EnvironmentUsageSendsWindowAndFilters(t *testing.T) {
	var query url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions/s1/environments/usage", func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":             []any{map[string]any{"environmentId": "abc", "usage": 10}},
			"lastModifiedTime": "2026-10-01T00:00:00Z",
		})
	})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewSubscriptionHandler(c)
	h.Path = baseURL + "/subscriptions"

	result, err := h.EnvironmentUsage(context.Background(), "s1",
		"2026-09-01T00:00:00Z", "2026-10-01T00:00:00Z",
		[]string{"abc", "def"}, []string{"full_stack_monitoring"})
	if err != nil {
		t.Fatalf("EnvironmentUsage() error: %v", err)
	}
	if result["data"] == nil {
		t.Error("data missing from the result")
	}
	if query.Get("startTime") != "2026-09-01T00:00:00Z" {
		t.Errorf("startTime = %q", query.Get("startTime"))
	}
	if query.Get("environmentIds") != "abc,def" {
		t.Errorf("environmentIds = %q, want 'abc,def'", query.Get("environmentIds"))
	}
	if query.Get("capabilityKeys") != "full_stack_monitoring" {
		t.Errorf("capabilityKeys = %q", query.Get("capabilityKeys"))
	}
}

func TestSubscriptionHandler_EnvironmentUsageOmitsEmptyFilters(t *testing.T) {
	var query url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions/s1/environments/usage", func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewSubscriptionHandler(c)
	h.Path = baseURL + "/subscriptions"

	if _, err := h.EnvironmentUsage(context.Background(), "s1", "s", "e", nil, nil); err != nil {
		t.Fatalf("EnvironmentUsage() error: %v", err)
	}
	for _, k := range []string{"environmentIds", "capabilityKeys"} {
		if _, ok := query[k]; ok {
			t.Errorf("%s was sent despite being empty", k)
		}
	}
}

func TestSubscriptionHandler_EnvironmentCostUsesV3Path(t *testing.T) {
	// Cost moved to sub/v3 while listing, usage and forecast stayed on v2.
	// Building this from h.Path would silently 404 against the live API.
	gotPath := ""
	mux := http.NewServeMux()
	mux.HandleFunc("/sub/v3/accounts/test-uuid/subscriptions/s1/environments/cost",
		func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []any{map[string]any{"environmentId": "abc", "cost": 42}},
			})
		})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewSubscriptionHandler(c)
	h.V3BaseURL = baseURL + "/sub/v3/accounts/test-uuid"

	result, err := h.EnvironmentCost(context.Background(), "s1",
		"2026-09-01T00:00:00Z", "2026-10-01T00:00:00Z")
	if err != nil {
		t.Fatalf("EnvironmentCost() error: %v", err)
	}
	if gotPath == "" {
		t.Fatal("the request never reached the v3 path")
	}
	if strings.Contains(gotPath, "/sub/v2/") {
		t.Errorf("path = %q, want the v3 path", gotPath)
	}
	if result["data"] == nil {
		t.Error("data missing from the result")
	}
}

func TestSubscriptionHandler_EnvironmentCostSendsWindow(t *testing.T) {
	var query url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/sub/v3/accounts/test-uuid/subscriptions/s1/environments/cost",
		func(w http.ResponseWriter, r *http.Request) {
			query = r.URL.Query()
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewSubscriptionHandler(c)
	h.V3BaseURL = baseURL + "/sub/v3/accounts/test-uuid"

	if _, err := h.EnvironmentCost(context.Background(), "s1", "START", "END"); err != nil {
		t.Fatalf("EnvironmentCost() error: %v", err)
	}
	if query.Get("startTime") != "START" || query.Get("endTime") != "END" {
		t.Errorf("window not sent: %v", query)
	}
}

func TestSubscriptionHandler_EnvironmentCostRequiresWindow(t *testing.T) {
	h := NewSubscriptionHandler(newTestClient(t, http.NewServeMux()))
	for _, tt := range []struct{ name, sub, start, end string }{
		{name: "should reject a missing subscription UUID", sub: "", start: "s", end: "e"},
		{name: "should reject a missing start time", sub: "s1", start: "", end: "e"},
		{name: "should reject a missing end time", sub: "s1", start: "s", end: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := h.EnvironmentCost(context.Background(), tt.sub, tt.start, tt.end); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestNewSubscriptionHandler_SetsV3BaseURL(t *testing.T) {
	h := NewSubscriptionHandler(newTestClient(t, http.NewServeMux()))
	want := "https://api.dynatrace.com/sub/v3/accounts/test-uuid"
	if h.V3BaseURL != want {
		t.Errorf("V3BaseURL = %q, want %q", h.V3BaseURL, want)
	}
	if !strings.Contains(h.Path, "/sub/v2/") {
		t.Errorf("Path = %q, want it to stay on v2", h.Path)
	}
}
