package resources

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

// The tests in this file pin API behavior verified against a live account on
// 2026-10-02 that the published documentation either omits or contradicts.
// Each one guards a command that was broken until 3.0.1.

// TestCollectionsWithoutSingleGetResolveFromList covers groups, environments
// and platform tokens. Their {path}/{id} GET does not exist and answers 404 for
// every ID, which made `get groups ID`, `describe environment ID`,
// `get tokens ID`, `analyze effective-group` and `export group` all report a
// valid ID as not found.
func TestCollectionsWithoutSingleGetResolveFromList(t *testing.T) {
	tests := []struct {
		name    string
		listKey string
		item    map[string]any
		id      string
		build   func(c *client.Client, base string) Getter
	}{
		{
			name:    "should resolve a group by uuid from the group list",
			listKey: "items",
			item:    map[string]any{"uuid": "g1", "name": "Admins"},
			id:      "g1",
			build: func(c *client.Client, _ string) Getter {
				return NewGroupHandler(c)
			},
		},
		{
			name:    "should resolve an environment by id from the environment list",
			listKey: "data",
			item:    map[string]any{"id": "abc12345", "name": "Prod"},
			id:      "abc12345",
			build: func(c *client.Client, base string) Getter {
				h := NewEnvironmentHandler(c)
				h.Path = base + "/groups" // reuse the mux route below
				return h
			},
		},
		{
			name:    "should resolve a platform token by tokenId from the token list",
			listKey: "results",
			item:    map[string]any{"tokenId": "dt0s16.ABC", "name": "ci"},
			id:      "dt0s16.ABC",
			build: func(c *client.Client, base string) Getter {
				h := NewTokenHandler(c)
				h.Path = base + "/groups"
				return h
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{
					tt.listKey: []any{map[string]any{"uuid": "other", "id": "other", "tokenId": "other"}, tt.item},
				})
			})
			mux.HandleFunc("/groups/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("Get() called the nonexistent single-item endpoint %s", r.URL.Path)
				w.WriteHeader(404)
			})

			c, base := newTestClientAndURL(t, mux)
			h := tt.build(c, base)

			got, err := h.Get(context.Background(), tt.id)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", tt.id, err)
			}
			if got["name"] != tt.item["name"] {
				t.Errorf("Get(%q) = %v, want %v", tt.id, got, tt.item)
			}

			_, err = h.Get(context.Background(), "missing")
			if err == nil || !strings.Contains(err.Error(), "not found") {
				t.Errorf("Get(missing) error = %v, want a not-found error", err)
			}
		})
	}
}

// TestGetOrResolve_FallsBackOnBadRequest covers name lookups for policies and
// boundaries: their GET-by-ID answers a non-UUID with 400 "Validation failed
// (uuid is expected)", not 404, so `get policies NAME` failed outright.
func TestGetOrResolve_FallsBackOnBadRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/policies", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"policies": []any{map[string]any{"uuid": "p1", "name": "Read Only"}},
		})
	})
	mux.HandleFunc("/policies/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":true,"message":"Validation failed (uuid is expected)","payload":null}`))
	})

	c := newTestClient(t, mux)
	h := &BaseHandler{Client: c, Name: "policy", Path: "/policies", ListKey: "policies", IDField: "uuid", NameField: "name"}

	got, err := GetOrResolve(context.Background(), h, "Read Only")
	if err != nil {
		t.Fatalf("GetOrResolve() error: %v", err)
	}
	if got == nil || got["uuid"] != "p1" {
		t.Errorf("GetOrResolve() = %v, want policy p1", got)
	}
}

// TestHandleError_KeepsAPIErrorReachable makes sure the friendlier message does
// not hide the status code from callers that need to branch on it.
func TestHandleError_KeepsAPIErrorReachable(t *testing.T) {
	h := &BaseHandler{Name: "group"}
	err := h.handleError("get", &client.APIError{StatusCode: 403, Message: "no scope"})

	if err.Error() != "permission denied: no scope" {
		t.Errorf("Error() = %q, want %q", err.Error(), "permission denied: no scope")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 403 {
		t.Errorf("errors.As did not reach the APIError: %v", err)
	}
}

// TestGetPolicyByName_ReturnsFullRecord covers apply and diff: the policy list
// omits statementQuery, so an update built from a list entry failed with
// "Empty policy statement" and diff always reported the statement as changed.
func TestGetPolicyByName_ReturnsFullRecord(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/policies", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"policies": []any{map[string]any{"uuid": "p1", "name": "Read Only", "description": ""}},
		})
	})
	mux.HandleFunc("/policies/p1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid": "p1", "name": "Read Only", "description": "", "statementQuery": "ALLOW iam:policies:read;",
		})
	})

	c := newTestClient(t, mux)
	h := NewPolicyHandler(c)
	h.Path = "/policies"

	got, err := GetPolicyByName(context.Background(), h, "Read Only")
	if err != nil {
		t.Fatalf("GetPolicyByName() error: %v", err)
	}
	if got["statementQuery"] != "ALLOW iam:policies:read;" {
		t.Errorf("GetPolicyByName() = %v, want the full record with statementQuery", got)
	}

	missing, err := GetPolicyByName(context.Background(), h, "nope")
	if err != nil || missing != nil {
		t.Errorf("GetPolicyByName(nope) = %v, %v; want nil, nil", missing, err)
	}
}
