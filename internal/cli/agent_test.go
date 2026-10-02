package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/safety"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"should mark a safety block", &safety.BlockedError{Context: "prod", Level: "readonly", Operation: safety.Delete, Command: "dtiam delete group"}, "safety_blocked"},
		{"should classify a wrapped 403", fmt.Errorf("failed: %w", &client.APIError{StatusCode: 403}), "permission_denied"},
		{"should classify a 404", &client.APIError{StatusCode: 404}, "not_found"},
		{"should classify a 429", &client.APIError{StatusCode: 429}, "rate_limited"},
		{"should classify a 503", &client.APIError{StatusCode: 503}, "server_error"},
		{"should classify an unknown command", errors.New(`unknown command "groupz" for "dtiam get"`), "usage"},
		{"should classify a missing required flag", errors.New(`required flag(s) "name" not set`), "usage"},
		{"should classify a not-found message", errors.New(`group "x" not found`), "not_found"},
		{"should fall back to error", errors.New("boom"), "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyError(tt.err); got.Code != tt.want {
				t.Errorf("code = %q, want %q", got.Code, tt.want)
			}
		})
	}
}

func TestClassifyError_KeepsStatusCode(t *testing.T) {
	got := classifyError(fmt.Errorf("x: %w", &client.APIError{StatusCode: 409}))
	if got.StatusCode != 409 || got.Code != "conflict" {
		t.Errorf("got %+v", got)
	}
}
