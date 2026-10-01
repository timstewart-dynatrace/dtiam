package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// The Account Management API is not consistent about how it wraps list results,
// and the published documentation does not always match what it returns. Every
// shape below was captured from a live account on 2026-10-01.
//
// These tests exist because six commands silently returned empty lists: the
// handlers read a key the API does not send, which yields an empty slice rather
// than an error. A fixture written from the documentation would have passed
// while the command was broken, so each case here encodes an observed shape.
func TestHandlersReadTheirLiveResponseKey(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		response map[string]any
		handler  func(*testing.T, *http.ServeMux) interface {
			List(context.Context, map[string]string) ([]map[string]any, error)
		}
		wantCount int
	}{
		{
			name: "groups use {count, items}",
			path: "/groups",
			response: map[string]any{
				"count": 2,
				"items": []any{
					map[string]any{"uuid": "g1", "name": "A"},
					map[string]any{"uuid": "g2", "name": "B"},
				},
			},
			handler: func(t *testing.T, m *http.ServeMux) interface {
				List(context.Context, map[string]string) ([]map[string]any, error)
			} {
				return NewGroupHandler(newTestClient(t, m))
			},
			wantCount: 2,
		},
		{
			name: "users use {count, items}",
			path: "/users",
			response: map[string]any{
				"count": 1,
				"items": []any{map[string]any{"uid": "u1", "email": "a@example.com"}},
			},
			handler: func(t *testing.T, m *http.ServeMux) interface {
				List(context.Context, map[string]string) ([]map[string]any, error)
			} {
				return NewUserHandler(newTestClient(t, m))
			},
			wantCount: 1,
		},
		{
			name: "environments use {data} -- not {tenants}",
			path: "/environments",
			response: map[string]any{
				"data": []any{
					map[string]any{"id": "abc12345", "name": "Prod", "active": true},
					map[string]any{"id": "def67890", "name": "Dev", "active": true},
				},
			},
			handler: func(t *testing.T, m *http.ServeMux) interface {
				List(context.Context, map[string]string) ([]map[string]any, error)
			} {
				h := NewEnvironmentHandler(newTestClient(t, m))
				h.Path = "/environments"
				return h
			},
			wantCount: 2,
		},
		{
			name: "boundaries use a Spring page keyed {content} -- not {boundaries}",
			path: "/boundaries",
			response: map[string]any{
				"pageSize":   100,
				"pageNumber": 1,
				"totalCount": 2,
				"content": []any{
					map[string]any{"uuid": "b1", "name": "Prod Only"},
					map[string]any{"uuid": "b2", "name": "Dev Only"},
				},
			},
			handler: func(t *testing.T, m *http.ServeMux) interface {
				List(context.Context, map[string]string) ([]map[string]any, error)
			} {
				h := NewBoundaryHandler(newTestClient(t, m))
				h.Path = "/boundaries"
				return h
			},
			wantCount: 2,
		},
		{
			name: "limits use {results} with limitType -- not {items} with name",
			path: "/limits",
			response: map[string]any{
				"pageSize":   1000,
				"pageNumber": 1,
				"total":      2,
				"results": []any{
					map[string]any{"limitType": "REGULAR_USER", "currentValue": 149, "limitValue": 10000},
					map[string]any{"limitType": "MAX_POLICIES", "currentValue": 10, "limitValue": 100},
				},
			},
			handler: func(t *testing.T, m *http.ServeMux) interface {
				List(context.Context, map[string]string) ([]map[string]any, error)
			} {
				return NewLimitsHandler(newTestClient(t, m))
			},
			wantCount: 2,
		},
		{
			name: "service users use {results, nextPageKey, totalCount} -- not {items}",
			path: "/service-users",
			response: map[string]any{
				"results":    []any{map[string]any{"uid": "su1", "name": "CI"}},
				"totalCount": 1,
			},
			handler: func(t *testing.T, m *http.ServeMux) interface {
				List(context.Context, map[string]string) ([]map[string]any, error)
			} {
				return NewServiceUserHandler(newTestClient(t, m))
			},
			wantCount: 1,
		},
		{
			name: "platform tokens use {pageSize, pageNumber, total, results} -- not {items}",
			path: "/platform-tokens",
			response: map[string]any{
				"pageSize":   1000,
				"pageNumber": 1,
				"total":      1,
				"results":    []any{map[string]any{"tokenId": "dt0s16.ABC", "name": "CI"}},
			},
			handler: func(t *testing.T, m *http.ServeMux) interface {
				List(context.Context, map[string]string) ([]map[string]any, error)
			} {
				return NewTokenHandler(newTestClient(t, m))
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(tt.path, func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(tt.response)
			})

			items, err := tt.handler(t, mux).List(context.Background(), nil)
			if err != nil {
				t.Fatalf("List() error: %v", err)
			}
			if len(items) != tt.wantCount {
				t.Fatalf("List() returned %d items, want %d -- the handler is reading the wrong response key",
					len(items), tt.wantCount)
			}
		})
	}
}

// TestSubscriptionHandlerReadsDataKey covers subscriptions separately: its List
// has a different signature path than the generic handlers above.
func TestSubscriptionHandlerReadsDataKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		// Live shape: {data: [...]}, not {items} or {subscriptions}.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{"uuid": "s1", "name": "DPS 1", "status": "ACTIVE"},
				map[string]any{"uuid": "s2", "name": "DPS 2", "status": "EXPIRED"},
			},
		})
	})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewSubscriptionHandler(c)
	h.Path = baseURL + "/subscriptions"

	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(items))
	}
}

// TestConstructorsDeclareTheirLiveListKey guards the configuration itself, so a
// handler cannot silently revert to the documented-but-wrong key.
func TestConstructorsDeclareTheirLiveListKey(t *testing.T) {
	c := newTestClient(t, http.NewServeMux())

	cases := []struct {
		resource    string
		gotListKey  string
		wantListKey string
		gotIDField  string
		wantIDField string
	}{
		{"group", NewGroupHandler(c).ListKey, "items", NewGroupHandler(c).IDField, "uuid"},
		{"user", NewUserHandler(c).ListKey, "items", NewUserHandler(c).IDField, "uid"},
		{"environment", NewEnvironmentHandler(c).ListKey, "data", NewEnvironmentHandler(c).IDField, "id"},
		{"boundary", NewBoundaryHandler(c).ListKey, "content", NewBoundaryHandler(c).IDField, "uuid"},
		{"limit", NewLimitsHandler(c).ListKey, "results", NewLimitsHandler(c).IDField, "limitType"},
		{"service-user", NewServiceUserHandler(c).ListKey, "results", NewServiceUserHandler(c).IDField, "uid"},
		{"platform-token", NewTokenHandler(c).ListKey, "results", NewTokenHandler(c).IDField, "tokenId"},
		{"subscription", NewSubscriptionHandler(c).ListKey, "data", NewSubscriptionHandler(c).IDField, "uuid"},
	}

	for _, tt := range cases {
		t.Run(tt.resource, func(t *testing.T) {
			if tt.gotListKey != tt.wantListKey {
				t.Errorf("ListKey = %q, want %q", tt.gotListKey, tt.wantListKey)
			}
			if tt.gotIDField != tt.wantIDField {
				t.Errorf("IDField = %q, want %q", tt.gotIDField, tt.wantIDField)
			}
		})
	}
}

// TestPaginatedEndpointsDeclarePagination guards the second half of the same
// bug: an endpoint that pages but is not configured to returns only page one.
func TestPaginatedEndpointsDeclarePagination(t *testing.T) {
	c := newTestClient(t, http.NewServeMux())

	paginated := map[string]*BaseHandler{
		"boundary":       &NewBoundaryHandler(c).BaseHandler,
		"limit":          &NewLimitsHandler(c).BaseHandler,
		"service-user":   &NewServiceUserHandler(c).BaseHandler,
		"platform-token": &NewTokenHandler(c).BaseHandler,
	}
	for name, h := range paginated {
		t.Run(name+" paginates", func(t *testing.T) {
			if h.Pagination == nil {
				t.Errorf("%s paginates but Pagination is nil; only the first page would be returned", name)
			}
		})
	}

	unpaginated := map[string]*BaseHandler{
		"group": &NewGroupHandler(c).BaseHandler,
		"user":  &NewUserHandler(c).BaseHandler,
	}
	for name, h := range unpaginated {
		t.Run(name+" does not paginate", func(t *testing.T) {
			if h.Pagination != nil {
				t.Errorf("%s exposes no paging parameters; Pagination should stay nil", name)
			}
		})
	}
}
