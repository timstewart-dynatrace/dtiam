package client

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestExtractErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			// The Account Management shape. Its boolean "error" made the old
			// struct decode fail, so every one of these messages printed blank.
			name: "should read message when error is a boolean",
			body: `{"error":true,"message":"payload.map is not a function","payload":null}`,
			want: "payload.map is not a function",
		},
		{
			name: "should read a nested error message from the environment Platform APIs",
			body: `{"error":{"code":400,"message":"Mandatory query param partialGroupName or uuid was not provided"}}`,
			want: "Mandatory query param partialGroupName or uuid was not provided",
		},
		{
			name: "should read a string error",
			body: `{"error":"invalid_request"}`,
			want: "invalid_request",
		},
		{
			name: "should fall back to error_description",
			body: `{"error":{"code":1},"error_description":"bad scope"}`,
			want: "bad scope",
		},
		{
			name: "should return a non-JSON body as text",
			body: "  Bad Gateway \n",
			want: "Bad Gateway",
		},
		{
			name: "should return empty when there is nothing to report",
			body: `{"error":true}`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractErrorMessage([]byte(tt.body)); got != tt.want {
				t.Errorf("extractErrorMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIHostOverride(t *testing.T) {
	tests := []struct {
		name     string
		override string
		in       string
		want     string
	}{
		{
			name:     "should leave URLs alone when no override is set",
			override: "",
			in:       "https://api.dynatrace.com/iam/v1/repo/account/a/bindings",
			want:     "https://api.dynatrace.com/iam/v1/repo/account/a/bindings",
		},
		{
			name:     "should move every account API to the override host",
			override: "https://api.example.com",
			in:       "https://api.dynatrace.com/sub/v3/accounts/a/subscriptions",
			want:     "https://api.example.com/sub/v3/accounts/a/subscriptions",
		},
		{
			name:     "should tolerate an override written with a path and trailing slash",
			override: "https://api.example.com/iam/v1/",
			in:       "https://api.dynatrace.com/iam/v1/accounts/a/groups",
			want:     "https://api.example.com/iam/v1/accounts/a/groups",
		},
		{
			name:     "should not touch environment URLs",
			override: "https://api.example.com",
			in:       "https://abc12345.apps.dynatrace.com/platform/iam/v1/x",
			want:     "https://abc12345.apps.dynatrace.com/platform/iam/v1/x",
		},
		{
			name:     "should not touch a host that merely shares the prefix",
			override: "https://api.example.com",
			in:       "https://api.dynatrace.com.evil.test/x",
			want:     "https://api.dynatrace.com.evil.test/x",
		},
		{
			name:     "should treat the default host as no override",
			override: "https://api.dynatrace.com",
			in:       "https://api.dynatrace.com/ref/v1/account/permissions",
			want:     "https://api.dynatrace.com/ref/v1/account/permissions",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(Config{AccountUUID: "a", TokenProvider: &mockTokenProvider{token: "t"}, APIHost: tt.override})
			if got := c.buildURL(tt.in); got != tt.want {
				t.Errorf("buildURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIHostOverride_AppliesToRelativePaths(t *testing.T) {
	c := New(Config{AccountUUID: "a", TokenProvider: &mockTokenProvider{token: "t"}, APIHost: "https://api.example.com"})
	if got, want := c.buildURL("/groups"), "https://api.example.com/iam/v1/accounts/a/groups"; got != want {
		t.Errorf("buildURL(/groups) = %q, want %q", got, want)
	}
}

func TestDeleteWithQuery_RepeatsParameters(t *testing.T) {
	var got url.Values
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		got = r.URL.Query()
		w.WriteHeader(200)
	})
	defer server.Close()

	query := url.Values{}
	query.Add("group-uuid", "g1")
	query.Add("group-uuid", "g2")
	if _, err := c.DeleteWithQuery(context.Background(), "/users/a@example.com/groups", query); err != nil {
		t.Fatalf("DeleteWithQuery() error: %v", err)
	}
	if v := got["group-uuid"]; len(v) != 2 || v[0] != "g1" || v[1] != "g2" {
		t.Errorf("group-uuid = %v, want [g1 g2]", v)
	}
}
