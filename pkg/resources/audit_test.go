package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestAuditQuery_ParamsOmitsUnsetFields(t *testing.T) {
	// Unset fields must not be sent, so the server applies its own defaults
	// rather than receiving "0" or "".
	got := AuditQuery{}.params()
	if len(got) != 0 {
		t.Errorf("empty query produced params %v, want none", got)
	}
}

func TestAuditQuery_ParamsMapsEveryField(t *testing.T) {
	q := AuditQuery{
		StartTime:               "now-24h",
		EndTime:                 "now",
		Filter:                  `eventType = "DELETE"`,
		AddFields:               []string{"details", "originAddress"},
		Limit:                   50,
		ScanLimitGigabyte:       5,
		ResultSizeLimitMegabyte: 10,
	}

	want := map[string]string{
		"startTime":               "now-24h",
		"endTime":                 "now",
		"filter":                  `eventType = "DELETE"`,
		"addFields":               "details,originAddress",
		"limit":                   "50",
		"scanLimitGigabyte":       "5",
		"resultSizeLimitMegabyte": "10",
	}

	got := q.params()
	for k, v := range want {
		if got[k] != v {
			t.Errorf("params[%q] = %q, want %q", k, got[k], v)
		}
	}
}

func TestAuditQuery_ParamsSkipsEmptyAddFields(t *testing.T) {
	q := AuditQuery{AddFields: []string{"details", "", "user"}}
	if got := q.params()["addFields"]; got != "details,user" {
		t.Errorf("addFields = %q, want 'details,user' with the empty entry dropped", got)
	}
}

func TestAuditHandler_QueryReturnsEntries(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/audit/v1/accounts/test-uuid", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("startTime"); got != "now-24h" {
			t.Errorf("startTime = %q, want 'now-24h'", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"audits": []any{
				map[string]any{"eventId": "e1", "eventType": "CREATE", "user": "alice@example.com"},
				map[string]any{"eventId": "e2", "eventType": "DELETE", "user": "bob@example.com"},
			},
		})
	})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewAuditHandler(c)
	h.Path = baseURL + "/audit/v1/accounts/test-uuid"

	result, err := h.Query(context.Background(), AuditQuery{StartTime: "now-24h"})
	if err != nil {
		t.Fatalf("Query() error: %v", err)
	}
	if len(result.Audits) != 2 {
		t.Fatalf("got %d audits, want 2", len(result.Audits))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("got warnings %v, want none", result.Warnings)
	}
}

func TestAuditHandler_QuerySurfacesWarnings(t *testing.T) {
	// A partial result arrives as HTTP 200 with warnings. Dropping them would
	// present a truncated audit trail as complete.
	mux := http.NewServeMux()
	mux.HandleFunc("/audit/v1/accounts/test-uuid", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"audits": []any{map[string]any{"eventId": "e1"}},
			"warnings": []any{
				map[string]any{"message": "scan limit reached"},
				map[string]any{"warning": "result truncated"},
			},
		})
	})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewAuditHandler(c)
	h.Path = baseURL + "/audit/v1/accounts/test-uuid"

	result, err := h.Query(context.Background(), AuditQuery{})
	if err != nil {
		t.Fatalf("Query() error: %v", err)
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("got warnings %v, want 2 (one from 'message', one from 'warning')", result.Warnings)
	}
	if result.Warnings[0] != "scan limit reached" {
		t.Errorf("warnings[0] = %q", result.Warnings[0])
	}
	if result.Warnings[1] != "result truncated" {
		t.Errorf("warnings[1] = %q, want the 'warning' field to be read too", result.Warnings[1])
	}
}

func TestNewAuditHandler_UsesAuditBaseURL(t *testing.T) {
	h := NewAuditHandler(newTestClient(t, http.NewServeMux()))
	if h.ListKey != "audits" {
		t.Errorf("ListKey = %q, want 'audits'", h.ListKey)
	}
	if h.IDField != "eventId" {
		t.Errorf("IDField = %q, want 'eventId'", h.IDField)
	}
}
