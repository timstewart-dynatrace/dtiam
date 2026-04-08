package resources

import (
	"testing"
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
	if h.ListKey != "items" {
		t.Errorf("ListKey = %q, want 'items'", h.ListKey)
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
