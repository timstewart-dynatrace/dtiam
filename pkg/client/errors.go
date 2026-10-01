package client

import "fmt"

// APIError represents an error from the Dynatrace API.
type APIError struct {
	StatusCode   int
	Message      string
	ResponseBody string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.ResponseBody)
}

// IsNotFound returns true if the error is a 404.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsPermissionDenied returns true if the error is a 403.
func (e *APIError) IsPermissionDenied() bool {
	return e.StatusCode == 403
}

// IsConflict returns true if the error is a 409.
func (e *APIError) IsConflict() bool {
	return e.StatusCode == 409
}

// IsServerError returns true if the error is a 5xx status.
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsRetryable returns true if the request should be retried.
func (e *APIError) IsRetryable() bool {
	switch e.StatusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}
