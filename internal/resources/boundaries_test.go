package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func newTestBoundaryHandler(t *testing.T, mux *http.ServeMux) *BoundaryHandler {
	t.Helper()
	c := newTestClient(t, mux)
	return &BoundaryHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "boundary",
			Path:      "/repo/account/test-uuid/boundaries",
			ListKey:   "boundaries",
			IDField:   "uuid",
			NameField: "name",
		},
	}
}

func TestBoundaryHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/boundaries", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"boundaries": []any{
				map[string]any{"uuid": "b1", "name": "Production"},
				map[string]any{"uuid": "b2", "name": "Staging"},
			},
		})
	})

	h := newTestBoundaryHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(items))
	}
}

func TestBoundaryHandler_Get_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/boundaries/b1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"uuid":          "b1",
			"name":          "Production",
			"boundaryQuery": "environment:management-zone IN (\"Production\");",
		})
	})

	h := newTestBoundaryHandler(t, mux)
	item, err := h.Get(context.Background(), "b1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["name"] != "Production" {
		t.Errorf("Get() name = %v, want Production", item["name"])
	}
}

func TestBoundaryHandler_Get_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/boundaries/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestBoundaryHandler(t, mux)
	_, err := h.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("Get() expected error for 404, got nil")
	}
}

func TestBoundaryHandler_Create_WithManagementZones(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/boundaries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "ProdBoundary" {
			t.Errorf("expected name ProdBoundary, got %v", body["name"])
		}
		query, ok := body["boundaryQuery"].(string)
		if !ok || query == "" {
			t.Error("expected boundaryQuery to be set from management zones")
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"uuid": "b-new", "name": "ProdBoundary"})
	})

	h := newTestBoundaryHandler(t, mux)
	desc := "Production boundary"
	result, err := h.Create(context.Background(), "ProdBoundary", []string{"Production", "Staging"}, nil, &desc)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uuid"] != "b-new" {
		t.Errorf("Create() uuid = %v, want b-new", result["uuid"])
	}
}

func TestBoundaryHandler_Create_WithBoundaryQuery(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/boundaries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"uuid": "b-new"})
	})

	h := newTestBoundaryHandler(t, mux)
	query := `shared:app-id IN ("dynatrace.dashboards");`
	result, err := h.Create(context.Background(), "AppBoundary", nil, &query, nil)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uuid"] != "b-new" {
		t.Errorf("Create() uuid = %v, want b-new", result["uuid"])
	}
}

func TestBoundaryHandler_Create_MissingQuery(t *testing.T) {
	mux := http.NewServeMux()
	h := newTestBoundaryHandler(t, mux)
	_, err := h.Create(context.Background(), "NoBoundary", nil, nil, nil)
	if err == nil {
		t.Fatal("Create() expected error when no query or zones, got nil")
	}
}

func TestBoundaryHandler_Delete_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/boundaries/b1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestBoundaryHandler(t, mux)
	err := h.Delete(context.Background(), "b1")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestBoundaryHandler_Delete_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/boundaries/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"message":"not found"}`))
	})

	h := newTestBoundaryHandler(t, mux)
	err := h.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("Delete() expected error for 404, got nil")
	}
}

func TestBoundaryHandler_BuildZoneQuery(t *testing.T) {
	mux := http.NewServeMux()
	h := newTestBoundaryHandler(t, mux)

	query := h.buildZoneQuery([]string{"Production", "Staging"})
	expected := `environment:management-zone IN ("Production", "Staging");
storage:dt.security_context IN ("Production", "Staging");
settings:dt.security_context IN ("Production", "Staging");`
	if query != expected {
		t.Errorf("buildZoneQuery() =\n%s\nwant\n%s", query, expected)
	}
}

func TestBoundaryHandler_BuildZoneQuery_SingleZone(t *testing.T) {
	mux := http.NewServeMux()
	h := newTestBoundaryHandler(t, mux)

	query := h.buildZoneQuery([]string{"Production"})
	expected := `environment:management-zone IN ("Production");
storage:dt.security_context IN ("Production");
settings:dt.security_context IN ("Production");`
	if query != expected {
		t.Errorf("buildZoneQuery() =\n%s\nwant\n%s", query, expected)
	}
}
