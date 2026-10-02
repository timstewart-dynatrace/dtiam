// Package client provides an HTTP client for the Dynatrace IAM API.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/logging"
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
	apiHost       string
	verbose       bool
}

// Config holds client configuration options.
type Config struct {
	AccountUUID   string
	TokenProvider TokenProvider
	Timeout       time.Duration
	RetryConfig   *RetryConfig
	Verbose       bool

	// APIHost overrides DefaultAPIHost for every account API request, relative
	// or absolute. Empty means the default. A trailing path such as /iam/v1 is
	// tolerated and stripped, since that is how the override is often written.
	APIHost string
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

	apiHost := normalizeAPIHost(config.APIHost)
	baseURL := rewriteHost(fmt.Sprintf("%s/%s", BaseURL, config.AccountUUID), apiHost)

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
		apiHost:       apiHost,
		verbose:       config.Verbose,
	}
}

// AccountUUID returns the account UUID.
func (c *Client) AccountUUID() string {
	return c.accountUUID
}

// AccessToken returns the bearer token the client currently authenticates
// with, obtaining or refreshing it as needed.
func (c *Client) AccessToken() (string, error) {
	if c.tokenProvider == nil {
		return "", fmt.Errorf("no token provider configured")
	}
	headers, err := c.tokenProvider.GetHeaders()
	if err != nil {
		return "", err
	}
	token := strings.TrimPrefix(headers.Get("Authorization"), "Bearer ")
	if token == "" {
		return "", fmt.Errorf("the token provider returned no bearer token")
	}
	return token, nil
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

// GetWithQuery performs a GET request with query parameters that may repeat,
// such as types=BUDGET&types=COST on the v2 notifications API, which rejects
// the comma-separated form with HTTP 400.
func (c *Client) GetWithQuery(ctx context.Context, path string, query url.Values) ([]byte, error) {
	resp, err := c.resty.R().SetContext(ctx).SetQueryParamsFromValues(query).Get(c.buildURL(path))
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

// DeleteWithParams performs a DELETE request with query parameters.
func (c *Client) DeleteWithParams(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}
	return c.DeleteWithQuery(ctx, path, query)
}

// DeleteWithQuery performs a DELETE request with query parameters that may
// repeat, such as the group-uuid list on DELETE /users/{email}/groups.
func (c *Client) DeleteWithQuery(ctx context.Context, path string, query url.Values) ([]byte, error) {
	resp, err := c.resty.R().SetContext(ctx).SetQueryParamsFromValues(query).Delete(c.buildURL(path))
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

		apiErr.Message = extractErrorMessage(resp.Body())

		return nil, apiErr
	}

	return resp.Body(), nil
}

// buildURL constructs the full URL for a request.
func (c *Client) buildURL(path string) string {
	// Handle absolute URLs
	if len(path) > 7 && (path[:7] == "http://" || path[:8] == "https://") {
		return rewriteHost(path, c.apiHost)
	}

	if len(path) > 0 && path[0] == '/' {
		return c.baseURL + path
	}
	return c.baseURL + "/" + path
}

// normalizeAPIHost reduces an API host override to scheme and host, so that
// "https://api.example.com/iam/v1/" and "https://api.example.com" are the same.
// An empty or default value yields "", meaning no rewrite.
func normalizeAPIHost(host string) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		return ""
	}
	if u, err := url.Parse(host); err == nil && u.Scheme != "" && u.Host != "" {
		host = u.Scheme + "://" + u.Host
	}
	if host == DefaultAPIHost {
		return ""
	}
	return host
}

// rewriteHost replaces DefaultAPIHost at the start of rawURL with apiHost.
// URLs on other hosts -- environment URLs, the SSO endpoint -- are unchanged.
func rewriteHost(rawURL, apiHost string) string {
	if apiHost == "" || !strings.HasPrefix(rawURL, DefaultAPIHost) {
		return rawURL
	}
	rest := rawURL[len(DefaultAPIHost):]
	if rest != "" && rest[0] != '/' && rest[0] != '?' {
		return rawURL // a different host that merely shares the prefix
	}
	return apiHost + rest
}

// extractErrorMessage pulls a human-readable message out of an error response.
//
// The APIs disagree on the shape: the Account Management API sends
// {"error": true, "message": "..."}, the environment Platform APIs send
// {"error": {"message": "..."}}, and others send {"error": "..."}. Decoding
// into a struct with a string "error" field failed on the boolean form and
// discarded the message, so every Account Management error printed blank.
func extractErrorMessage(body []byte) string {
	var resp map[string]any
	if json.Unmarshal(body, &resp) != nil {
		return strings.TrimSpace(string(body))
	}
	if msg, ok := resp["message"].(string); ok && msg != "" {
		return msg
	}
	switch e := resp["error"].(type) {
	case string:
		return e
	case map[string]any:
		if msg, ok := e["message"].(string); ok {
			return msg
		}
	}
	if desc, ok := resp["error_description"].(string); ok {
		return desc
	}
	return ""
}
