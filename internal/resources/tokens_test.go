package resources

import (
	"testing"

	"github.com/jtimothystewart/dtiam/internal/client"
)

func TestNewTokenHandler(t *testing.T) {
	h := NewTokenHandler(nil)
	if h.Name != "platform-token" {
		t.Errorf("Name = %q, want 'platform-token'", h.Name)
	}
	if h.Path != "/platform-tokens" {
		t.Errorf("Path = %q, want '/platform-tokens'", h.Path)
	}
	if h.IDField != "id" {
		t.Errorf("IDField = %q, want 'id'", h.IDField)
	}
	// The platform tokens API returns {pageSize, pageNumber, total, results};
	// reading "items" here yielded an empty list against the live API.
	if h.ListKey != "results" {
		t.Errorf("ListKey = %q, want 'results'", h.ListKey)
	}
	if h.Pagination == nil {
		t.Fatal("Pagination is nil; the platform tokens API is paginated")
	}
	if h.Pagination.Style != client.PaginationPageNumber {
		t.Errorf("Pagination.Style = %v, want PaginationPageNumber", h.Pagination.Style)
	}
	if h.Pagination.PageSizeParam != "size" {
		t.Errorf("PageSizeParam = %q, want 'size'", h.Pagination.PageSizeParam)
	}
}

func TestTokenHandler_ResourceName(t *testing.T) {
	h := NewTokenHandler(nil)
	if h.ResourceName() != "platform-token" {
		t.Errorf("ResourceName() = %q, want 'platform-token'", h.ResourceName())
	}
}

func TestTokenHandler_APIPath(t *testing.T) {
	h := NewTokenHandler(nil)
	if h.APIPath() != "/platform-tokens" {
		t.Errorf("APIPath() = %q, want '/platform-tokens'", h.APIPath())
	}
}
