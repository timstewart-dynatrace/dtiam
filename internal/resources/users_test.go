package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func newTestUserHandler(t *testing.T, mux *http.ServeMux) *UserHandler {
	t.Helper()
	c := newTestClient(t, mux)
	return NewUserHandler(c)
}

func TestUserHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uid": "u1", "email": "alice@example.com"},
				map[string]any{"uid": "u2", "email": "bob@example.com"},
			},
		})
	})

	h := newTestUserHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(items))
	}
}

func TestUserHandler_Get_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users/u1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uid":   "u1",
			"email": "alice@example.com",
		})
	})

	h := newTestUserHandler(t, mux)
	item, err := h.Get(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["email"] != "alice@example.com" {
		t.Errorf("Get() email = %v, want alice@example.com", item["email"])
	}
}

func TestUserHandler_GetByEmail_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uid": "u1", "email": "alice@example.com"},
				map[string]any{"uid": "u2", "email": "bob@example.com"},
			},
		})
	})

	h := newTestUserHandler(t, mux)
	item, err := h.GetByEmail(context.Background(), "Bob@Example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error: %v", err)
	}
	if item == nil {
		t.Fatal("GetByEmail() returned nil")
	}
	if item["uid"] != "u2" {
		t.Errorf("GetByEmail() uid = %v, want u2", item["uid"])
	}
}

func TestUserHandler_GetByEmail_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	})

	h := newTestUserHandler(t, mux)
	item, err := h.GetByEmail(context.Background(), "nobody@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error: %v", err)
	}
	if item != nil {
		t.Errorf("GetByEmail() should return nil for missing email, got %v", item)
	}
}

func TestUserHandler_GetByName_AliasForGetByEmail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uid": "u1", "email": "alice@example.com"},
			},
		})
	})

	h := newTestUserHandler(t, mux)
	item, err := h.GetByName(context.Background(), "alice@example.com")
	if err != nil {
		t.Fatalf("GetByName() error: %v", err)
	}
	if item == nil {
		t.Fatal("GetByName() returned nil")
	}
	if item["uid"] != "u1" {
		t.Errorf("GetByName() uid = %v, want u1", item["uid"])
	}
}

func TestUserHandler_Create_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["email"] != "new@example.com" {
			t.Errorf("expected email new@example.com, got %v", body["email"])
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{"uid": "u-new", "email": "new@example.com"})
	})

	h := newTestUserHandler(t, mux)
	firstName := "New"
	lastName := "User"
	result, err := h.Create(context.Background(), "new@example.com", &firstName, &lastName, []string{"g1"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uid"] != "u-new" {
		t.Errorf("Create() uid = %v, want u-new", result["uid"])
	}
}

func TestUserHandler_Create_MinimalFields(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["name"]; ok {
			t.Error("Create() should not send name when nil")
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{"uid": "u-new", "email": body["email"]})
	})

	h := newTestUserHandler(t, mux)
	result, err := h.Create(context.Background(), "minimal@example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uid"] != "u-new" {
		t.Errorf("Create() uid = %v, want u-new", result["uid"])
	}
}

func TestUserHandler_Delete_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users/u1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestUserHandler(t, mux)
	err := h.Delete(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestUserHandler_GetGroups_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users/u1/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uuid": "g1", "name": "Admins"},
			},
		})
	})

	h := newTestUserHandler(t, mux)
	groups, err := h.GetGroups(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetGroups() error: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("GetGroups() returned %d groups, want 1", len(groups))
	}
}

func TestUserHandler_GetGroups_FallbackToUserObject(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users/u1/groups", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})
	mux.HandleFunc("/users/u1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uid":   "u1",
			"email": "alice@example.com",
			"groups": []any{
				map[string]any{"uuid": "g1", "name": "Admins"},
			},
		})
	})

	h := newTestUserHandler(t, mux)
	groups, err := h.GetGroups(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetGroups() error: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("GetGroups() returned %d groups, want 1", len(groups))
	}
}

func TestUserHandler_ReplaceGroups_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users/alice@example.com/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestUserHandler(t, mux)
	err := h.ReplaceGroups(context.Background(), "alice@example.com", []string{"g1", "g2"})
	if err != nil {
		t.Fatalf("ReplaceGroups() error: %v", err)
	}
}

func TestUserHandler_RemoveFromGroups_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users/alice@example.com/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestUserHandler(t, mux)
	err := h.RemoveFromGroups(context.Background(), "alice@example.com", []string{"g1"})
	if err != nil {
		t.Fatalf("RemoveFromGroups() error: %v", err)
	}
}

func TestUserHandler_AddToGroups_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users/alice@example.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestUserHandler(t, mux)
	err := h.AddToGroups(context.Background(), "alice@example.com", []string{"g1", "g2"})
	if err != nil {
		t.Fatalf("AddToGroups() error: %v", err)
	}
}

func TestUserHandler_ListWithServiceUsers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("service-users") != "true" {
			t.Errorf("expected service-users=true query param")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uid": "u1", "email": "alice@example.com"},
			},
		})
	})

	h := newTestUserHandler(t, mux)
	items, err := h.ListWithServiceUsers(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListWithServiceUsers() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("ListWithServiceUsers() returned %d items, want 1", len(items))
	}
}
