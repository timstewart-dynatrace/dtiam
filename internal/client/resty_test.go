package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockTokenProvider implements TokenProvider for testing.
type mockTokenProvider struct {
	token string
}

func (m *mockTokenProvider) GetHeaders() (http.Header, error) {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+m.token)
	return h, nil
}

func (m *mockTokenProvider) IsValid() bool { return true }
func (m *mockTokenProvider) Close() error  { return nil }

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	tp := &mockTokenProvider{token: "test-token"}

	// Build a minimal resty client for testing (no retry to keep tests fast)
	config := Config{
		AccountUUID:   "test-uuid",
		TokenProvider: tp,
		Timeout:       5 * time.Second,
		RetryConfig:   &RetryConfig{MaxRetries: 0},
	}

	c := New(config)
	c.baseURL = server.URL
	return c, server
}

func TestClient_Get(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong auth header: %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	defer srv.Close()

	body, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if !strings.Contains(string(body), "ok") {
		t.Errorf("Get() body = %q, want to contain 'ok'", string(body))
	}
}

func TestClient_Get_WithParams(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") != "test" {
			t.Errorf("expected query param name=test, got %s", r.URL.Query().Get("name"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"found": true}`))
	})
	defer srv.Close()

	body, err := c.Get(context.Background(), "/search", map[string]string{"name": "test"})
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if !strings.Contains(string(body), "true") {
		t.Errorf("Get() body = %q, want to contain 'true'", string(body))
	}
}

func TestClient_Post(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "new-group" {
			t.Errorf("expected name='new-group', got %q", body["name"])
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "abc-123"})
	})
	defer srv.Close()

	resp, err := c.Post(context.Background(), "/groups", map[string]string{"name": "new-group"})
	if err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	if !strings.Contains(string(resp), "abc-123") {
		t.Errorf("Post() response = %q, want to contain 'abc-123'", string(resp))
	}
}

func TestClient_Put(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"updated": true}`))
	})
	defer srv.Close()

	resp, err := c.Put(context.Background(), "/groups/abc", map[string]string{"name": "updated"})
	if err != nil {
		t.Fatalf("Put() error: %v", err)
	}
	if !strings.Contains(string(resp), "true") {
		t.Errorf("Put() response = %q, want 'true'", string(resp))
	}
}

func TestClient_Patch(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"patched": true}`))
	})
	defer srv.Close()

	resp, err := c.Patch(context.Background(), "/groups/abc", map[string]string{"desc": "new"})
	if err != nil {
		t.Fatalf("Patch() error: %v", err)
	}
	if !strings.Contains(string(resp), "true") {
		t.Errorf("Patch() response = %q, want 'true'", string(resp))
	}
}

func TestClient_Delete(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := c.Delete(context.Background(), "/groups/abc")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestClient_DeleteWithBody(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		var body []string
		json.NewDecoder(r.Body).Decode(&body)
		if len(body) != 2 {
			t.Errorf("expected 2 items in body, got %d", len(body))
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := c.DeleteWithBody(context.Background(), "/users/alice/groups", []string{"g1", "g2"})
	if err != nil {
		t.Fatalf("DeleteWithBody() error: %v", err)
	}
}

func TestClient_ErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantMsg    string
	}{
		{
			"404 with message",
			404,
			`{"message": "group not found"}`,
			"group not found",
		},
		{
			"403 with error field",
			403,
			`{"error": "access denied"}`,
			"access denied",
		},
		{
			"500 with raw body",
			500,
			`Internal Server Error`,
			"Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			})
			defer srv.Close()

			_, err := c.Get(context.Background(), "/test", nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected *APIError, got %T", err)
			}
			if apiErr.StatusCode != tt.status {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.status)
			}
			if !strings.Contains(apiErr.Error(), tt.wantMsg) {
				t.Errorf("Error() = %q, want to contain %q", apiErr.Error(), tt.wantMsg)
			}
		})
	}
}

func TestClient_GetJSON(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"name": "test-group"})
	})
	defer srv.Close()

	var result map[string]string
	err := c.GetJSON(context.Background(), "/groups/abc", nil, &result)
	if err != nil {
		t.Fatalf("GetJSON() error: %v", err)
	}
	if result["name"] != "test-group" {
		t.Errorf("GetJSON() name = %q, want 'test-group'", result["name"])
	}
}

func TestClient_PostJSON(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "new-123"})
	})
	defer srv.Close()

	var result map[string]string
	err := c.PostJSON(context.Background(), "/groups", map[string]string{"name": "g"}, &result)
	if err != nil {
		t.Fatalf("PostJSON() error: %v", err)
	}
	if result["uuid"] != "new-123" {
		t.Errorf("PostJSON() uuid = %q, want 'new-123'", result["uuid"])
	}
}

func TestClient_PostJSON_NilTarget(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := c.PostJSON(context.Background(), "/groups", map[string]string{"name": "g"}, nil)
	if err != nil {
		t.Fatalf("PostJSON(nil) error: %v", err)
	}
}

func TestClient_PutJSON(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"updated": "true"})
	})
	defer srv.Close()

	var result map[string]string
	err := c.PutJSON(context.Background(), "/groups/abc", map[string]string{"name": "updated"}, &result)
	if err != nil {
		t.Fatalf("PutJSON() error: %v", err)
	}
	if result["updated"] != "true" {
		t.Errorf("PutJSON() updated = %q, want 'true'", result["updated"])
	}
}

func TestClient_BuildURL(t *testing.T) {
	c := &Client{baseURL: "https://api.example.com/accounts/uuid"}

	tests := []struct {
		path string
		want string
	}{
		{"/groups", "https://api.example.com/accounts/uuid/groups"},
		{"groups", "https://api.example.com/accounts/uuid/groups"},
		{"https://other.api.com/v1/data", "https://other.api.com/v1/data"},
		{"http://localhost:8080/test", "http://localhost:8080/test"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := c.buildURL(tt.path)
			if got != tt.want {
				t.Errorf("buildURL(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestClient_AccountUUID(t *testing.T) {
	c := &Client{accountUUID: "test-uuid"}
	if c.AccountUUID() != "test-uuid" {
		t.Errorf("AccountUUID() = %q, want 'test-uuid'", c.AccountUUID())
	}
}

func TestClient_BaseURLMethod(t *testing.T) {
	c := &Client{baseURL: "https://api.example.com"}
	if c.BaseURL() != "https://api.example.com" {
		t.Errorf("BaseURL() = %q, want 'https://api.example.com'", c.BaseURL())
	}
}

func TestClient_Close(t *testing.T) {
	tp := &mockTokenProvider{token: "test"}
	c := &Client{tokenProvider: tp}
	if err := c.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := c.Get(ctx, "/slow", nil)
	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}
}

func TestNew_Defaults(t *testing.T) {
	tp := &mockTokenProvider{token: "test"}
	c := New(Config{
		AccountUUID:   "uuid-123",
		TokenProvider: tp,
	})

	if c.AccountUUID() != "uuid-123" {
		t.Errorf("AccountUUID = %q, want 'uuid-123'", c.AccountUUID())
	}
	if !strings.Contains(c.BaseURL(), "uuid-123") {
		t.Errorf("BaseURL = %q, should contain 'uuid-123'", c.BaseURL())
	}
}

func TestNew_CustomRetryConfig(t *testing.T) {
	tp := &mockTokenProvider{token: "test"}
	rc := &RetryConfig{
		MaxRetries:    5,
		RetryStatuses: []int{429},
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      5 * time.Second,
	}
	c := New(Config{
		AccountUUID:   "uuid-123",
		TokenProvider: tp,
		RetryConfig:   rc,
	})

	// Client should be created without panicking
	if c == nil {
		t.Fatal("New() returned nil with custom retry config")
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	rc := DefaultRetryConfig()
	if rc.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", rc.MaxRetries)
	}
	if len(rc.RetryStatuses) != 5 {
		t.Errorf("RetryStatuses len = %d, want 5", len(rc.RetryStatuses))
	}
}
