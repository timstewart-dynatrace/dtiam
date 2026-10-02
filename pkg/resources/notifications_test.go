package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func newTestNotificationHandler(t *testing.T, mux *http.ServeMux) *NotificationHandler {
	t.Helper()
	c, baseURL := newTestClientAndURL(t, mux)
	h := NewNotificationHandler(c)
	h.BaseURL = baseURL + "/v2/accounts"
	return h
}

func TestNotificationQuery_Validate(t *testing.T) {
	tests := []struct {
		name    string
		query   NotificationQuery
		wantErr string
	}{
		{name: "should accept an empty query", query: NotificationQuery{}},
		{
			name:  "should accept valid types and severities",
			query: NotificationQuery{Types: []string{NotificationBudget, NotificationCost}, Severities: []string{SeveritySevere}},
		},
		{
			name:  "should accept the environment upgrade and downgrade types added in v2",
			query: NotificationQuery{Types: []string{NotificationEnvironmentUpgrade, NotificationEnvironmentDowngrade}},
		},
		{
			name:    "should reject an unknown type",
			query:   NotificationQuery{Types: []string{"BUDGETS"}},
			wantErr: "invalid notification type",
		},
		{
			name:    "should reject an unknown severity",
			query:   NotificationQuery{Severities: []string{"CRITICAL"}},
			wantErr: "invalid severity",
		},
		{
			name:    "should reject lowercase values, which the API does not accept",
			query:   NotificationQuery{Types: []string{"budget"}},
			wantErr: "invalid notification type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestNotificationQuery_QueryOmitsEmptyFilters(t *testing.T) {
	if got := (NotificationQuery{}).query(); len(got) != 0 {
		t.Errorf("empty query produced params %v, want none", got)
	}
}

func TestNotificationQuery_QueryRepeatsListFilters(t *testing.T) {
	q := NotificationQuery{
		StartDateTime: "2026-09-01T00:00:00Z",
		EndDateTime:   "2026-10-01T00:00:00Z",
		Types:         []string{NotificationBudget, NotificationForecast},
		Severities:    []string{SeverityWarn},
		Environments:  []string{"abc12345"},
		Capabilities:  []string{"FULLSTACK_MONITORING"},
	}
	got := q.query()

	tests := []struct {
		param string
		want  []string
	}{
		{"start-time", []string{"2026-09-01T00:00:00Z"}},
		{"end-time", []string{"2026-10-01T00:00:00Z"}},
		// Repeated, not comma-joined: the v2 API answers types=A,B with 400.
		{"types", []string{"BUDGET", "FORECAST"}},
		{"severities", []string{"WARN"}},
		{"environments", []string{"abc12345"}},
		{"capabilities", []string{"FULLSTACK_MONITORING"}},
	}
	for _, tt := range tests {
		if v := got[tt.param]; strings.Join(v, "|") != strings.Join(tt.want, "|") {
			t.Errorf("%s = %v, want %v", tt.param, v, tt.want)
		}
	}
}

func TestNotificationHandler_QueryUsesV2GetAndFollowsPages(t *testing.T) {
	var requests []url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/accounts/test-uuid/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET (the v1 POST is deprecated)", r.Method)
		}
		q := r.URL.Query()
		requests = append(requests, q)
		if q.Get("page-key") == "" {
			// Live shape: no totalRecordCount, despite the spec.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"records":     []any{map[string]any{"key": "n1", "type": "budget"}},
				"hasNextPage": true,
				"nextPageKey": "cursor-2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"records":     []any{map[string]any{"key": "n2", "type": "budget"}},
			"hasNextPage": false,
		})
	})

	result, err := newTestNotificationHandler(t, mux).Query(
		context.Background(), NotificationQuery{Types: []string{NotificationBudget}})
	if err != nil {
		t.Fatalf("Query() error: %v", err)
	}
	if len(result.Records) != 2 || result.TotalCount != 2 || result.HasMore {
		t.Errorf("result = %d records, TotalCount %d, HasMore %v; want 2, 2, false",
			len(result.Records), result.TotalCount, result.HasMore)
	}
	if len(requests) != 2 {
		t.Fatalf("made %d requests, want 2", len(requests))
	}
	if requests[0].Get("types") != "BUDGET" || requests[0].Get("page-size") == "" {
		t.Errorf("first request = %v, want the filters and a page size", requests[0])
	}
	// The cursor carries the query; resending filters with it is a 400.
	if len(requests[1]) != 1 || requests[1].Get("page-key") != "cursor-2" {
		t.Errorf("second request = %v, want page-key alone", requests[1])
	}
}

func TestNotificationHandler_QueryValidatesBeforeSending(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/accounts/test-uuid/notifications", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	h := newTestNotificationHandler(t, mux)
	if _, err := h.Query(context.Background(), NotificationQuery{Types: []string{"NOPE"}}); err == nil {
		t.Fatal("expected a validation error")
	}
	if called {
		t.Error("an invalid filter must not reach the API")
	}
}

func TestNotificationHandler_APIPathUsesV2(t *testing.T) {
	h := NewNotificationHandler(newTestClient(t, http.NewServeMux()))
	want := "https://api.dynatrace.com/v2/accounts/test-uuid/notifications"
	if got := h.APIPath(); got != want {
		t.Errorf("APIPath() = %q, want %q", got, want)
	}
}
