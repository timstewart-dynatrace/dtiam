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
		json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uid": "su1", "name": "CI Bot"},
				map[string]any{"uid": "su2", "name": "Deploy Bot"},
			},
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
		json.NewEncoder(w).Encode(map[string]any{
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
		json.NewEncoder(w).Encode(map[string]any{
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
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
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

func TestServiceUserHandler_Create_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "NewBot" {
			t.Errorf("expected name NewBot, got %v", body["name"])
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"uid": "su-new", "name": "NewBot"})
	})

	h := newTestServiceUserHandler(t, mux)
	desc := "A new service user"
	result, err := h.Create(context.Background(), "NewBot", &desc, []string{"g1"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uid"] != "su-new" {
		t.Errorf("Create() uid = %v, want su-new", result["uid"])
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
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["description"]; ok {
			t.Error("should not include description when nil")
		}
		if _, ok := body["groups"]; ok {
			t.Error("should not include groups when nil")
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"uid": "su-new", "name": body["name"]})
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
		w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestServiceUserHandler(t, mux)
	err := h.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("Delete() expected error for 404, got nil")
	}
}

func TestServiceUserHandler_GetGroups_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users/su1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"uid":  "su1",
			"name": "CI Bot",
			"groups": []any{
				map[string]any{"uuid": "g1", "name": "Admins"},
				"g2",
			},
		})
	})

	h := newTestServiceUserHandler(t, mux)
	groups, err := h.GetGroups(context.Background(), "su1")
	if err != nil {
		t.Fatalf("GetGroups() error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("GetGroups() returned %d groups, want 2", len(groups))
	}
	// First group is a map
	if groups[0]["uuid"] != "g1" {
		t.Errorf("groups[0] uuid = %v, want g1", groups[0]["uuid"])
	}
	// Second group is a string UUID
	if groups[1]["uuid"] != "g2" {
		t.Errorf("groups[1] uuid = %v, want g2", groups[1]["uuid"])
	}
}

func TestServiceUserHandler_GetGroups_NoGroups(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users/su1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"uid":  "su1",
			"name": "CI Bot",
		})
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

func TestServiceUserHandler_Update_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users/su1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			// GET for other tests
			json.NewEncoder(w).Encode(map[string]any{"uid": "su1", "name": "CI Bot"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"uid": "su1", "name": "Updated Bot"})
	})

	h := newTestServiceUserHandler(t, mux)
	name := "Updated Bot"
	result, err := h.Update(context.Background(), "su1", &name, nil, nil)
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if result["name"] != "Updated Bot" {
		t.Errorf("Update() name = %v, want Updated Bot", result["name"])
	}
}
