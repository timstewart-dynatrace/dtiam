package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func newTestNotificationHandler(t *testing.T, mux *http.ServeMux) *NotificationHandler {
	t.Helper()
	c, baseURL := newTestClientAndURL(t, mux)
	h := NewNotificationHandler(c)
	h.BaseURL = baseURL + "/v1/accounts"
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

func TestNotificationQuery_BodyOmitsEmptyFilters(t *testing.T) {
	if got := (NotificationQuery{}).body(); len(got) != 0 {
		t.Errorf("empty query produced body %v, want none", got)
	}
}

func TestNotificationQuery_BodyIncludesEveryFilter(t *testing.T) {
	q := NotificationQuery{
		StartDateTime: "2026-09-01T00:00:00Z",
		EndDateTime:   "2026-10-01T00:00:00Z",
		Types:         []string{NotificationBudget},
		Severities:    []string{SeverityWarn},
	}
	got := q.body()
	if got["startDateTime"] != "2026-09-01T00:00:00Z" {
		t.Errorf("startDateTime = %v", got["startDateTime"])
	}
	if got["endDateTime"] != "2026-10-01T00:00:00Z" {
		t.Errorf("endDateTime = %v", got["endDateTime"])
	}
	if _, ok := got["types"]; !ok {
		t.Error("types missing from body")
	}
	if _, ok := got["severities"]; !ok {
		t.Error("severities missing from body")
	}
}

func TestNotificationHandler_QueryUsesPostWithFilterBody(t *testing.T) {
	// Unusually for a read, the notifications API is a POST with the filter in
	// the body rather than in query parameters.
	var gotBody map[string]any
	method := ""

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/accounts/test-uuid/notifications", func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"records": []any{
				map[string]any{"type": "BUDGET", "severity": "WARN", "message": "80% of budget used"},
			},
			"totalCount": 1,
			"hasMore":    false,
		})
	})

	result, err := newTestNotificationHandler(t, mux).Query(
		context.Background(), NotificationQuery{Types: []string{NotificationBudget}})
	if err != nil {
		t.Fatalf("Query() error: %v", err)
	}
	if method != http.MethodPost {
		t.Errorf("method = %s, want POST", method)
	}
	if len(result.Records) != 1 {
		t.Fatalf("got %d records, want 1", len(result.Records))
	}
	if result.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", result.TotalCount)
	}
	if gotBody["types"] == nil {
		t.Error("the type filter was not sent in the request body")
	}
}

func TestNotificationHandler_QueryReportsHasMore(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/accounts/test-uuid/notifications", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"records":    []any{map[string]any{"type": "COST"}},
			"totalCount": 500,
			"hasMore":    true,
		})
	})

	result, err := newTestNotificationHandler(t, mux).Query(context.Background(), NotificationQuery{})
	if err != nil {
		t.Fatalf("Query() error: %v", err)
	}
	if !result.HasMore {
		t.Error("HasMore = false, want true so the command can warn about truncation")
	}
}

func TestNotificationHandler_QueryValidatesBeforeSending(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/accounts/test-uuid/notifications", func(w http.ResponseWriter, r *http.Request) {
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

func TestNotificationHandler_APIPathIncludesAccountUUID(t *testing.T) {
	h := NewNotificationHandler(newTestClient(t, http.NewServeMux()))
	want := "https://api.dynatrace.com/v1/accounts/test-uuid/notifications"
	if got := h.APIPath(); got != want {
		t.Errorf("APIPath() = %q, want %q", got, want)
	}
}
