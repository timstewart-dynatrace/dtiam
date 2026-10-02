package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

func TestNewTokenHandler(t *testing.T) {
	h := NewTokenHandler(nil)
	if h.Name != "platform-token" {
		t.Errorf("Name = %q, want 'platform-token'", h.Name)
	}
	if h.Path != "/platform-tokens" {
		t.Errorf("Path = %q, want '/platform-tokens'", h.Path)
	}
	// The response field is tokenId; "id" does not exist, so get/delete by ID
	// could never resolve a token.
	if h.IDField != "tokenId" {
		t.Errorf("IDField = %q, want 'tokenId'", h.IDField)
	}
	// The platform tokens API returns {pageSize, pageNumber, total, results};
	// reading "items" here yielded an empty list against the live API.
	if h.ListKey != "results" {
		t.Errorf("ListKey = %q, want 'results'", h.ListKey)
	}
	if h.Pagination == nil {
		t.Fatal("Pagination is nil; the platform tokens API is paginated")
	}
	if h.Pagination.Style != client.PaginationPageNumber {
		t.Errorf("Pagination.Style = %v, want PaginationPageNumber", h.Pagination.Style)
	}
	if h.Pagination.PageSizeParam != "size" {
		t.Errorf("PageSizeParam = %q, want 'size'", h.Pagination.PageSizeParam)
	}
}

func TestTokenHandler_ResourceName(t *testing.T) {
	h := NewTokenHandler(nil)
	if h.ResourceName() != "platform-token" {
		t.Errorf("ResourceName() = %q, want 'platform-token'", h.ResourceName())
	}
}

func TestTokenHandler_APIPath(t *testing.T) {
	h := NewTokenHandler(nil)
	if h.APIPath() != "/platform-tokens" {
		t.Errorf("APIPath() = %q, want '/platform-tokens'", h.APIPath())
	}
}

func TestTokenHandler_SetStatusAndExpiration(t *testing.T) {
	tests := []struct {
		name     string
		call     func(h *TokenHandler) error
		wantPath string
		wantBody map[string]string
		wantErr  bool
	}{
		{
			name:     "should PUT INACTIVE to the status endpoint",
			call:     func(h *TokenHandler) error { return h.SetStatus(context.Background(), "dt0s16.A", TokenStatusInactive) },
			wantPath: "/platform-tokens/dt0s16.A/status",
			wantBody: map[string]string{"status": "INACTIVE"},
		},
		{
			name: "should PUT the date to the expiration endpoint",
			call: func(h *TokenHandler) error {
				return h.SetExpiration(context.Background(), "dt0s16.A", "2027-01-01T00:00:00Z")
			},
			wantPath: "/platform-tokens/dt0s16.A/expiration-date",
			wantBody: map[string]string{"expirationDate": "2027-01-01T00:00:00Z"},
		},
		{
			name:    "should reject an unknown status without calling the API",
			call:    func(h *TokenHandler) error { return h.SetStatus(context.Background(), "dt0s16.A", "PAUSED") },
			wantErr: true,
		},
		{
			name:    "should reject a non-RFC3339 date without calling the API",
			call:    func(h *TokenHandler) error { return h.SetExpiration(context.Background(), "dt0s16.A", "2027-01-01") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotMethod string
			var gotBody map[string]string
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				gotPath, gotMethod = r.URL.Path, r.Method
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				w.WriteHeader(204)
			})
			c := newTestClient(t, mux)
			h := NewTokenHandler(c)
			h.Path = "/platform-tokens"

			err := tt.call(h)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				if gotPath != "" {
					t.Errorf("invalid input reached the API at %s", gotPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotMethod != http.MethodPut || gotPath != tt.wantPath {
				t.Errorf("sent %s %s, want PUT %s", gotMethod, gotPath, tt.wantPath)
			}
			for k, v := range tt.wantBody {
				if gotBody[k] != v {
					t.Errorf("body[%s] = %q, want %q", k, gotBody[k], v)
				}
			}
		})
	}
}

func TestTokenHandler_CreateSendsSpecFields(t *testing.T) {
	var body map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/platform-tokens", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "ci", "tokenId": "dt0s16.A", "token": "secret"})
	})
	h := NewTokenHandler(newTestClient(t, mux))
	h.Path = "/platform-tokens"

	got, err := h.Create(context.Background(), PlatformTokenRequest{
		Name: "ci", Scopes: []string{"a:b:read"}, Resources: []string{AccountResource("acct")},
		ExpirationDate: "2027-01-01T00:00:00Z", UserUUID: "u1",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if got["tokenId"] != "dt0s16.A" {
		t.Errorf("Create() = %v", got)
	}
	// The API requires all six fields; the pre-3.2.0 {scopes, expiresIn} body
	// was rejected with "resource must not be empty, scope list cannot be empty".
	for _, k := range []string{"name", "scope", "resource", "tags", "expirationDate", "userUuid"} {
		if _, ok := body[k]; !ok {
			t.Errorf("body is missing %q: %v", k, body)
		}
	}
	for _, k := range []string{"scopes", "expiresIn"} {
		if _, ok := body[k]; ok {
			t.Errorf("body still sends the old field %q", k)
		}
	}
	if r, _ := body["resource"].([]any); len(r) != 1 || r[0] != "urn:dtaccount:acct" {
		t.Errorf("resource = %v", body["resource"])
	}
}

func TestTokenHandler_CreateValidatesLocally(t *testing.T) {
	h := NewTokenHandler(newTestClient(t, http.NewServeMux()))
	valid := PlatformTokenRequest{Name: "n", Scopes: []string{"s"}, Resources: []string{"r"},
		ExpirationDate: "2027-01-01T00:00:00Z", UserUUID: "u"}

	tests := []struct {
		name   string
		mutate func(*PlatformTokenRequest)
	}{
		{"should require a name", func(r *PlatformTokenRequest) { r.Name = "" }},
		{"should require a scope", func(r *PlatformTokenRequest) { r.Scopes = nil }},
		{"should require a resource", func(r *PlatformTokenRequest) { r.Resources = nil }},
		{"should require an owner", func(r *PlatformTokenRequest) { r.UserUUID = "" }},
		{"should require an RFC 3339 expiration", func(r *PlatformTokenRequest) { r.ExpirationDate = "30d" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			tt.mutate(&req)
			if _, err := h.Create(context.Background(), req); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestParseTokenLifetime(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "30d", want: "2026-11-01T12:00:00Z"},
		{in: "2w", want: "2026-10-16T12:00:00Z"},
		{in: "1y", want: "2027-10-02T12:00:00Z"},
		{in: "12h", want: "2026-10-03T00:00:00Z"},
		{in: "90m", want: "2026-10-02T13:30:00Z"},
		{in: "soon", wantErr: true},
		{in: "0d", wantErr: true},
		{in: "-5d", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseTokenLifetime(tt.in, now)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseTokenLifetime(%q) = %q, want an error", tt.in, got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("ParseTokenLifetime(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
			}
		})
	}
}
