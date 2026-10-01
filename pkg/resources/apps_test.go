package resources

import (
	"testing"
)

func TestNewAppHandler_FullURL(t *testing.T) {
	h := NewAppHandler(nil, "https://abc12345.apps.dynatrace.com")
	if h.EnvironmentURL != "https://abc12345.apps.dynatrace.com" {
		t.Errorf("EnvironmentURL = %q, want full URL", h.EnvironmentURL)
	}
}

func TestNewAppHandler_EnvID(t *testing.T) {
	h := NewAppHandler(nil, "abc12345")
	want := "https://abc12345.apps.dynatrace.com"
	if h.EnvironmentURL != want {
		t.Errorf("EnvironmentURL = %q, want %q", h.EnvironmentURL, want)
	}
}

func TestNewAppHandler_TrailingSlash(t *testing.T) {
	h := NewAppHandler(nil, "https://abc12345.apps.dynatrace.com/")
	want := "https://abc12345.apps.dynatrace.com"
	if h.EnvironmentURL != want {
		t.Errorf("EnvironmentURL = %q, want %q (trailing slash stripped)", h.EnvironmentURL, want)
	}
}

func TestAppHandler_ResourceName(t *testing.T) {
	h := NewAppHandler(nil, "abc12345")
	if h.ResourceName() != "app" {
		t.Errorf("ResourceName() = %q, want 'app'", h.ResourceName())
	}
}

func TestAppHandler_APIPath(t *testing.T) {
	h := NewAppHandler(nil, "abc12345")
	want := "https://abc12345.apps.dynatrace.com/platform/app-engine/registry/v1/apps"
	if h.APIPath() != want {
		t.Errorf("APIPath() = %q, want %q", h.APIPath(), want)
	}
}
