package client

import (
	"strings"
	"testing"
)

func TestURLConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		contains string
	}{
		{"IAMBaseURL has iam/v1", IAMBaseURL, "iam/v1"},
		{"AccountsBasePath extends IAMBaseURL", AccountsBasePath, IAMBaseURL},
		{"RepoBasePath extends IAMBaseURL", RepoBasePath, IAMBaseURL + "/repo"},
		{"ResolutionBasePath extends IAMBaseURL", ResolutionBasePath, IAMBaseURL + "/resolution"},
		{"EnvBaseURL has env/v2", EnvBaseURL, "env/v2"},
		{"SubBaseURL has sub/v2", SubBaseURL, "sub/v2"},
		{"SSOTokenURL has sso", SSOTokenURL, "sso.dynatrace.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.constant, tt.contains) {
				t.Errorf("%s = %q, want to contain %q", tt.name, tt.constant, tt.contains)
			}
		})
	}
}

func TestBaseURLBackwardCompat(t *testing.T) {
	// BaseURL should equal AccountsBasePath for backward compatibility
	if BaseURL != AccountsBasePath {
		t.Errorf("BaseURL = %q, want %q (must match AccountsBasePath)", BaseURL, AccountsBasePath)
	}
}
