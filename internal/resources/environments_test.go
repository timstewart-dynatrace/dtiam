package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func newTestEnvironmentHandler(t *testing.T, mux *http.ServeMux) *EnvironmentHandler {
	t.Helper()
	c := newTestClient(t, mux)
	return &EnvironmentHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "environment",
			Path:      "/environments",
			ListKey:   "tenants",
			IDField:   "id",
			NameField: "name",
		},
	}
}

func TestEnvironmentHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"tenants": []any{
				map[string]any{"id": "env1", "name": "Production"},
				map[string]any{"id": "env2", "name": "Staging"},
			},
		})
	})

	h := newTestEnvironmentHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(items))
	}
}

func TestEnvironmentHandler_List_EnvironmentsKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"environments": []any{
				map[string]any{"id": "env1", "name": "Production"},
			},
		})
	})

	h := newTestEnvironmentHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() returned %d items, want 1", len(items))
	}
}

func TestEnvironmentHandler_List_ArrayResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{
			map[string]any{"id": "env1", "name": "Production"},
		})
	})

	h := newTestEnvironmentHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() returned %d items, want 1", len(items))
	}
}

func TestEnvironmentHandler_List_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"message":"server error"}`))
	})

	h := newTestEnvironmentHandler(t, mux)
	_, err := h.List(context.Background(), nil)
	if err == nil {
		t.Fatal("List() expected error for 500, got nil")
	}
}

func TestEnvironmentHandler_Get_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/env1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "env1",
			"name": "Production",
		})
	})

	h := newTestEnvironmentHandler(t, mux)
	item, err := h.Get(context.Background(), "env1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["name"] != "Production" {
		t.Errorf("Get() name = %v, want Production", item["name"])
	}
}

func TestEnvironmentHandler_Get_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestEnvironmentHandler(t, mux)
	_, err := h.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("Get() expected error for 404, got nil")
	}
}

func TestEnvironmentHandler_GetByName_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"tenants": []any{
				map[string]any{"id": "env1", "name": "Production"},
				map[string]any{"id": "env2", "name": "Staging"},
			},
		})
	})

	h := newTestEnvironmentHandler(t, mux)
	item, err := h.GetByName(context.Background(), "staging")
	if err != nil {
		t.Fatalf("GetByName() error: %v", err)
	}
	if item == nil {
		t.Fatal("GetByName() returned nil")
	}
	if item["id"] != "env2" {
		t.Errorf("GetByName() id = %v, want env2", item["id"])
	}
}

func TestEnvironmentHandler_GetByName_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"tenants": []any{
				map[string]any{"id": "env1", "name": "Production"},
			},
		})
	})

	h := newTestEnvironmentHandler(t, mux)
	item, err := h.GetByName(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByName() error: %v", err)
	}
	if item != nil {
		t.Errorf("GetByName() should return nil for missing name, got %v", item)
	}
}

func TestEnvironmentHandler_List_Empty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"tenants": []any{},
		})
	})

	h := newTestEnvironmentHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("List() returned %d items, want 0", len(items))
	}
}
