package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// newTestBaseHandler creates a BaseHandler backed by a mock server.
func newTestBaseHandler(t *testing.T, mux *http.ServeMux) *BaseHandler {
	t.Helper()
	c := newTestClient(t, mux)
	return &BaseHandler{
		Client:    c,
		Name:      "widget",
		Path:      "/widgets",
		ListKey:   "items",
		IDField:   "uuid",
		NameField: "name",
	}
}

func TestBaseHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uuid": "w1", "name": "Alpha"},
				map[string]any{"uuid": "w2", "name": "Beta"},
			},
		})
	})

	h := newTestBaseHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(items))
	}
	if items[0]["uuid"] != "w1" {
		t.Errorf("items[0][uuid] = %v, want w1", items[0]["uuid"])
	}
}

func TestBaseHandler_List_ArrayResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{
			map[string]any{"uuid": "w1"},
		})
	})

	h := newTestBaseHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() returned %d items, want 1", len(items))
	}
}

func TestBaseHandler_List_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"message":"internal error"}`))
	})

	h := newTestBaseHandler(t, mux)
	_, err := h.List(context.Background(), nil)
	if err == nil {
		t.Fatal("List() expected error for 500, got nil")
	}
}

func TestBaseHandler_Get_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/w1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "w1", "name": "Alpha"})
	})

	h := newTestBaseHandler(t, mux)
	item, err := h.Get(context.Background(), "w1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["name"] != "Alpha" {
		t.Errorf("Get() name = %v, want Alpha", item["name"])
	}
}

func TestBaseHandler_Get_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestBaseHandler(t, mux)
	_, err := h.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("Get() expected error for 404, got nil")
	}
	if err.Error() != "widget not found" {
		t.Errorf("Get() error = %q, want 'widget not found'", err.Error())
	}
}

func TestBaseHandler_Create_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "new-1", "name": "Gamma"})
	})

	h := newTestBaseHandler(t, mux)
	result, err := h.Create(context.Background(), map[string]any{"name": "Gamma"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uuid"] != "new-1" {
		t.Errorf("Create() uuid = %v, want new-1", result["uuid"])
	}
}

func TestBaseHandler_Create_EmptyBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	h := newTestBaseHandler(t, mux)
	data := map[string]any{"name": "Gamma"}
	result, err := h.Create(context.Background(), data)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["name"] != "Gamma" {
		t.Errorf("Create() should return input data when body empty, got %v", result)
	}
}

func TestBaseHandler_Delete_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/w1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestBaseHandler(t, mux)
	err := h.Delete(context.Background(), "w1")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestBaseHandler_Delete_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestBaseHandler(t, mux)
	err := h.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("Delete() expected error for 404, got nil")
	}
}

func TestBaseHandler_Update_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/w1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(405)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "w1", "name": "Updated"})
	})

	h := newTestBaseHandler(t, mux)
	result, err := h.Update(context.Background(), "w1", map[string]any{"name": "Updated"})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if result["name"] != "Updated" {
		t.Errorf("Update() name = %v, want Updated", result["name"])
	}
}

func TestBaseHandler_GetByName_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uuid": "w1", "name": "Alpha"},
				map[string]any{"uuid": "w2", "name": "Beta"},
			},
		})
	})

	h := newTestBaseHandler(t, mux)
	item, err := h.GetByName(context.Background(), "beta")
	if err != nil {
		t.Fatalf("GetByName() error: %v", err)
	}
	if item == nil {
		t.Fatal("GetByName() returned nil, want item")
	}
	if item["uuid"] != "w2" {
		t.Errorf("GetByName() uuid = %v, want w2", item["uuid"])
	}
}

func TestBaseHandler_GetByName_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uuid": "w1", "name": "Alpha"},
			},
		})
	})

	h := newTestBaseHandler(t, mux)
	item, err := h.GetByName(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByName() error: %v", err)
	}
	if item != nil {
		t.Errorf("GetByName() should return nil for missing name, got %v", item)
	}
}

func TestBaseHandler_Exists_True(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/w1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "w1"})
	})

	h := newTestBaseHandler(t, mux)
	if !h.Exists(context.Background(), "w1") {
		t.Error("Exists() = false, want true")
	}
}

func TestBaseHandler_Exists_False(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestBaseHandler(t, mux)
	if h.Exists(context.Background(), "missing") {
		t.Error("Exists() = true, want false")
	}
}

func TestBaseHandler_Resolve_ByID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/w1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "w1", "name": "Alpha"})
	})

	h := newTestBaseHandler(t, mux)
	item, err := h.Resolve(context.Background(), "w1")
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if item["uuid"] != "w1" {
		t.Errorf("Resolve() uuid = %v, want w1", item["uuid"])
	}
}

func TestBaseHandler_Resolve_ByName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/Alpha", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uuid": "w1", "name": "Alpha"},
			},
		})
	})

	h := newTestBaseHandler(t, mux)
	item, err := h.Resolve(context.Background(), "Alpha")
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if item["uuid"] != "w1" {
		t.Errorf("Resolve() uuid = %v, want w1", item["uuid"])
	}
}

func TestBaseHandler_Resolve_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	})

	h := newTestBaseHandler(t, mux)
	_, err := h.Resolve(context.Background(), "missing")
	if err == nil {
		t.Fatal("Resolve() expected error, got nil")
	}
}

func TestBaseHandler_ResourceName(t *testing.T) {
	h := &BaseHandler{Name: "widget"}
	if h.ResourceName() != "widget" {
		t.Errorf("ResourceName() = %q, want widget", h.ResourceName())
	}
}

func TestBaseHandler_APIPath(t *testing.T) {
	h := &BaseHandler{Path: "/widgets"}
	if h.APIPath() != "/widgets" {
		t.Errorf("APIPath() = %q, want /widgets", h.APIPath())
	}
}

func TestGetOrResolve_ByID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/w1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "w1", "name": "Alpha"})
	})

	h := newTestBaseHandler(t, mux)
	item, err := GetOrResolve(context.Background(), h, "w1")
	if err != nil {
		t.Fatalf("GetOrResolve() error: %v", err)
	}
	if item["uuid"] != "w1" {
		t.Errorf("GetOrResolve() uuid = %v, want w1", item["uuid"])
	}
}

func TestGetOrResolve_FallbackToList_ByName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/Alpha", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uuid": "w1", "name": "Alpha"},
			},
		})
	})

	h := newTestBaseHandler(t, mux)
	item, err := GetOrResolve(context.Background(), h, "Alpha")
	if err != nil {
		t.Fatalf("GetOrResolve() error: %v", err)
	}
	if item == nil {
		t.Fatal("GetOrResolve() returned nil")
	}
	if item["uuid"] != "w1" {
		t.Errorf("GetOrResolve() uuid = %v, want w1", item["uuid"])
	}
}

func TestGetOrResolve_FallbackToList_ByUUID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/w1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"uuid": "w1", "name": "Alpha"},
			},
		})
	})

	h := newTestBaseHandler(t, mux)
	item, err := GetOrResolve(context.Background(), h, "w1")
	if err != nil {
		t.Fatalf("GetOrResolve() error: %v", err)
	}
	if item == nil {
		t.Fatal("GetOrResolve() returned nil")
	}
	if item["uuid"] != "w1" {
		t.Errorf("GetOrResolve() uuid = %v, want w1", item["uuid"])
	}
}

func TestGetOrResolve_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})
	mux.HandleFunc("/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	})

	h := newTestBaseHandler(t, mux)
	item, err := GetOrResolve(context.Background(), h, "missing")
	if err != nil {
		t.Fatalf("GetOrResolve() unexpected error: %v", err)
	}
	if item != nil {
		t.Errorf("GetOrResolve() should return nil for not found, got %v", item)
	}
}

func TestGetOrResolve_NonNotFoundError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/widgets/w1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"message":"access denied"}`))
	})

	h := newTestBaseHandler(t, mux)
	_, err := GetOrResolve(context.Background(), h, "w1")
	if err == nil {
		t.Fatal("GetOrResolve() expected error for 403, got nil")
	}
}
