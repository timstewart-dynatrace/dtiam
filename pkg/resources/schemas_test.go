package resources

import (
	"testing"
)

func TestNewSchemaHandler_FullURL(t *testing.T) {
	h := NewSchemaHandler(nil, "https://abc12345.live.dynatrace.com")
	if h.EnvironmentURL != "https://abc12345.live.dynatrace.com" {
		t.Errorf("EnvironmentURL = %q, want full URL", h.EnvironmentURL)
	}
}

func TestNewSchemaHandler_EnvID(t *testing.T) {
	h := NewSchemaHandler(nil, "abc12345")
	want := "https://abc12345.live.dynatrace.com"
	if h.EnvironmentURL != want {
		t.Errorf("EnvironmentURL = %q, want %q", h.EnvironmentURL, want)
	}
}

func TestNewSchemaHandler_TrailingSlash(t *testing.T) {
	h := NewSchemaHandler(nil, "https://abc12345.live.dynatrace.com/")
	want := "https://abc12345.live.dynatrace.com"
	if h.EnvironmentURL != want {
		t.Errorf("EnvironmentURL = %q, want %q (trailing slash stripped)", h.EnvironmentURL, want)
	}
}

func TestSchemaHandler_ResourceName(t *testing.T) {
	h := NewSchemaHandler(nil, "abc12345")
	if h.ResourceName() != "schema" {
		t.Errorf("ResourceName() = %q, want 'schema'", h.ResourceName())
	}
}

func TestSchemaHandler_APIPath(t *testing.T) {
	h := NewSchemaHandler(nil, "abc12345")
	want := "https://abc12345.live.dynatrace.com/api/v2/settings/schemas"
	if h.APIPath() != want {
		t.Errorf("APIPath() = %q, want %q", h.APIPath(), want)
	}
}
