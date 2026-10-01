package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// newTestReferenceHandler points a reference handler at the test server.
func newTestReferenceHandler(t *testing.T, mux *http.ServeMux) *ReferenceHandler {
	t.Helper()
	c, baseURL := newTestClientAndURL(t, mux)
	h := NewReferenceHandler(c)
	h.BaseURL = baseURL + "/ref/v1/account"
	return h
}

func TestReferenceHandler_ListPermissionsParsesBareArray(t *testing.T) {
	// The reference API returns a bare array, not the {count, items} envelope the
	// rest of the Account Management API uses.
	mux := http.NewServeMux()
	mux.HandleFunc("/ref/v1/account/permissions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{
			map[string]any{"id": "account-user-management", "description": "Manage users"},
			map[string]any{"id": "tenant-viewer", "description": "View environment"},
		})
	})

	perms, err := newTestReferenceHandler(t, mux).ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("ListPermissions() error: %v", err)
	}
	if len(perms) != 2 {
		t.Fatalf("got %d permissions, want 2", len(perms))
	}
	if perms[0]["id"] != "account-user-management" {
		t.Errorf("perms[0].id = %v", perms[0]["id"])
	}
}

func TestReferenceHandler_ListPermissionsToleratesWrappedEnvelope(t *testing.T) {
	// Accept a wrapped shape too, so a future API change does not break the
	// command outright.
	mux := http.NewServeMux()
	mux.HandleFunc("/ref/v1/account/permissions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"permissions": []any{map[string]any{"id": "a", "description": "A"}},
		})
	})

	perms, err := newTestReferenceHandler(t, mux).ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("ListPermissions() error: %v", err)
	}
	if len(perms) != 1 || perms[0]["id"] != "a" {
		t.Errorf("got %v, want one permission with id 'a'", perms)
	}
}

func TestReferenceHandler_ListPermissionsHandlesEmptyArray(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ref/v1/account/permissions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	})

	perms, err := newTestReferenceHandler(t, mux).ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("ListPermissions() error: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("got %d permissions, want 0", len(perms))
	}
}

func TestReferenceHandler_PermissionIDsSkipsBlankAndMissingIDs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ref/v1/account/permissions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{
			map[string]any{"id": "a", "description": "A"},
			map[string]any{"id": "", "description": "blank id"},
			map[string]any{"description": "no id at all"},
			map[string]any{"id": "b", "description": "B"},
		})
	})

	ids, err := newTestReferenceHandler(t, mux).PermissionIDs(context.Background())
	if err != nil {
		t.Fatalf("PermissionIDs() error: %v", err)
	}
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Errorf("ids = %v, want [a b]", ids)
	}
}

func TestReferenceHandler_APIPathDefaultsToRefBaseURL(t *testing.T) {
	h := NewReferenceHandler(newTestClient(t, http.NewServeMux()))
	want := "https://api.dynatrace.com/ref/v1/account/permissions"
	if got := h.APIPath(); got != want {
		t.Errorf("APIPath() = %q, want %q", got, want)
	}

	// An empty BaseURL must still resolve rather than produce "/permissions".
	h.BaseURL = ""
	if got := h.APIPath(); got != want {
		t.Errorf("APIPath() with empty BaseURL = %q, want %q", got, want)
	}
}

func TestReferenceHandler_ListPermissionsReportsParseFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ref/v1/account/permissions", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})

	if _, err := newTestReferenceHandler(t, mux).ListPermissions(context.Background()); err == nil {
		t.Fatal("expected a parse error for a non-JSON response")
	}
}
