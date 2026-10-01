package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestNewOrgLevelHandler_ExpandsBareEnvironmentID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "should expand a bare ID to the apps host",
			// This API is served from .apps, not .live.
			input: "abc12345",
			want:  "https://abc12345.apps.dynatrace.com",
		},
		{
			name:  "should keep a full https URL",
			input: "https://abc12345.apps.dynatrace.com",
			want:  "https://abc12345.apps.dynatrace.com",
		},
		{
			name:  "should strip a trailing slash",
			input: "https://abc12345.apps.dynatrace.com/",
			want:  "https://abc12345.apps.dynatrace.com",
		},
		{
			name:  "should leave an empty value empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewOrgLevelHandler(nil, tt.input)
			if h.EnvironmentURL != tt.want {
				t.Errorf("EnvironmentURL = %q, want %q", h.EnvironmentURL, tt.want)
			}
		})
	}
}

func TestValidateLevelType(t *testing.T) {
	for _, valid := range []string{LevelAccount, LevelEnvironment} {
		if err := ValidateLevelType(valid); err != nil {
			t.Errorf("ValidateLevelType(%q) returned %v, want nil", valid, err)
		}
	}
	// "global" is valid for policy levels but not for organizational levels.
	for _, invalid := range []string{"global", "tenant", "", "Account"} {
		if err := ValidateLevelType(invalid); err == nil {
			t.Errorf("ValidateLevelType(%q) returned nil, want an error", invalid)
		}
	}
}

func TestOrgLevelHandler_RequiresEnvironmentURL(t *testing.T) {
	h := NewOrgLevelHandler(newTestClient(t, http.NewServeMux()), "")
	_, err := h.ListGroups(context.Background(), LevelEnvironment, "abc12345", "")
	if err == nil {
		t.Fatal("expected an error when no environment URL is configured")
	}
	if !strings.Contains(err.Error(), "DTIAM_ENVIRONMENT_URL") {
		t.Errorf("error = %q, want it to name the env var that fixes it", err)
	}
}

func TestOrgLevelHandler_ListUsersRequiresSearchTermOrUUID(t *testing.T) {
	// The API will not enumerate all users; it rejects a request with neither
	// filter, so fail locally with a clearer message.
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { called = true })

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewOrgLevelHandler(c, baseURL)

	_, err := h.ListUsers(context.Background(), LevelEnvironment, "abc12345", "", "")
	if err == nil {
		t.Fatal("expected an error when neither a search term nor a UUID is given")
	}
	if called {
		t.Error("no request should be made without a filter")
	}
}

func TestOrgLevelHandler_ListUsersSendsSearchParams(t *testing.T) {
	var query url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/platform/iam/v1/organizational-levels/environment/abc12345/users",
		func(w http.ResponseWriter, r *http.Request) {
			query = r.URL.Query()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results":    []any{map[string]any{"uuid": "u1", "email": "alice@example.com"}},
				"totalCount": 1,
			})
		})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewOrgLevelHandler(c, baseURL)

	users, err := h.ListUsers(context.Background(), LevelEnvironment, "abc12345", "alice", "")
	if err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("got %d users, want 1", len(users))
	}
	if query.Get("partialString") != "alice" {
		t.Errorf("partialString = %q, want 'alice'", query.Get("partialString"))
	}
	// This API uses pageSize, not size or page-size.
	if query.Get("pageSize") == "" {
		t.Error("pageSize was not sent")
	}
}

func TestOrgLevelHandler_ListUsersAcceptsUUIDOnly(t *testing.T) {
	var query url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/platform/iam/v1/organizational-levels/account/acct-1/users",
		func(w http.ResponseWriter, r *http.Request) {
			query = r.URL.Query()
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}, "totalCount": 0})
		})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewOrgLevelHandler(c, baseURL)

	if _, err := h.ListUsers(context.Background(), LevelAccount, "acct-1", "", "uuid-1"); err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}
	if query.Get("uuid") != "uuid-1" {
		t.Errorf("uuid = %q, want 'uuid-1'", query.Get("uuid"))
	}
}

func TestOrgLevelHandler_ListGroupsReadsResultsKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/platform/iam/v1/organizational-levels/environment/abc12345/groups",
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []any{
					map[string]any{"uuid": "g1", "name": "Admins"},
					map[string]any{"uuid": "g2", "name": "Devs"},
				},
				"totalCount": 2,
			})
		})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewOrgLevelHandler(c, baseURL)

	groups, err := h.ListGroups(context.Background(), LevelEnvironment, "abc12345", "")
	if err != nil {
		t.Fatalf("ListGroups() error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
}

func TestOrgLevelHandler_GetUserRequiresUUID(t *testing.T) {
	c, baseURL := newTestClientAndURL(t, http.NewServeMux())
	h := NewOrgLevelHandler(c, baseURL)
	if _, err := h.GetUser(context.Background(), LevelEnvironment, "abc12345", ""); err == nil {
		t.Fatal("expected an error for an empty user UUID")
	}
}

func TestOrgLevelHandler_GetUserFetchesByUUID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/platform/iam/v1/organizational-levels/environment/abc12345/users/u1",
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "u1", "email": "alice@example.com"})
		})

	c, baseURL := newTestClientAndURL(t, mux)
	h := NewOrgLevelHandler(c, baseURL)

	user, err := h.GetUser(context.Background(), LevelEnvironment, "abc12345", "u1")
	if err != nil {
		t.Fatalf("GetUser() error: %v", err)
	}
	if user["email"] != "alice@example.com" {
		t.Errorf("email = %v", user["email"])
	}
}

func TestOrgLevelHandler_RejectsInvalidLevelTypeBeforeRequest(t *testing.T) {
	c, baseURL := newTestClientAndURL(t, http.NewServeMux())
	h := NewOrgLevelHandler(c, baseURL)
	if _, err := h.ListGroups(context.Background(), "global", "x", ""); err == nil {
		t.Fatal("expected an error for level type 'global'")
	}
}

func TestOrgLevelHandler_RequiresLevelID(t *testing.T) {
	c, baseURL := newTestClientAndURL(t, http.NewServeMux())
	h := NewOrgLevelHandler(c, baseURL)
	if _, err := h.ListGroups(context.Background(), LevelEnvironment, "", ""); err == nil {
		t.Fatal("expected an error for an empty level ID")
	}
}
