package resources

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/client"
)

// mockTokenProvider implements client.TokenProvider for testing.
type mockTokenProvider struct{}

func (m *mockTokenProvider) GetHeaders() (http.Header, error) {
	h := http.Header{}
	h.Set("Authorization", "Bearer test-token")
	return h, nil
}

func (m *mockTokenProvider) IsValid() bool { return true }
func (m *mockTokenProvider) Close() error  { return nil }

// newTestClient creates a test HTTP server and a client.Client with its baseURL
// pointing at the server.
func newTestClient(t *testing.T, handler http.Handler) *client.Client {
	t.Helper()
	c, _ := newTestClientAndURL(t, handler)
	return c
}

// newTestClientAndURL is newTestClient plus the server's base URL.
//
// Handlers whose Path is an absolute URL (audit logs, reference data,
// notifications, environment-level IAM) ignore the client baseURL, so their
// tests must rewrite Path to point at the test server. Without the URL they
// would issue real requests to api.dynatrace.com.
func newTestClientAndURL(t *testing.T, handler http.Handler) (*client.Client, string) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := client.Config{
		AccountUUID:   "test-uuid",
		TokenProvider: &mockTokenProvider{},
		Timeout:       5 * time.Second,
		RetryConfig:   &client.RetryConfig{MaxRetries: 0},
	}
	c := client.New(cfg)
	c.SetBaseURL(server.URL)
	return c, server.URL
}
