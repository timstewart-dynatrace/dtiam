package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/jtimothystewart/dtiam/pkg/client"
)

// recordingMux captures the query parameters of every request so tests can
// assert which paging parameters were actually sent.
type recordingMux struct {
	mu      sync.Mutex
	queries []map[string]string
}

func (r *recordingMux) record(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	q := map[string]string{}
	for k, v := range req.URL.Query() {
		q[k] = v[0]
	}
	r.queries = append(r.queries, q)
}

func (r *recordingMux) calls() []map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.queries
}

func TestList_PageKeyStyle_FollowsCursorToCompletion(t *testing.T) {
	rec := &recordingMux{}
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		switch r.URL.Query().Get("page-key") {
		case "":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results":     []any{map[string]any{"uid": "su1"}, map[string]any{"uid": "su2"}},
				"nextPageKey": "KEY2",
				"totalCount":  5,
			})
		case "KEY2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results":     []any{map[string]any{"uid": "su3"}, map[string]any{"uid": "su4"}},
				"nextPageKey": "KEY3",
				"totalCount":  5,
			})
		case "KEY3":
			// Final page: no nextPageKey.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results":    []any{map[string]any{"uid": "su5"}},
				"totalCount": 5,
			})
		default:
			t.Errorf("unexpected page-key %q", r.URL.Query().Get("page-key"))
		}
	})

	h := newTestServiceUserHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("List() returned %d items, want all 5 across 3 pages", len(items))
	}
	if got := len(rec.calls()); got != 3 {
		t.Errorf("made %d requests, want 3", got)
	}
	if first := rec.calls()[0]; first["page-size"] == "" {
		t.Error("first request did not send page-size")
	}
}

func TestList_PageKeyStyle_StopsWhenCursorRepeats(t *testing.T) {
	// A server that keeps echoing the same cursor must not loop forever.
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results":     []any{map[string]any{"uid": "su1"}},
			"nextPageKey": "SAME",
		})
	})

	h := newTestServiceUserHandler(t, mux)
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if calls > 3 {
		t.Fatalf("made %d requests; a repeated cursor must terminate immediately", calls)
	}
	if len(items) == 0 {
		t.Error("expected the first page to be returned")
	}
}

func TestList_PageNumberStyle_StopsAtReportedTotal(t *testing.T) {
	rec := &recordingMux{}
	mux := http.NewServeMux()
	mux.HandleFunc("/platform-tokens", func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		page := r.URL.Query().Get("page")
		switch page {
		case "1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results":    []any{map[string]any{"id": "t1"}, map[string]any{"id": "t2"}},
				"total":      3,
				"pageNumber": 1,
			})
		case "2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results":    []any{map[string]any{"id": "t3"}},
				"total":      3,
				"pageNumber": 2,
			})
		default:
			t.Errorf("requested page %q after total was reached", page)
		}
	})

	h := NewTokenHandler(newTestClient(t, mux))
	// A small page size makes the short-final-page path observable.
	h.Pagination.PageSize = 2

	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("List() returned %d items, want 3", len(items))
	}
	if got := len(rec.calls()); got != 2 {
		t.Errorf("made %d requests, want 2", got)
	}
	if rec.calls()[0]["size"] != "2" {
		t.Errorf("size = %q, want '2'", rec.calls()[0]["size"])
	}
}

func TestList_PageNumberStyle_SinglePageDoesNotRefetch(t *testing.T) {
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/platform-tokens", func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []any{map[string]any{"id": "t1"}},
			"total":   1,
		})
	})

	h := NewTokenHandler(newTestClient(t, mux))
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if calls != 1 {
		t.Errorf("made %d requests, want 1", calls)
	}
}

func TestList_EmptyFirstPageTerminates(t *testing.T) {
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/platform-tokens", func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}, "total": 0})
	})

	h := NewTokenHandler(newTestClient(t, mux))
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0", len(items))
	}
	if calls != 1 {
		t.Errorf("made %d requests, want 1", calls)
	}
}

func TestList_PaginationPreservesCallerFilters(t *testing.T) {
	rec := &recordingMux{}
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		if r.URL.Query().Get("page-key") == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results":     []any{map[string]any{"uid": "su1"}},
				"nextPageKey": "K2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{map[string]any{"uid": "su2"}}})
	})

	h := newTestServiceUserHandler(t, mux)
	if _, err := h.List(context.Background(), map[string]string{"service-users": "true"}); err != nil {
		t.Fatalf("List() error: %v", err)
	}

	// Filters are not embedded in the cursor, so they must be resent every page.
	for i, q := range rec.calls() {
		if q["service-users"] != "true" {
			t.Errorf("request %d dropped the caller filter: %v", i, q)
		}
	}
}

func TestList_UnpaginatedHandlerSendsNoPagingParams(t *testing.T) {
	rec := &recordingMux{}
	mux := http.NewServeMux()
	mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count": 1,
			"items": []any{map[string]any{"uuid": "g1", "name": "Admins"}},
		})
	})

	h := NewGroupHandler(newTestClient(t, mux))
	if h.Pagination != nil {
		t.Fatal("the group API is not paginated; Pagination must stay nil")
	}
	items, err := h.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	for _, forbidden := range []string{"page", "size", "page-size", "page-key"} {
		if v, ok := rec.calls()[0][forbidden]; ok {
			t.Errorf("sent %s=%q to an unpaginated endpoint", forbidden, v)
		}
	}
}

func TestList_RespectsMaxPageRequests(t *testing.T) {
	// A server that always issues a fresh cursor must still be bounded.
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/service-users", func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results":     []any{map[string]any{"uid": fmt.Sprintf("su%d", calls)}},
			"nextPageKey": fmt.Sprintf("KEY%d", calls),
		})
	})

	h := newTestServiceUserHandler(t, mux)
	if _, err := h.List(context.Background(), nil); err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if calls != client.MaxPageRequests {
		t.Errorf("made %d requests, want the MaxPageRequests cap of %d", calls, client.MaxPageRequests)
	}
}
