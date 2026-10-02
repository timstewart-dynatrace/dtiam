package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func newTestServiceUserHandler(t *testing.T, mux *http.ServeMux) *ServiceUserHandler {
	t.Helper()
	c := newTestClient(t, mux)
	return NewServiceUserHandler(c)
}

func TestServiceUserHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		// Live shape: {results, nextPageKey, totalCount}.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []any{
				map[string]any{"uid": "su1", "name": "CI Bot"},
				map[string]any{"uid": "su2", "name": "Deploy Bot"},
			},
			"totalCount": 2,
		})
	})

	h := newTestServiceUserHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(items))
	}
}

func TestServiceUserHandler_Get_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users/su1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uid":  "su1",
			"name": "CI Bot",
		})
	})

	h := newTestServiceUserHandler(t, mux)
	item, err := h.Get(context.Background(), "su1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["name"] != "CI Bot" {
		t.Errorf("Get() name = %v, want CI Bot", item["name"])
	}
}

func TestServiceUserHandler_GetByName_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uid": "su1", "name": "CI Bot"},
				map[string]any{"uid": "su2", "name": "Deploy Bot"},
			},
		})
	})

	h := newTestServiceUserHandler(t, mux)
	item, err := h.GetByName(context.Background(), "deploy bot")
	if err != nil {
		t.Fatalf("GetByName() error: %v", err)
	}
	if item == nil {
		t.Fatal("GetByName() returned nil")
	}
	if item["uid"] != "su2" {
		t.Errorf("GetByName() uid = %v, want su2", item["uid"])
	}
}

func TestServiceUserHandler_GetByName_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	})

	h := newTestServiceUserHandler(t, mux)
	item, err := h.GetByName(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByName() error: %v", err)
	}
	if item != nil {
		t.Errorf("GetByName() should return nil, got %v", item)
	}
}

// serviceUserMux serves the live shapes for service user "su1": GET
// /service-users/su1 has no groups, and its memberships live on the user
// record at GET /users/{email}.
func serviceUserMux(t *testing.T, onUsers func(w http.ResponseWriter, r *http.Request)) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users/su1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"uid": "su1", "email": "su1@service.sso.dynatrace.com", "name": "CI Bot", "description": "old",
			})
		case http.MethodPut:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] == nil || body["name"] == "" {
				w.WriteHeader(400) // PUT requires name
				return
			}
			if _, ok := body["groups"]; ok {
				t.Error("PUT /service-users/{uid} does not accept groups")
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			w.WriteHeader(405)
		}
	})
	if onUsers != nil {
		mux.HandleFunc("/users/su1@service.sso.dynatrace.com", onUsers)
		mux.HandleFunc("/users/su1@service.sso.dynatrace.com/groups", onUsers)
	}
	return mux
}

func TestServiceUserHandler_Create_AddsGroupsThroughUserEndpoint(t *testing.T) {
	var created map[string]any
	var groupsPosted []string
	mux := serviceUserMux(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&groupsPosted)
			w.WriteHeader(201)
		}
	})
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&created)
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{"uid": "su1", "email": "su1@service.sso.dynatrace.com", "name": "NewBot"})
	})

	h := newTestServiceUserHandler(t, mux)
	desc := "A new service user"
	result, err := h.Create(context.Background(), "NewBot", &desc, []string{"g1"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uid"] != "su1" {
		t.Errorf("Create() uid = %v, want su1", result["uid"])
	}
	if _, ok := created["groups"]; ok {
		t.Error("Create() sent groups to POST /service-users, which accepts only name and description")
	}
	if len(groupsPosted) != 1 || groupsPosted[0] != "g1" {
		t.Errorf("Create() group membership = %v, want [g1] via POST /users/{email}", groupsPosted)
	}
}

func TestServiceUserHandler_Create_MinimalFields(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["description"]; ok {
			t.Error("should not include description when nil")
		}
		if _, ok := body["groups"]; ok {
			t.Error("should not include groups when nil")
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{"uid": "su-new", "name": body["name"]})
	})

	h := newTestServiceUserHandler(t, mux)
	result, err := h.Create(context.Background(), "MinimalBot", nil, nil)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uid"] != "su-new" {
		t.Errorf("Create() uid = %v, want su-new", result["uid"])
	}
}

func TestServiceUserHandler_Delete_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users/su1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestServiceUserHandler(t, mux)
	err := h.Delete(context.Background(), "su1")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestServiceUserHandler_Delete_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestServiceUserHandler(t, mux)
	err := h.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("Delete() expected error for 404, got nil")
	}
}

func TestServiceUserHandler_GetGroups_ReadsUserRecord(t *testing.T) {
	mux := serviceUserMux(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uid":    "su1",
			"email":  "su1@service.sso.dynatrace.com",
			"groups": []any{map[string]any{"uuid": "g1", "groupName": "Admins"}, map[string]any{"uuid": "g2"}},
		})
	})

	h := newTestServiceUserHandler(t, mux)
	groups, err := h.GetGroups(context.Background(), "su1")
	if err != nil {
		t.Fatalf("GetGroups() error: %v", err)
	}
	if len(groups) != 2 || groups[0]["uuid"] != "g1" || groups[1]["uuid"] != "g2" {
		t.Errorf("GetGroups() = %v, want g1 and g2", groups)
	}
}

func TestServiceUserHandler_GetGroups_NoGroups(t *testing.T) {
	mux := serviceUserMux(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"uid": "su1", "email": "su1@service.sso.dynatrace.com"})
	})

	h := newTestServiceUserHandler(t, mux)
	groups, err := h.GetGroups(context.Background(), "su1")
	if err != nil {
		t.Fatalf("GetGroups() error: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("GetGroups() returned %d groups, want 0", len(groups))
	}
}

func TestServiceUserHandler_Update_FillsRequiredName(t *testing.T) {
	h := newTestServiceUserHandler(t, serviceUserMux(t, nil))
	desc := "new description"
	result, err := h.Update(context.Background(), "su1", nil, &desc, nil)
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if result["name"] != "CI Bot" || result["description"] != "new description" {
		t.Errorf("Update() = %v, want name kept as CI Bot and description replaced", result)
	}
}

func TestServiceUserHandler_GroupMembership(t *testing.T) {
	tests := []struct {
		name       string
		change     func(h *ServiceUserHandler) error
		wantMethod string
		wantGroup  string
	}{
		{
			name:       "should add to a group with POST /users/{email}",
			change:     func(h *ServiceUserHandler) error { return h.AddToGroup(context.Background(), "su1", "g9") },
			wantMethod: http.MethodPost,
			wantGroup:  "g9",
		},
		{
			name:       "should remove from a group with DELETE /users/{email}/groups",
			change:     func(h *ServiceUserHandler) error { return h.RemoveFromGroup(context.Background(), "su1", "g9") },
			wantMethod: http.MethodDelete,
			wantGroup:  "g9",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method, group string
			mux := serviceUserMux(t, func(w http.ResponseWriter, r *http.Request) {
				method = r.Method
				if r.Method == http.MethodPost {
					var body []string
					_ = json.NewDecoder(r.Body).Decode(&body)
					if len(body) == 1 {
						group = body[0]
					}
				} else {
					group = r.URL.Query().Get("group-uuid")
				}
				w.WriteHeader(200)
			})

			h := newTestServiceUserHandler(t, mux)
			if err := tt.change(h); err != nil {
				t.Fatalf("membership change error: %v", err)
			}
			if method != tt.wantMethod || group != tt.wantGroup {
				t.Errorf("sent %s for group %q, want %s for %q", method, group, tt.wantMethod, tt.wantGroup)
			}
		})
	}
}
