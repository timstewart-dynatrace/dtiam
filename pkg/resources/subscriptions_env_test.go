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
	mux.HandleFunc("/sub/v3/accounts/test-uuid/subscriptions/s1/environments/usage", func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":             []any{map[string]any{"environmentId": "abc", "usage": []any{}}},
			"lastModifiedTime": "2026-10-01T00:00:00Z",
		})
	})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewSubscriptionHandler(c)
	h.V3BaseURL = baseURL + "/sub/v3/accounts/test-uuid"

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
	mux.HandleFunc("/sub/v3/accounts/test-uuid/subscriptions/s1/environments/usage", func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewSubscriptionHandler(c)
	h.V3BaseURL = baseURL + "/sub/v3/accounts/test-uuid"

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

func TestSubscriptionHandler_EnvironmentUsageFollowsV3Pages(t *testing.T) {
	var requests []url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/sub/v3/accounts/test-uuid/subscriptions/s1/environments/usage", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		requests = append(requests, q)
		// Live behavior: one environment's records are split across pages.
		if q.Get("page-key") == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []any{map[string]any{"environmentId": "abc", "usage": []any{
					map[string]any{"capabilityKey": "A", "value": 1.0},
				}}},
				"lastModifiedTime": "2026-10-01T00:00:00Z",
				"nextPageKey":      "k2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{"environmentId": "abc", "clusterId": "c1", "usage": []any{
				map[string]any{"capabilityKey": "B", "value": 2.0},
			}}},
			"lastModifiedTime": "2026-10-01T00:00:00Z",
			"nextPageKey":      nil,
		})
	})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewSubscriptionHandler(c)
	h.V3BaseURL = baseURL + "/sub/v3/accounts/test-uuid"

	result, err := h.EnvironmentUsage(context.Background(), "s1", "START", "END", []string{"abc"}, nil)
	if err != nil {
		t.Fatalf("EnvironmentUsage() error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("made %d requests, want 2", len(requests))
	}
	// Unlike notifications, the Subscription API needs the full query on every
	// page: page-key alone is rejected with "'endTime' must be provided".
	second := requests[1]
	if second.Get("page-key") != "k2" || second.Get("startTime") != "START" ||
		second.Get("endTime") != "END" || second.Get("environmentIds") != "abc" {
		t.Errorf("second request = %v, want page-key plus the full query", second)
	}
	// 50 is the API's maximum; larger values are a 400.
	if requests[0].Get("page-size") != "50" {
		t.Errorf("page-size = %q, want 50", requests[0].Get("page-size"))
	}

	rows := FlattenEnvironmentData(result, "usage")
	if len(rows) != 2 {
		t.Fatalf("flattened %d rows, want 2", len(rows))
	}
	if rows[0]["capabilityKey"] != "A" || rows[1]["capabilityKey"] != "B" || rows[1]["clusterId"] != "c1" {
		t.Errorf("rows = %v, want A then B (with cluster c1)", rows)
	}
	for _, r := range rows {
		if r["environmentId"] != "abc" {
			t.Errorf("row %v lost its environmentId", r)
		}
	}
}

func TestFlattenEnvironmentData(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]any
		itemsKey string
		want     int
	}{
		{name: "should return no rows for no data", result: map[string]any{"data": []any{}}, itemsKey: "usage", want: 0},
		{name: "should tolerate a missing data key", result: map[string]any{}, itemsKey: "usage", want: 0},
		{
			name: "should emit one row per cost record across environments",
			result: map[string]any{"data": []any{
				map[string]any{"environmentId": "e1", "cost": []any{
					map[string]any{"value": 1.0, "currencyCode": "USD"},
					map[string]any{"value": 2.0, "currencyCode": "USD"},
				}},
				map[string]any{"environmentId": "e2", "cost": []any{
					map[string]any{"value": 3.0, "currencyCode": "USD"},
				}},
			}},
			itemsKey: "cost",
			want:     3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FlattenEnvironmentData(tt.result, tt.itemsKey); len(got) != tt.want {
				t.Errorf("got %d rows, want %d", len(got), tt.want)
			}
		})
	}
}
