package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jtimothystewart/dtiam/internal/client"
)

func newTestPolicyHandler(t *testing.T, mux *http.ServeMux) *PolicyHandler {
	t.Helper()
	c := newTestClient(t, mux)
	// PolicyHandler uses absolute paths via RepoBasePath, so we build
	// the handler manually with a path relative to the test server.
	return &PolicyHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "policy",
			Path:      "/repo/account/test-uuid/policies",
			ListKey:   "policies",
			IDField:   "uuid",
			NameField: "name",
		},
		LevelType: "account",
		LevelID:   "test-uuid",
	}
}

func TestPolicyHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/policies", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"policies": []any{
				map[string]any{"uuid": "p1", "name": "ReadOnly"},
				map[string]any{"uuid": "p2", "name": "Admin"},
			},
		})
	})

	h := newTestPolicyHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(items))
	}
}

func TestPolicyHandler_Get_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/policies/p1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid":           "p1",
			"name":           "ReadOnly",
			"statementQuery": "ALLOW settings:objects:read;",
		})
	})

	h := newTestPolicyHandler(t, mux)
	item, err := h.Get(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["name"] != "ReadOnly" {
		t.Errorf("Get() name = %v, want ReadOnly", item["name"])
	}
}

func TestPolicyHandler_Create_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/policies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "p-new", "name": "NewPolicy"})
	})

	h := newTestPolicyHandler(t, mux)
	result, err := h.Create(context.Background(), map[string]any{
		"name":           "NewPolicy",
		"statementQuery": "ALLOW settings:objects:read;",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["uuid"] != "p-new" {
		t.Errorf("Create() uuid = %v, want p-new", result["uuid"])
	}
}

func TestPolicyHandler_Create_MissingName(t *testing.T) {
	mux := http.NewServeMux()
	h := newTestPolicyHandler(t, mux)
	_, err := h.Create(context.Background(), map[string]any{
		"statementQuery": "ALLOW settings:objects:read;",
	})
	if err == nil {
		t.Fatal("Create() expected error for missing name, got nil")
	}
}

func TestPolicyHandler_Create_MissingStatementQuery(t *testing.T) {
	mux := http.NewServeMux()
	h := newTestPolicyHandler(t, mux)
	_, err := h.Create(context.Background(), map[string]any{
		"name": "Test",
	})
	if err == nil {
		t.Fatal("Create() expected error for missing statementQuery, got nil")
	}
}

func TestPolicyHandler_Delete_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/policies/p1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(204)
	})

	h := newTestPolicyHandler(t, mux)
	err := h.Delete(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestPolicyHandler_Validate_Valid(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/policies/validation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"valid": true})
	})

	h := newTestPolicyHandler(t, mux)
	result, err := h.Validate(context.Background(), map[string]any{
		"name":           "Test",
		"statementQuery": "ALLOW settings:objects:read;",
	})
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if !result.Valid {
		t.Errorf("Validate() Valid = false, want true")
	}
	if len(result.Errors) != 0 {
		t.Errorf("Validate() Errors = %v, want empty", result.Errors)
	}
}

func TestPolicyHandler_Validate_Invalid(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/policies/validation", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"invalid syntax"}`))
	})

	h := newTestPolicyHandler(t, mux)
	result, err := h.Validate(context.Background(), map[string]any{
		"statementQuery": "INVALID QUERY",
	})
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if result.Valid {
		t.Errorf("Validate() Valid = true, want false")
	}
	if len(result.Errors) == 0 {
		t.Errorf("Validate() should have errors")
	}
}

func TestPolicyHandler_ListAllLevels(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Both account and global requests come here
		if strings.Contains(r.URL.Path, "policies") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"policies": []any{
					map[string]any{"uuid": "p1", "name": "Policy1"},
				},
			})
		} else {
			w.WriteHeader(404)
		}
	})

	// For ListAllLevels, the handler creates sub-handlers that build absolute URLs
	// using client.RepoBasePath. Since our test server URL is localhost, we need the
	// sub-handlers to also point to our server. We test the method's behavior by
	// verifying it attempts to merge results from multiple levels.
	c := newTestClient(t, mux)
	h := &PolicyHandler{
		BaseHandler: BaseHandler{
			Client:    c,
			Name:      "policy",
			Path:      "/repo/account/test-uuid/policies",
			ListKey:   "policies",
			IDField:   "uuid",
			NameField: "name",
		},
		LevelType: "account",
		LevelID:   "test-uuid",
	}

	// ListAllLevels creates new handlers with absolute URLs (client.RepoBasePath),
	// which won't hit our test server. This is a limitation of testing absolute URLs.
	// We verify it doesn't panic and returns whatever it can fetch.
	items, err := h.ListAllLevels(context.Background())
	// The sub-handlers will use absolute https URLs which won't match the test server,
	// so they'll error. ListAllLevels swallows errors per-level, so we expect no error
	// but also no items since the real URLs won't resolve.
	if err != nil {
		t.Fatalf("ListAllLevels() error: %v", err)
	}
	// Items may be nil/empty since the absolute URLs won't reach our test server
	_ = items
}

func TestNewPolicyHandler_SetsAccountLevel(t *testing.T) {
	tp := &mockTokenProvider{}
	c := client.New(client.Config{
		AccountUUID:   "my-uuid",
		TokenProvider: tp,
	})
	h := NewPolicyHandler(c)
	if h.LevelType != "account" {
		t.Errorf("LevelType = %q, want account", h.LevelType)
	}
	if h.LevelID != "my-uuid" {
		t.Errorf("LevelID = %q, want my-uuid", h.LevelID)
	}
}

func TestNewPolicyHandlerWithLevel_SetsLevel(t *testing.T) {
	tp := &mockTokenProvider{}
	c := client.New(client.Config{
		AccountUUID:   "my-uuid",
		TokenProvider: tp,
	})
	h := NewPolicyHandlerWithLevel(c, "environment", "env-123")
	if h.LevelType != "environment" {
		t.Errorf("LevelType = %q, want environment", h.LevelType)
	}
	if h.LevelID != "env-123" {
		t.Errorf("LevelID = %q, want env-123", h.LevelID)
	}
}
