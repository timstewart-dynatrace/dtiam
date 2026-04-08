package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func newTestBindingHandler(t *testing.T, mux *http.ServeMux) *BindingHandler {
	t.Helper()
	c := newTestClient(t, mux)
	return &BindingHandler{
		BaseHandler: BaseHandler{
			Client:  c,
			Name:    "binding",
			Path:    "/repo/account/test-uuid/bindings",
			ListKey: "policyBindings",
			IDField: "policyUuid",
		},
		LevelType: "account",
		LevelID:   "test-uuid",
	}
}

func bindingsResponse() map[string]any {
	return map[string]any{
		"policyBindings": []any{
			map[string]any{
				"policyUuid": "p1",
				"groups":     []any{"g1", "g2"},
				"boundaries": []any{"b1"},
			},
			map[string]any{
				"policyUuid": "p2",
				"groups":     []any{"g1"},
				"boundaries": []any{},
			},
		},
	}
}

func TestBindingHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(bindingsResponse())
	})

	h := newTestBindingHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	// p1 has 2 groups, p2 has 1 group = 3 flattened bindings
	if len(items) != 3 {
		t.Fatalf("List() returned %d items, want 3", len(items))
	}

	// Verify flattened structure
	first := items[0]
	if first["policyUuid"] != "p1" {
		t.Errorf("first binding policyUuid = %v, want p1", first["policyUuid"])
	}
	if first["groupUuid"] != "g1" {
		t.Errorf("first binding groupUuid = %v, want g1", first["groupUuid"])
	}
	if first["levelType"] != "account" {
		t.Errorf("first binding levelType = %v, want account", first["levelType"])
	}
}

func TestBindingHandler_List_EmptyBindings(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"policyBindings": []any{},
		})
	})

	h := newTestBindingHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("List() returned %d items, want 0", len(items))
	}
}

func TestBindingHandler_List_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"message":"server error"}`))
	})

	h := newTestBindingHandler(t, mux)
	_, err := h.List(context.Background(), nil)
	if err == nil {
		t.Fatal("List() expected error for 500, got nil")
	}
}

func TestBindingHandler_Create_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestBindingHandler(t, mux)
	result, err := h.Create(context.Background(), "g1", "p1", []string{"b1"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["groupUuid"] != "g1" {
		t.Errorf("Create() groupUuid = %v, want g1", result["groupUuid"])
	}
	if result["policyUuid"] != "p1" {
		t.Errorf("Create() policyUuid = %v, want p1", result["policyUuid"])
	}
}

func TestBindingHandler_Create_NoBoundaries(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		bindings := body["policyBindings"].([]any)
		binding := bindings[0].(map[string]any)
		if _, ok := binding["boundaries"]; ok {
			t.Error("Create() should not include boundaries when empty")
		}
		w.WriteHeader(204)
	})

	h := newTestBindingHandler(t, mux)
	_, err := h.Create(context.Background(), "g1", "p1", nil)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
}

func TestBindingHandler_Delete_Success(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(bindingsResponse())
		case http.MethodPut:
			w.WriteHeader(200)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(405)
		}
	})

	h := newTestBindingHandler(t, mux)
	err := h.Delete(context.Background(), "g1", "p1")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestBindingHandler_Delete_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(bindingsResponse())
	})

	h := newTestBindingHandler(t, mux)
	err := h.Delete(context.Background(), "g-nonexistent", "p-nonexistent")
	if err == nil {
		t.Fatal("Delete() expected error for missing binding, got nil")
	}
}

func TestBindingHandler_GetForGroup_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/groups/g1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"policyBindings": []any{
				map[string]any{
					"policyUuid": "p1",
					"groups":     []any{"g1"},
				},
			},
		})
	})

	h := newTestBindingHandler(t, mux)
	items, err := h.GetForGroup(context.Background(), "g1")
	if err != nil {
		t.Fatalf("GetForGroup() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("GetForGroup() returned %d items, want 1", len(items))
	}
}

func TestBindingHandler_GetForGroup_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/groups/g-missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestBindingHandler(t, mux)
	items, err := h.GetForGroup(context.Background(), "g-missing")
	if err != nil {
		t.Fatalf("GetForGroup() error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("GetForGroup() returned %d items, want 0 for 404", len(items))
	}
}

func TestBindingHandler_GetForPolicy_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/p1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"policyUuid": "p1",
			"groups":     []any{"g1", "g2"},
		})
	})

	h := newTestBindingHandler(t, mux)
	result, err := h.GetForPolicy(context.Background(), "p1")
	if err != nil {
		t.Fatalf("GetForPolicy() error: %v", err)
	}
	if result["policyUuid"] != "p1" {
		t.Errorf("GetForPolicy() policyUuid = %v, want p1", result["policyUuid"])
	}
}

func TestBindingHandler_ListRaw_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(bindingsResponse())
	})

	h := newTestBindingHandler(t, mux)
	raw, err := h.ListRaw(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListRaw() error: %v", err)
	}
	bindings, ok := raw["policyBindings"].([]any)
	if !ok {
		t.Fatal("ListRaw() policyBindings not found")
	}
	if len(bindings) != 2 {
		t.Errorf("ListRaw() returned %d bindings, want 2", len(bindings))
	}
}
