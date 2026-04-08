// Package client provides an HTTP client for the Dynatrace IAM API.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/jtimothystewart/dtiam/internal/logging"
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

// Client is the HTTP client for the Dynatrace IAM API.
type Client struct {
	accountUUID   string
	tokenProvider TokenProvider
	resty         *resty.Client
	baseURL       string
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

// New creates a new API client.
func New(config Config) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	rc := DefaultRetryConfig()
	if config.RetryConfig != nil {
		rc = *config.RetryConfig
	}

	baseURL := fmt.Sprintf("%s/%s", BaseURL, config.AccountUUID)

	r := resty.New().
		SetTimeout(timeout).
		SetRetryCount(rc.MaxRetries).
		SetRetryWaitTime(rc.InitialDelay).
		SetRetryMaxWaitTime(rc.MaxDelay).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			if err != nil {
				return false
			}
			code := resp.StatusCode()
			for _, s := range rc.RetryStatuses {
				if code == s {
					return true
				}
			}
			return false
		}).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")

	if config.Verbose {
		r.SetDebug(true)
	}

	// Pre-request hook: inject auth headers
	r.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		headers, err := config.TokenProvider.GetHeaders()
		if err != nil {
			return fmt.Errorf("failed to get auth headers: %w", err)
		}
		for key, values := range headers {
			for _, value := range values {
				req.SetHeader(key, value)
			}
		}
		return nil
	})

	// Post-response hook: log HTTP requests
	r.OnAfterResponse(func(_ *resty.Client, resp *resty.Response) error {
		logging.HTTPRequest(resp.Request.Method, resp.Request.URL, resp.StatusCode())
		return nil
	})

	return &Client{
		accountUUID:   config.AccountUUID,
		tokenProvider: config.TokenProvider,
		resty:         r,
		baseURL:       baseURL,
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

// SetBaseURL overrides the base URL (for testing only).
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	url := c.buildURL(path)
	req := c.resty.R().SetContext(ctx)
	if len(params) > 0 {
		req.SetQueryParams(params)
	}

	resp, err := req.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return c.handleResponse(resp)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body any) ([]byte, error) {
	url := c.buildURL(path)
	resp, err := c.resty.R().SetContext(ctx).SetBody(body).Post(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return c.handleResponse(resp)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body any) ([]byte, error) {
	url := c.buildURL(path)
	resp, err := c.resty.R().SetContext(ctx).SetBody(body).Put(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return c.handleResponse(resp)
}

// Patch performs a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, body any) ([]byte, error) {
	url := c.buildURL(path)
	resp, err := c.resty.R().SetContext(ctx).SetBody(body).Patch(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return c.handleResponse(resp)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	url := c.buildURL(path)
	resp, err := c.resty.R().SetContext(ctx).Delete(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return c.handleResponse(resp)
}

// DeleteWithBody performs a DELETE request with a body.
func (c *Client) DeleteWithBody(ctx context.Context, path string, body any) ([]byte, error) {
	url := c.buildURL(path)
	resp, err := c.resty.R().SetContext(ctx).SetBody(body).Delete(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return c.handleResponse(resp)
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

// Close closes the client and releases resources.
func (c *Client) Close() error {
	return c.tokenProvider.Close()
}

// handleResponse checks the response status and returns the body or an error.
func (c *Client) handleResponse(resp *resty.Response) ([]byte, error) {
	if resp.StatusCode() >= 400 {
		apiErr := &APIError{
			StatusCode:   resp.StatusCode(),
			ResponseBody: string(resp.Body()),
		}

		// Try to extract error message from JSON
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(resp.Body(), &errResp) == nil {
			if errResp.Message != "" {
				apiErr.Message = errResp.Message
			} else if errResp.Error != "" {
				apiErr.Message = errResp.Error
			}
		}

		return nil, apiErr
	}

	return resp.Body(), nil
}

// buildURL constructs the full URL for a request.
func (c *Client) buildURL(path string) string {
	// Handle absolute URLs
	if len(path) > 7 && (path[:7] == "http://" || path[:8] == "https://") {
		return path
	}

	if len(path) > 0 && path[0] == '/' {
		return c.baseURL + path
	}
	return c.baseURL + "/" + path
}
