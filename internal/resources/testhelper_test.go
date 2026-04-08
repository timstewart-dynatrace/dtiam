package resources

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jtimothystewart/dtiam/internal/client"
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
// pointing at the server. The caller must call t.Cleanup or defer server.Close().
func newTestClient(t *testing.T, handler http.Handler) *client.Client {
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
	return c
}
