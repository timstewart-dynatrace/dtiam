// Package client provides an HTTP client for the Dynatrace IAM API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TokenProvider provides authentication headers for HTTP requests.
type TokenProvider interface {
	// GetHeaders returns HTTP headers with valid Authorization.
	GetHeaders() (http.Header, error)

	// IsValid checks if the current token is valid.
	IsValid() bool

	// Close cleans up any resources.
	Close() error
}

const (
	// BaseURL is the base URL for the Dynatrace IAM API.
	// Deprecated: Use AccountsBasePath from urls.go instead.
	BaseURL = AccountsBasePath

	// DefaultTimeout is the default HTTP timeout.
	DefaultTimeout = 30 * time.Second
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries      int
	RetryStatuses   []int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	ExponentialBase float64
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		RetryStatuses:   []int{429, 500, 502, 503, 504},
		InitialDelay:    1 * time.Second,
		MaxDelay:        10 * time.Second,
		ExponentialBase: 2.0,
	}
}

// Client is the HTTP client for the Dynatrace IAM API.
type Client struct {
	accountUUID   string
	tokenProvider TokenProvider
	httpClient    *http.Client
	baseURL       string
	retryConfig   RetryConfig
	verbose       bool
}

// Config holds client configuration options.
type Config struct {
	AccountUUID   string
	TokenProvider TokenProvider
	Timeout       time.Duration
	RetryConfig   *RetryConfig
	Verbose       bool
}

// New creates a new API client.
func New(config Config) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	retryConfig := DefaultRetryConfig()
	if config.RetryConfig != nil {
		retryConfig = *config.RetryConfig
	}

	return &Client{
		accountUUID:   config.AccountUUID,
		tokenProvider: config.TokenProvider,
		httpClient:    &http.Client{Timeout: timeout},
		baseURL:       fmt.Sprintf("%s/%s", BaseURL, config.AccountUUID),
		retryConfig:   retryConfig,
		verbose:       config.Verbose,
	}
}

// AccountUUID returns the account UUID.
func (c *Client) AccountUUID() string {
	return c.accountUUID
}

// BaseURL returns the base URL for the API.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	return c.request(ctx, http.MethodGet, path, params, nil)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body any) ([]byte, error) {
	return c.request(ctx, http.MethodPost, path, nil, body)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body any) ([]byte, error) {
	return c.request(ctx, http.MethodPut, path, nil, body)
}

// Patch performs a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, body any) ([]byte, error) {
	return c.request(ctx, http.MethodPatch, path, nil, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, http.MethodDelete, path, nil, nil)
}

// DeleteWithBody performs a DELETE request with a body.
func (c *Client) DeleteWithBody(ctx context.Context, path string, body any) ([]byte, error) {
	return c.request(ctx, http.MethodDelete, path, nil, body)
}

// request performs an HTTP request with retry logic.
func (c *Client) request(ctx context.Context, method, path string, params map[string]string, body any) ([]byte, error) {
	// Build URL
	reqURL := c.buildURL(path, params)

	// Marshal body if present
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	var lastErr error
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateDelay(attempt, nil)
			if c.verbose {
				fmt.Printf("Retrying request (attempt %d/%d) after %v\n", attempt+1, c.retryConfig.MaxRetries+1, delay)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			// Reset body reader for retry
			if body != nil {
				jsonBody, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(jsonBody)
			}
		}

		respBody, err := c.doRequest(ctx, method, reqURL, bodyReader)
		if err == nil {
			return respBody, nil
		}

		lastErr = err

		// Check if error is retryable
		apiErr, ok := err.(*APIError)
		if !ok || !apiErr.IsRetryable() {
			return nil, err
		}
	}

	return nil, lastErr
}

// doRequest performs a single HTTP request.
func (c *Client) doRequest(ctx context.Context, method, reqURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Get auth headers
	headers, err := c.tokenProvider.GetHeaders()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth headers: %w", err)
	}

	// Copy headers to request
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if c.verbose {
		fmt.Printf("%s %s\n", method, reqURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.verbose {
		fmt.Printf("Response status: %d\n", resp.StatusCode)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		apiErr := &APIError{
			StatusCode:   resp.StatusCode,
			ResponseBody: string(respBody),
		}

		// Try to extract error message from JSON
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &errResp) == nil {
			if errResp.Message != "" {
				apiErr.Message = errResp.Message
			} else if errResp.Error != "" {
				apiErr.Message = errResp.Error
			}
		}

		return nil, apiErr
	}

	return respBody, nil
}

// buildURL constructs the full URL for a request.
func (c *Client) buildURL(path string, params map[string]string) string {
	// Handle absolute URLs
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if len(params) == 0 {
			return path
		}
		return path + "?" + buildQueryString(params)
	}

	// Handle relative paths
	var fullURL string
	if strings.HasPrefix(path, "/") {
		fullURL = c.baseURL + path
	} else {
		fullURL = c.baseURL + "/" + path
	}

	if len(params) > 0 {
		fullURL += "?" + buildQueryString(params)
	}

	return fullURL
}

// buildQueryString builds a URL query string from params.
func buildQueryString(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return values.Encode()
}

// calculateDelay calculates the delay for a retry attempt.
func (c *Client) calculateDelay(attempt int, resp *http.Response) time.Duration {
	// Check for Retry-After header
	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	// Exponential backoff
	delay := float64(c.retryConfig.InitialDelay) * math.Pow(c.retryConfig.ExponentialBase, float64(attempt-1))
	if delay > float64(c.retryConfig.MaxDelay) {
		delay = float64(c.retryConfig.MaxDelay)
	}

	return time.Duration(delay)
}

// Close closes the client and releases resources.
func (c *Client) Close() error {
	return c.tokenProvider.Close()
}

// GetJSON performs a GET request and unmarshals the response into v.
func (c *Client) GetJSON(ctx context.Context, path string, params map[string]string, v any) error {
	body, err := c.Get(ctx, path, params)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// PostJSON performs a POST request and unmarshals the response into v.
func (c *Client) PostJSON(ctx context.Context, path string, reqBody any, v any) error {
	body, err := c.Post(ctx, path, reqBody)
	if err != nil {
		return err
	}
	if v != nil && len(body) > 0 {
		return json.Unmarshal(body, v)
	}
	return nil
}

// PutJSON performs a PUT request and unmarshals the response into v.
func (c *Client) PutJSON(ctx context.Context, path string, reqBody any, v any) error {
	body, err := c.Put(ctx, path, reqBody)
	if err != nil {
		return err
	}
	if v != nil && len(body) > 0 {
		return json.Unmarshal(body, v)
	}
	return nil
}

// ParseJSON is a helper function to unmarshal JSON into the provided value.
func ParseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
