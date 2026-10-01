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

func TestGroupHandler_Get_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid": "g1",
			"name": "Admins",
		})
	})

	h := newTestGroupHandler(t, mux)
	item, err := h.Get(context.Background(), "g1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["name"] != "Admins" {
		t.Errorf("Get() name = %v, want Admins", item["name"])
	}
}

func TestGroupHandler_Create_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "g-new", "name": "NewGroup"})
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

func TestGroupHandler_AddMember_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestGroupHandler(t, mux)
	err := h.AddMember(context.Background(), "g1", "alice@example.com")
	if err != nil {
		t.Fatalf("AddMember() error: %v", err)
	}
}

func TestGroupHandler_RemoveMember_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/users/u1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestGroupHandler(t, mux)
	err := h.RemoveMember(context.Background(), "g1", "u1")
	if err != nil {
		t.Fatalf("RemoveMember() error: %v", err)
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
