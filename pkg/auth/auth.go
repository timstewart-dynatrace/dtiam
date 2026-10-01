// Package auth provides authentication mechanisms for the Dynatrace IAM API.
package auth

import "net/http"

// TokenProvider provides authentication headers for HTTP requests.
type TokenProvider interface {
	// GetHeaders returns HTTP headers with valid Authorization.
	GetHeaders() (http.Header, error)

	// IsValid checks if the current token is valid.
	IsValid() bool

	// Close cleans up any resources.
	Close() error
}
