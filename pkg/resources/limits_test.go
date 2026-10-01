package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// newTestLimitsHandler builds the handler through its real constructor.
//
// It used to hand-roll a BaseHandler with its own ListKey and IDField, which
// meant the tests exercised a configuration the production code never used --
// so a wrong ListKey in NewLimitsHandler could not fail here.
func newTestLimitsHandler(t *testing.T, mux *http.ServeMux) *LimitsHandler {
	t.Helper()
	return NewLimitsHandler(newTestClient(t, mux))
}

// limitsResponse mirrors the live shape of GET /iam/v1/accounts/{uuid}/limits:
// a page envelope keyed "results", with each entry identified by limitType and
// carrying currentValue/limitValue. The earlier fixture used items/name/current/
// max, none of which the API returns -- so these tests passed while the command
// rendered an empty table and check-capacity reported every limit as missing.
func limitsResponse() map[string]any {
	return map[string]any{
		"pageSize":   1000,
		"pageNumber": 1,
		"total":      3,
		"results": []any{
			map[string]any{"limitType": "REGULAR_USER", "currentValue": float64(50), "limitValue": float64(100)},
			map[string]any{"limitType": "MAX_GROUPS_PER_ACCOUNT", "currentValue": float64(95), "limitValue": float64(100)},
			map[string]any{"limitType": "MAX_POLICIES", "currentValue": float64(100), "limitValue": float64(100)},
		},
	}
}

func TestLimitsHandler_List_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(limitsResponse())
	})

	h := newTestLimitsHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("List() returned %d items, want 3", len(items))
	}
}

func TestLimitsHandler_List_LimitsKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"limits": []any{
				map[string]any{"name": "user-count", "current": float64(10), "max": float64(50)},
			},
		})
	})

	h := newTestLimitsHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() returned %d items, want 1", len(items))
	}
}

func TestLimitsHandler_List_ArrayResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{
			map[string]any{"name": "user-count"},
		})
	})

	h := newTestLimitsHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() returned %d items, want 1", len(items))
	}
}

func TestLimitsHandler_List_SingleItemResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		// Some responses carry one limit directly rather than a page envelope.
		_ = json.NewEncoder(w).Encode(map[string]any{"limitType": "REGULAR_USER", "currentValue": float64(10)})
	})

	h := newTestLimitsHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() returned %d items, want 1", len(items))
	}
}

func TestLimitsHandler_List_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"message":"server error"}`))
	})

	h := newTestLimitsHandler(t, mux)
	_, err := h.List(context.Background(), nil)
	if err == nil {
		t.Fatal("List() expected error for 500, got nil")
	}
}

func TestLimitsHandler_Get_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(limitsResponse())
	})

	h := newTestLimitsHandler(t, mux)
	item, err := h.Get(context.Background(), "REGULAR_USER")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if item["limitType"] != "REGULAR_USER" {
		t.Errorf("Get() limitType = %v, want REGULAR_USER", item["limitType"])
	}
}

func TestLimitsHandler_Get_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}, "total": 0})
	})

	h := newTestLimitsHandler(t, mux)
	_, err := h.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Get() expected error for missing limit, got nil")
	}
}

func TestLimitsHandler_GetSummary_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(limitsResponse())
	})

	h := newTestLimitsHandler(t, mux)
	summary, err := h.GetSummary(context.Background())
	if err != nil {
		t.Fatalf("GetSummary() error: %v", err)
	}
	if summary["total_limits"] != 3 {
		t.Errorf("GetSummary() total_limits = %v, want 3", summary["total_limits"])
	}
	if summary["limits_near_capacity"] != 1 {
		t.Errorf("GetSummary() limits_near_capacity = %v, want 1", summary["limits_near_capacity"])
	}
	if summary["limits_at_capacity"] != 1 {
		t.Errorf("GetSummary() limits_at_capacity = %v, want 1", summary["limits_at_capacity"])
	}
}

func TestLimitsHandler_CheckCapacity_HasCapacity(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(limitsResponse())
	})

	h := newTestLimitsHandler(t, mux)
	result, err := h.CheckCapacity(context.Background(), "REGULAR_USER", 10)
	if err != nil {
		t.Fatalf("CheckCapacity() error: %v", err)
	}
	if result["has_capacity"] != true {
		t.Errorf("CheckCapacity() has_capacity = %v, want true", result["has_capacity"])
	}
	if result["available"] != 50 {
		t.Errorf("CheckCapacity() available = %v, want 50", result["available"])
	}
}

func TestLimitsHandler_CheckCapacity_NoCapacity(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(limitsResponse())
	})

	h := newTestLimitsHandler(t, mux)
	result, err := h.CheckCapacity(context.Background(), "MAX_POLICIES", 1)
	if err != nil {
		t.Fatalf("CheckCapacity() error: %v", err)
	}
	if result["has_capacity"] != false {
		t.Errorf("CheckCapacity() has_capacity = %v, want false", result["has_capacity"])
	}
}

func TestLimitsHandler_CheckCapacity_LimitNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}, "total": 0})
	})

	h := newTestLimitsHandler(t, mux)
	result, err := h.CheckCapacity(context.Background(), "nonexistent", 1)
	if err != nil {
		t.Fatalf("CheckCapacity() error: %v", err)
	}
	if result["found"] != false {
		t.Errorf("CheckCapacity() found = %v, want false", result["found"])
	}
}
