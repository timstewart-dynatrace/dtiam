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
		_ = json.NewEncoder(w).Encode(bindingsResponse())
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
		_, _ = w.Write([]byte(`{"message":"server error"}`))
	})

	h := newTestBindingHandler(t, mux)
	_, err := h.List(context.Background(), nil)
	if err == nil {
		t.Fatal("List() expected error for 500, got nil")
	}
}

func TestBindingHandler_Create_PostsToPolicyGroupPath(t *testing.T) {
	var gotBody map[string]any
	mux := http.NewServeMux()
	// The level-wide .../bindings collection has no POST; a binding is added
	// with POST .../bindings/{policy}/{group}.
	mux.HandleFunc("/repo/account/test-uuid/bindings/p1/g1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(204)
	})

	h := newTestBindingHandler(t, mux)
	result, err := h.Create(context.Background(), "g1", "p1", []string{"b1"}, map[string]string{"env": "prod"})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if result["groupUuid"] != "g1" || result["policyUuid"] != "p1" {
		t.Errorf("Create() = %v, want groupUuid g1 and policyUuid p1", result)
	}
	if _, ok := gotBody["policyBindings"]; ok {
		t.Error("Create() sent the policyBindings envelope; the per-pair endpoint takes a bare binding")
	}
	if bs, _ := gotBody["boundaries"].([]any); len(bs) != 1 || bs[0] != "b1" {
		t.Errorf("Create() boundaries = %v, want [b1]", gotBody["boundaries"])
	}
	if params, _ := gotBody["parameters"].(map[string]any); params["env"] != "prod" {
		t.Errorf("Create() parameters = %v, want env=prod", gotBody["parameters"])
	}
}

func TestBindingHandler_Create_NoBoundaries(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/p1/g1", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["boundaries"]; ok {
			t.Error("Create() should not include boundaries when empty")
		}
		w.WriteHeader(204)
	})

	h := newTestBindingHandler(t, mux)
	if _, err := h.Create(context.Background(), "g1", "p1", nil, nil); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
}

func TestBindingHandler_Delete_DeletesOnlyThePair(t *testing.T) {
	var method string
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/p1/g1", func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.WriteHeader(204)
	})
	// The level-wide collection must not be touched: rewriting it with PUT was
	// the old behavior, and DELETE on it removes every binding at the level.
	mux.HandleFunc("/repo/account/test-uuid/bindings", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Delete() touched the level-wide bindings collection with %s", r.Method)
		w.WriteHeader(500)
	})

	h := newTestBindingHandler(t, mux)
	if err := h.Delete(context.Background(), "g1", "p1"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if method != http.MethodDelete {
		t.Errorf("Delete() used %s, want DELETE", method)
	}
}

func TestBindingHandler_Delete_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/p-nonexistent/g-nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"error":true,"message":"Cannot get requested resource.","payload":null}`))
	})

	h := newTestBindingHandler(t, mux)
	err := h.Delete(context.Background(), "g-nonexistent", "p-nonexistent")
	if err == nil || err.Error() != "binding not found" {
		t.Fatalf("Delete() error = %v, want \"binding not found\"", err)
	}
}

func TestBindingHandler_GetForGroup(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		wantPolicies   []string
		wantBoundaries [][]string
	}{
		{
			name: "should read bindingsDetails with boundaries when details are returned",
			// Live shape of GET .../bindings/groups/{uuid}?details=true.
			response: `{"policyUuids":["p1","p2"],"bindingsDetails":[
				{"policyUuid":"p1","groups":["g1"],"levelType":"account","levelId":"test-uuid","boundaries":["b1"]},
				{"policyUuid":"p2","groups":["g1"],"levelType":"account","levelId":"test-uuid"}]}`,
			wantPolicies:   []string{"p1", "p2"},
			wantBoundaries: [][]string{{"b1"}, {}},
		},
		{
			name:           "should fall back to policyUuids when details are absent",
			response:       `{"policyUuids":["p1","p2"]}`,
			wantPolicies:   []string{"p1", "p2"},
			wantBoundaries: [][]string{{}, {}},
		},
		{
			name:         "should return nothing for a group with no bindings",
			response:     `{"policyUuids":[]}`,
			wantPolicies: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/repo/account/test-uuid/bindings/groups/g1", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("details") != "true" {
					t.Error("GetForGroup() did not request details=true")
				}
				_, _ = w.Write([]byte(tt.response))
			})

			h := newTestBindingHandler(t, mux)
			items, err := h.GetForGroup(context.Background(), "g1")
			if err != nil {
				t.Fatalf("GetForGroup() error: %v", err)
			}
			if len(items) != len(tt.wantPolicies) {
				t.Fatalf("GetForGroup() returned %d items, want %d", len(items), len(tt.wantPolicies))
			}
			for i, item := range items {
				if item["policyUuid"] != tt.wantPolicies[i] || item["groupUuid"] != "g1" {
					t.Errorf("item %d = %v, want policy %s for group g1", i, item, tt.wantPolicies[i])
				}
				got, _ := item["boundaries"].([]string)
				if len(got) != len(tt.wantBoundaries[i]) {
					t.Errorf("item %d boundaries = %v, want %v", i, got, tt.wantBoundaries[i])
				}
			}
		})
	}
}

func TestBindingHandler_GetForGroup_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/groups/g-missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
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

// pairBindingResponse is the live shape of GET .../bindings/{policy}/{group}:
// the binding is wrapped in a policyBindings envelope.
func pairBindingResponse(boundaries ...string) map[string]any {
	bs := make([]any, len(boundaries))
	for i, b := range boundaries {
		bs[i] = b
	}
	return map[string]any{
		"levelType": "account",
		"levelId":   "test-uuid",
		"policyBindings": []any{map[string]any{
			"policyUuid": "p1",
			"groups":     []any{"g1"},
			"boundaries": bs,
			"parameters": map[string]any{"env": "prod"},
		}},
	}
}

func TestBindingHandler_BoundaryChanges(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		change   func(h *BindingHandler) error
		want     []string
	}{
		{
			name:     "should keep existing boundaries when attaching one",
			existing: []string{"b1"},
			change: func(h *BindingHandler) error {
				return h.AddBoundary(context.Background(), "g1", "p1", "b2")
			},
			want: []string{"b1", "b2"},
		},
		{
			name:     "should not duplicate a boundary that is already attached",
			existing: []string{"b1"},
			change: func(h *BindingHandler) error {
				return h.AddBoundary(context.Background(), "g1", "p1", "b1")
			},
			want: []string{"b1"},
		},
		{
			name:     "should remove only the named boundary when detaching",
			existing: []string{"b1", "b2"},
			change: func(h *BindingHandler) error {
				return h.RemoveBoundary(context.Background(), "g1", "p1", "b1")
			},
			want: []string{"b2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var put map[string]any
			mux := http.NewServeMux()
			mux.HandleFunc("/repo/account/test-uuid/bindings/p1/g1", func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					_ = json.NewEncoder(w).Encode(pairBindingResponse(tt.existing...))
				case http.MethodPut:
					_ = json.NewDecoder(r.Body).Decode(&put)
					w.WriteHeader(204)
				default:
					w.WriteHeader(405)
				}
			})

			h := newTestBindingHandler(t, mux)
			if err := tt.change(h); err != nil {
				t.Fatalf("boundary change error: %v", err)
			}

			got, _ := put["boundaries"].([]any)
			if len(got) != len(tt.want) {
				t.Fatalf("PUT boundaries = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("PUT boundaries = %v, want %v", got, tt.want)
				}
			}
			if params, _ := put["parameters"].(map[string]any); params["env"] != "prod" {
				t.Errorf("PUT dropped the binding parameters: %v", put["parameters"])
			}
			if _, ok := put["policyBindings"]; ok {
				t.Error("PUT echoed the GET envelope; the endpoint takes a bare binding")
			}
		})
	}
}

func TestBindingHandler_BoundaryChange_RefusesParameterizedDuplicates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/p1/g1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected %s with more than one binding for the pair", r.Method)
		}
		resp := pairBindingResponse("b1")
		bindings := resp["policyBindings"].([]any)
		resp["policyBindings"] = append(bindings, bindings[0])
		_ = json.NewEncoder(w).Encode(resp)
	})

	h := newTestBindingHandler(t, mux)
	if err := h.AddBoundary(context.Background(), "g1", "p1", "b2"); err == nil {
		t.Fatal("AddBoundary() expected an error for multiple bindings of one pair, got nil")
	}
}

func TestBindingHandler_UpdateGroupBindings_SendsPolicyUUIDs(t *testing.T) {
	var body map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/groups/g1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(204)
	})

	h := newTestBindingHandler(t, mux)
	if err := h.UpdateGroupBindings(context.Background(), "g1", []string{"p1", "p2"}); err != nil {
		t.Fatalf("UpdateGroupBindings() error: %v", err)
	}
	if ids, _ := body["policyUuids"].([]any); len(ids) != 2 {
		t.Errorf("UpdateGroupBindings() body = %v, want policyUuids [p1 p2]", body)
	}
}

func TestBindingHandler_GetForPolicy_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/account/test-uuid/bindings/p1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
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
		_ = json.NewEncoder(w).Encode(bindingsResponse())
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
