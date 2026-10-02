package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func newTestGroupHandler(t *testing.T, mux *http.ServeMux) *GroupHandler {
	t.Helper()
	c := newTestClient(t, mux)
	return NewGroupHandler(c)
}

func TestGroupHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uuid": "g1", "name": "Admins"},
				map[string]any{"uuid": "g2", "name": "Readers"},
			},
		})
	})

	h := newTestGroupHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(items))
	}
}

func TestGroupHandler_Get_ResolvesFromList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uuid": "g0", "name": "Other"},
				map[string]any{"uuid": "g1", "name": "Admins"},
			},
		})
	})
	// GET /groups/{uuid} does not exist; the live API answers 404 for every
	// UUID. Get must never call it.
	mux.HandleFunc("/groups/g1", func(w http.ResponseWriter, r *http.Request) {
		t.Error("Get() called the nonexistent GET /groups/{uuid} endpoint")
		w.WriteHeader(404)
	})

	h := newTestGroupHandler(t, mux)
	item, err := h.Get(context.Background(), "g1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["name"] != "Admins" {
		t.Errorf("Get() name = %v, want Admins", item["name"])
	}

	if _, err := h.Get(context.Background(), "missing"); err == nil {
		t.Error("Get() of an unknown UUID returned no error")
	}
}

func TestGroupHandler_Create_SendsArray(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		// POST /groups takes and returns an array; a bare object makes the
		// live API fail with 500 "payload.map is not a function".
		var body []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"error":true,"message":"payload.map is not a function","payload":null}`))
			return
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode([]any{map[string]any{"uuid": "g-new", "name": body[0]["name"]}})
	})

	h := newTestGroupHandler(t, mux)
	result, err := h.Create(context.Background(), map[string]any{"name": "NewGroup"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uuid"] != "g-new" {
		t.Errorf("Create() uuid = %v, want g-new", result["uuid"])
	}
}

func TestGroupHandler_Create_MissingName(t *testing.T) {
	mux := http.NewServeMux()
	h := newTestGroupHandler(t, mux)
	_, err := h.Create(context.Background(), map[string]any{"description": "no name"})
	if err == nil {
		t.Fatal("Create() expected error for missing name, got nil")
	}
}

func TestGroupHandler_Delete_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestGroupHandler(t, mux)
	err := h.Delete(context.Background(), "g1")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestGroupHandler_GetMembers_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/users", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uid": "u1", "email": "alice@example.com"},
				map[string]any{"uid": "u2", "email": "bob@example.com"},
			},
		})
	})

	h := newTestGroupHandler(t, mux)
	members, err := h.GetMembers(context.Background(), "g1")
	if err != nil {
		t.Fatalf("GetMembers() error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("GetMembers() returned %d members, want 2", len(members))
	}
}

func TestGroupHandler_GetMembers_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"message":"server error"}`))
	})

	h := newTestGroupHandler(t, mux)
	_, err := h.GetMembers(context.Background(), "g1")
	if err == nil {
		t.Fatal("GetMembers() expected error for 500, got nil")
	}
}

func TestGroupHandler_AddMember_PostsGroupToUser(t *testing.T) {
	var body []string
	mux := http.NewServeMux()
	// Membership is added through the user: POST /users/{email} with a list of
	// group UUIDs. There is no POST on /groups/{uuid}/users.
	mux.HandleFunc("/users/alice@example.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(201)
	})

	h := newTestGroupHandler(t, mux)
	if err := h.AddMember(context.Background(), "g1", "alice@example.com"); err != nil {
		t.Fatalf("AddMember() error: %v", err)
	}
	if len(body) != 1 || body[0] != "g1" {
		t.Errorf("AddMember() body = %v, want [g1]", body)
	}
}

func TestGroupHandler_RemoveMember(t *testing.T) {
	tests := []struct {
		name string
		user string
	}{
		{name: "should remove a user given by email", user: "alice@example.com"},
		{name: "should resolve a UID to an email before removing", user: "u1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var groupParam string
			mux := http.NewServeMux()
			mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("service-users") != "true" {
					t.Error("UID lookup must include service users")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"items": []any{map[string]any{"uid": "u1", "email": "alice@example.com"}},
				})
			})
			mux.HandleFunc("/users/alice@example.com/groups", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					w.WriteHeader(405)
					return
				}
				groupParam = r.URL.Query().Get("group-uuid")
				w.WriteHeader(200)
			})

			h := newTestGroupHandler(t, mux)
			if err := h.RemoveMember(context.Background(), "g1", tt.user); err != nil {
				t.Fatalf("RemoveMember() error: %v", err)
			}
			if groupParam != "g1" {
				t.Errorf("RemoveMember() group-uuid = %q, want g1", groupParam)
			}
		})
	}
}

func TestGroupHandler_GetPolicies_UsesRepoPathAndPolicyUuids(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/groups/g1", func(w http.ResponseWriter, r *http.Request) {
		// Live shape: one entry per binding, so a parameterized policy can
		// appear twice; GetPolicies reports each policy once.
		_, _ = w.Write([]byte(`{"policyUuids":["p1","p2"],"bindingsDetails":[
			{"policyUuid":"p1","groups":["g1"]},{"policyUuid":"p1","groups":["g1"]},
			{"policyUuid":"p2","groups":["g1"]}]}`))
	})

	h := newTestGroupHandler(t, mux)
	h.bindingsPath = "/repo/account/test-uuid/bindings"
	policies, err := h.GetPolicies(context.Background(), "g1")
	if err != nil {
		t.Fatalf("GetPolicies() error: %v", err)
	}
	if len(policies) != 2 || policies[0] != "p1" || policies[1] != "p2" {
		t.Errorf("GetPolicies() = %v, want [p1 p2]", policies)
	}
}

func TestGroupHandler_DefaultBindingsPathIsRepoLevel(t *testing.T) {
	h := NewGroupHandler(newTestClient(t, http.NewServeMux()))
	// The old relative path resolved under /accounts/{uuid}/ and 404ed.
	want := "https://api.dynatrace.com/iam/v1/repo/account/test-uuid/bindings"
	if got := h.bindingHandler().Path; got != want {
		t.Errorf("binding path = %q, want %q", got, want)
	}
}

func TestGroupHandler_Update_KeepsUnspecifiedFields(t *testing.T) {
	var put map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{map[string]any{"uuid": "g1", "name": "Admins", "description": "old"}},
		})
	})
	mux.HandleFunc("/groups/g1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(405)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&put)
		w.WriteHeader(200)
	})

	h := newTestGroupHandler(t, mux)
	if _, err := h.Update(context.Background(), "g1", map[string]any{"description": "new"}); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if put["name"] != "Admins" || put["description"] != "new" || put["uuid"] != "g1" {
		t.Errorf("Update() body = %v, want name Admins, description new, uuid g1", put)
	}
}

func TestGroupHandler_GetMemberCount_WithCountField(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/users", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count": float64(42),
		})
	})

	h := newTestGroupHandler(t, mux)
	count, err := h.GetMemberCount(context.Background(), "g1")
	if err != nil {
		t.Fatalf("GetMemberCount() error: %v", err)
	}
	if count != 42 {
		t.Errorf("GetMemberCount() = %d, want 42", count)
	}
}

func TestGroupHandler_GetMemberCount_FallbackToList(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/users", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call with count=true returns no count field
			_ = json.NewEncoder(w).Encode(map[string]any{"other": "data"})
		} else {
			// Fallback call returns items
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []any{
					map[string]any{"uid": "u1"},
					map[string]any{"uid": "u2"},
					map[string]any{"uid": "u3"},
				},
			})
		}
	})

	h := newTestGroupHandler(t, mux)
	count, err := h.GetMemberCount(context.Background(), "g1")
	if err != nil {
		t.Fatalf("GetMemberCount() error: %v", err)
	}
	if count != 3 {
		t.Errorf("GetMemberCount() = %d, want 3", count)
	}
}
