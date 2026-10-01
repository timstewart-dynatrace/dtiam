// Package diagnostic provides enhanced error types with context, suggestions,
// and structured exit codes for dtiam CLI commands.
package diagnostic

import (
	"errors"
	"fmt"
	"strings"
)

// Exit codes for structured error reporting.
const (
	ExitSuccess   = 0
	ExitError     = 1 // General error
	ExitUsage     = 2 // Invalid usage (bad args, missing flags)
	ExitAuth      = 3 // Authentication failure (401)
	ExitNotFound  = 4 // Resource not found (404)
	ExitForbidden = 5 // Permission denied (403)
)

// Error is an enhanced error with operation context, HTTP status, and suggestions.
type Error struct {
	Operation   string   // e.g., "get groups", "delete policy"
	StatusCode  int      // HTTP status code (0 if not an API error)
	Message     string   // Human-readable error message
	RequestID   string   // Dynatrace request ID for support tickets
	Suggestions []string // Troubleshooting suggestions
	Err         error    // Underlying error
}

// Error implements the error interface.
func (e *Error) Error() string {
	var b strings.Builder
	if e.Operation != "" {
		b.WriteString(e.Operation)
		b.WriteString(": ")
	}
	b.WriteString(e.Message)
	if e.RequestID != "" {
		b.WriteString(fmt.Sprintf(" (request-id: %s)", e.RequestID))
	}
	return b.String()
}

// Unwrap returns the underlying error for errors.Is/As.
func (e *Error) Unwrap() error {
	return e.Err
}

// ExitCode returns the appropriate exit code for this error.
func (e *Error) ExitCode() int {
	switch {
	case e.StatusCode == 401:
		return ExitAuth
	case e.StatusCode == 403:
		return ExitForbidden
	case e.StatusCode == 404:
		return ExitNotFound
	case e.StatusCode >= 400 && e.StatusCode < 500:
		return ExitUsage
	default:
		return ExitError
	}
}

// HasSuggestions returns true if the error has troubleshooting suggestions.
func (e *Error) HasSuggestions() bool {
	return len(e.Suggestions) > 0
}

// FormatSuggestions returns suggestions as a formatted string.
func (e *Error) FormatSuggestions() string {
	if len(e.Suggestions) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Suggestions:\n")
	for _, s := range e.Suggestions {
		b.WriteString("  - ")
		b.WriteString(s)
		b.WriteString("\n")
	}
	return b.String()
}

// New creates a new diagnostic error.
func New(operation, message string) *Error {
	return &Error{
		Operation: operation,
		Message:   message,
	}
}

// Wrap wraps an existing error with diagnostic context.
func Wrap(err error, operation string) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		Operation: operation,
		Message:   err.Error(),
		Err:       err,
	}
}

// WrapWithStatus wraps an error with operation context and HTTP status.
func WrapWithStatus(err error, operation string, statusCode int) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		Operation:  operation,
		StatusCode: statusCode,
		Message:    err.Error(),
		Err:        err,
	}
}

// WithSuggestions adds suggestions to an existing diagnostic error.
func WithSuggestions(err *Error, suggestions ...string) *Error {
	if err == nil {
		return nil
	}
	err.Suggestions = append(err.Suggestions, suggestions...)
	return err
}

// ExitCodeForError returns the appropriate exit code for any error.
// It checks for *diagnostic.Error first, then falls back to ExitError.
func ExitCodeForError(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var diagErr *Error
	if errors.As(err, &diagErr) {
		return diagErr.ExitCode()
	}
	return ExitError
}

// URLSuggestions returns common suggestions for URL-related errors.
func URLSuggestions(statusCode int) []string {
	switch statusCode {
	case 401:
		return []string{
			"Check that your OAuth2 credentials are valid",
			"Run 'dtiam config view' to verify your context",
			"Try regenerating your client secret",
		}
	case 403:
		return []string{
			"Your OAuth2 client may lack required scopes",
			"Check 'account-idm-read' and 'iam-policies-management' scopes",
			"Run 'dtiam config get-credentials' to check your credentials",
		}
	case 404:
		return []string{
			"The resource may have been deleted",
			"Check the UUID or name is correct",
			"Run the corresponding 'get' command to list available resources",
		}
	default:
		return nil
	}
}
