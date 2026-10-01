package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestValidatePermissions(t *testing.T) {
	tests := []struct {
		name    string
		perm    GroupPermission
		wantErr string
	}{
		{
			name: "should accept a tenant-scoped grant",
			perm: GroupPermission{PermissionName: "tenant-viewer", Scope: "abc12345", ScopeType: ScopeTypeTenant},
		},
		{
			name: "should accept an account-scoped grant",
			perm: GroupPermission{PermissionName: "account-viewer", Scope: "uuid-1", ScopeType: ScopeTypeAccount},
		},
		{
			name: "should accept a management-zone scope with a colon",
			perm: GroupPermission{PermissionName: "tenant-viewer", Scope: "abc12345:-123", ScopeType: ScopeTypeManagementZone},
		},
		{
			name:    "should reject a missing permission name",
			perm:    GroupPermission{Scope: "abc12345", ScopeType: ScopeTypeTenant},
			wantErr: "permissionName is required",
		},
		{
			name:    "should reject an unknown scope type",
			perm:    GroupPermission{PermissionName: "tenant-viewer", Scope: "abc", ScopeType: "environment"},
			wantErr: "invalid scopeType",
		},
		{
			name:    "should reject a missing scope",
			perm:    GroupPermission{PermissionName: "tenant-viewer", ScopeType: ScopeTypeTenant},
			wantErr: "scope is required",
		},
		{
			name: "should reject a management-zone scope without a colon",
			perm: GroupPermission{
				PermissionName: "tenant-viewer", Scope: "abc12345", ScopeType: ScopeTypeManagementZone,
			},
			wantErr: "management-zone scope must be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePermissions([]GroupPermission{tt.perm})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestGroupPermissionHandler_ListReadsPermissionsKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/permissions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"permissions": []any{
				map[string]any{"permissionName": "tenant-viewer", "scope": "abc", "scopeType": "tenant"},
			},
		})
	})

	h := NewGroupPermissionHandler(newTestClient(t, mux))
	perms, err := h.List(context.Background(), "g1")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(perms) != 1 || perms[0].PermissionName != "tenant-viewer" {
		t.Fatalf("got %+v, want one tenant-viewer grant", perms)
	}
}

func TestGroupPermissionHandler_ListAcceptsBareArray(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/permissions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{
			map[string]any{"permissionName": "account-viewer", "scope": "u", "scopeType": "account"},
		})
	})

	h := NewGroupPermissionHandler(newTestClient(t, mux))
	perms, err := h.List(context.Background(), "g1")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(perms) != 1 {
		t.Fatalf("got %d perms, want 1", len(perms))
	}
}

func TestGroupPermissionHandler_GrantPostsPermissionArray(t *testing.T) {
	var got []GroupPermission
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/permissions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
	})

	h := NewGroupPermissionHandler(newTestClient(t, mux))
	perm := GroupPermission{PermissionName: "tenant-viewer", Scope: "abc", ScopeType: ScopeTypeTenant}
	if err := h.Grant(context.Background(), "g1", []GroupPermission{perm}); err != nil {
		t.Fatalf("Grant() error: %v", err)
	}
	if len(got) != 1 || got[0].PermissionName != "tenant-viewer" {
		t.Errorf("request body = %+v, want a one-element array", got)
	}
}

func TestGroupPermissionHandler_GrantRejectsEmptySet(t *testing.T) {
	h := NewGroupPermissionHandler(newTestClient(t, http.NewServeMux()))
	if err := h.Grant(context.Background(), "g1", nil); err == nil {
		t.Fatal("expected an error when granting nothing")
	}
}

func TestGroupPermissionHandler_GrantValidatesBeforeSending(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/permissions", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	h := NewGroupPermissionHandler(newTestClient(t, mux))
	bad := GroupPermission{PermissionName: "x", Scope: "y", ScopeType: "nonsense"}
	if err := h.Grant(context.Background(), "g1", []GroupPermission{bad}); err == nil {
		t.Fatal("expected a validation error")
	}
	if called {
		t.Error("an invalid grant must not reach the API")
	}
}

func TestGroupPermissionHandler_ReplaceRefusesEmptySet(t *testing.T) {
	// An empty PUT would strip every grant. That is too destructive to infer
	// from an omitted argument.
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/permissions", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	h := NewGroupPermissionHandler(newTestClient(t, mux))
	err := h.Replace(context.Background(), "g1", nil)
	if err == nil {
		t.Fatal("expected Replace to refuse an empty set")
	}
	if !strings.Contains(err.Error(), "revoke") {
		t.Errorf("error = %q, want it to point at revoke", err)
	}
	if called {
		t.Error("Replace must not issue a request for an empty set")
	}
}

func TestGroupPermissionHandler_ReplaceUsesPut(t *testing.T) {
	method := ""
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/permissions", func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.WriteHeader(http.StatusOK)
	})

	h := NewGroupPermissionHandler(newTestClient(t, mux))
	perm := GroupPermission{PermissionName: "tenant-viewer", Scope: "abc", ScopeType: ScopeTypeTenant}
	if err := h.Replace(context.Background(), "g1", []GroupPermission{perm}); err != nil {
		t.Fatalf("Replace() error: %v", err)
	}
	if method != http.MethodPut {
		t.Errorf("method = %s, want PUT", method)
	}
}

func TestGroupPermissionHandler_RevokeSendsIdentifyingQueryParams(t *testing.T) {
	var query url.Values
	method := ""
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/permissions", func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		query = r.URL.Query()
		w.WriteHeader(http.StatusNoContent)
	})

	h := NewGroupPermissionHandler(newTestClient(t, mux))
	err := h.Revoke(context.Background(), "g1", "tenant-viewer", "abc12345", ScopeTypeTenant)
	if err != nil {
		t.Fatalf("Revoke() error: %v", err)
	}
	if method != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", method)
	}
	// A grant has no ID of its own, so all three fields identify it.
	for k, want := range map[string]string{
		"permission-name": "tenant-viewer",
		"scope":           "abc12345",
		"scope-type":      ScopeTypeTenant,
	} {
		if got := query.Get(k); got != want {
			t.Errorf("query %s = %q, want %q", k, got, want)
		}
	}
}

func TestGroupPermissionHandler_RevokeRequiresPermissionName(t *testing.T) {
	h := NewGroupPermissionHandler(newTestClient(t, http.NewServeMux()))
	if err := h.Revoke(context.Background(), "g1", "", "abc", ScopeTypeTenant); err == nil {
		t.Fatal("expected an error for an empty permission name")
	}
}

func TestGroupPermissionHandler_RevokeValidatesScopeType(t *testing.T) {
	h := NewGroupPermissionHandler(newTestClient(t, http.NewServeMux()))
	if err := h.Revoke(context.Background(), "g1", "tenant-viewer", "abc", "bogus"); err == nil {
		t.Fatal("expected an error for an invalid scope type")
	}
}
