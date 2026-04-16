package diagnostic

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		contains []string
	}{
		{
			"with operation and message",
			&Error{Operation: "get groups", Message: "connection refused"},
			[]string{"get groups", "connection refused"},
		},
		{
			"with request ID",
			&Error{Operation: "delete policy", Message: "not found", RequestID: "abc-123"},
			[]string{"delete policy", "not found", "request-id: abc-123"},
		},
		{
			"message only",
			&Error{Message: "something failed"},
			[]string{"something failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, want := range tt.contains {
				if !strings.Contains(msg, want) {
					t.Errorf("Error() = %q, want to contain %q", msg, want)
				}
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &Error{Message: "outer", Err: inner}

	if !errors.Is(err, inner) {
		t.Error("Unwrap: errors.Is should match inner error")
	}
}

func TestError_ExitCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   int
	}{
		{"401 → auth", 401, ExitAuth},
		{"403 → forbidden", 403, ExitForbidden},
		{"404 → not found", 404, ExitNotFound},
		{"400 → usage", 400, ExitUsage},
		{"409 → usage", 409, ExitUsage},
		{"500 → general", 500, ExitError},
		{"0 → general", 0, ExitError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &Error{StatusCode: tt.statusCode, Message: "test"}
			if got := err.ExitCode(); got != tt.expected {
				t.Errorf("ExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestError_HasSuggestions(t *testing.T) {
	err := &Error{Message: "test"}
	if err.HasSuggestions() {
		t.Error("HasSuggestions() should be false with no suggestions")
	}

	err.Suggestions = []string{"try this"}
	if !err.HasSuggestions() {
		t.Error("HasSuggestions() should be true with suggestions")
	}
}

func TestError_FormatSuggestions(t *testing.T) {
	err := &Error{Message: "test"}
	if got := err.FormatSuggestions(); got != "" {
		t.Errorf("FormatSuggestions() with none = %q, want empty", got)
	}

	err.Suggestions = []string{"check credentials", "run doctor"}
	got := err.FormatSuggestions()
	if !strings.Contains(got, "Suggestions:") {
		t.Errorf("FormatSuggestions() should contain header, got %q", got)
	}
	if !strings.Contains(got, "- check credentials") {
		t.Errorf("FormatSuggestions() should contain first suggestion, got %q", got)
	}
	if !strings.Contains(got, "- run doctor") {
		t.Errorf("FormatSuggestions() should contain second suggestion, got %q", got)
	}
}

func TestNew(t *testing.T) {
	err := New("get groups", "connection refused")
	if err.Operation != "get groups" {
		t.Errorf("New().Operation = %q, want 'get groups'", err.Operation)
	}
	if err.Message != "connection refused" {
		t.Errorf("New().Message = %q, want 'connection refused'", err.Message)
	}
}

func TestWrap(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		inner := fmt.Errorf("inner")
		err := Wrap(inner, "delete policy")
		if err.Operation != "delete policy" {
			t.Errorf("Wrap().Operation = %q, want 'delete policy'", err.Operation)
		}
		if err.Err != inner {
			t.Error("Wrap().Err should be the inner error")
		}
	})

	t.Run("nil returns nil", func(t *testing.T) {
		if Wrap(nil, "op") != nil {
			t.Error("Wrap(nil) should return nil")
		}
	})
}

func TestWrapWithStatus(t *testing.T) {
	inner := fmt.Errorf("not found")
	err := WrapWithStatus(inner, "get group", 404)
	if err.StatusCode != 404 {
		t.Errorf("WrapWithStatus().StatusCode = %d, want 404", err.StatusCode)
	}
	if err.ExitCode() != ExitNotFound {
		t.Errorf("WrapWithStatus().ExitCode() = %d, want ExitNotFound", err.ExitCode())
	}

	if WrapWithStatus(nil, "op", 500) != nil {
		t.Error("WrapWithStatus(nil) should return nil")
	}
}

func TestWithSuggestions(t *testing.T) {
	err := New("op", "msg")
	_ = WithSuggestions(err, "try A", "try B")
	if len(err.Suggestions) != 2 {
		t.Errorf("WithSuggestions: expected 2 suggestions, got %d", len(err.Suggestions))
	}

	if WithSuggestions(nil, "x") != nil {
		t.Error("WithSuggestions(nil) should return nil")
	}
}

func TestExitCodeForError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if got := ExitCodeForError(nil); got != ExitSuccess {
			t.Errorf("ExitCodeForError(nil) = %d, want 0", got)
		}
	})

	t.Run("diagnostic error", func(t *testing.T) {
		err := &Error{StatusCode: 403, Message: "forbidden"}
		if got := ExitCodeForError(err); got != ExitForbidden {
			t.Errorf("ExitCodeForError(403) = %d, want %d", got, ExitForbidden)
		}
	})

	t.Run("wrapped diagnostic error", func(t *testing.T) {
		inner := &Error{StatusCode: 401, Message: "unauthorized"}
		wrapped := fmt.Errorf("context: %w", inner)
		if got := ExitCodeForError(wrapped); got != ExitAuth {
			t.Errorf("ExitCodeForError(wrapped 401) = %d, want %d", got, ExitAuth)
		}
	})

	t.Run("plain error", func(t *testing.T) {
		err := fmt.Errorf("plain error")
		if got := ExitCodeForError(err); got != ExitError {
			t.Errorf("ExitCodeForError(plain) = %d, want %d", got, ExitError)
		}
	})
}

func TestURLSuggestions(t *testing.T) {
	tests := []struct {
		status   int
		wantLen  int
	}{
		{401, 3},
		{403, 3},
		{404, 3},
		{500, 0},
		{200, 0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			got := URLSuggestions(tt.status)
			if len(got) != tt.wantLen {
				t.Errorf("URLSuggestions(%d) returned %d suggestions, want %d", tt.status, len(got), tt.wantLen)
			}
		})
	}
}

func TestErrorsAs_Integration(t *testing.T) {
	// Verify that errors.As works through wrapping chains
	original := WrapWithStatus(fmt.Errorf("api: 404"), "get group", 404)
	wrapped := fmt.Errorf("command failed: %w", original)
	doubleWrapped := fmt.Errorf("execute: %w", wrapped)

	var diagErr *Error
	if !errors.As(doubleWrapped, &diagErr) {
		t.Fatal("errors.As should find *diagnostic.Error through wrapping")
	}
	if diagErr.StatusCode != 404 {
		t.Errorf("extracted error status = %d, want 404", diagErr.StatusCode)
	}
	if diagErr.ExitCode() != ExitNotFound {
		t.Errorf("extracted error exit code = %d, want %d", diagErr.ExitCode(), ExitNotFound)
	}
}
