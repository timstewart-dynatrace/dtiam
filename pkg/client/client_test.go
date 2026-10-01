package client

import (
	"testing"
)

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid JSON object",
			data:    []byte(`{"key": "value", "number": 42}`),
			wantErr: false,
		},
		{
			name:    "valid JSON array",
			data:    []byte(`[1, 2, 3]`),
			wantErr: false,
		},
		{
			name:    "empty object",
			data:    []byte(`{}`),
			wantErr: false,
		},
		{
			name:    "empty array",
			data:    []byte(`[]`),
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{invalid}`),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte(``),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result any
			err := ParseJSON(tt.data, &result)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseJSON() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseJSON() unexpected error: %v", err)
			}
		})
	}
}

func TestParseJSON_StructTarget(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	data := []byte(`{"name": "test", "value": 123}`)
	var result TestStruct

	err := ParseJSON(data, &result)
	if err != nil {
		t.Fatalf("ParseJSON() unexpected error: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("ParseJSON() Name = %v, want test", result.Name)
	}
	if result.Value != 123 {
		t.Errorf("ParseJSON() Value = %v, want 123", result.Value)
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "with message",
			err: &APIError{
				StatusCode: 404,
				Message:    "Resource not found",
			},
			want: "API error (status 404): Resource not found",
		},
		{
			name: "with response body",
			err: &APIError{
				StatusCode:   500,
				ResponseBody: "Internal server error",
			},
			want: "API error (status 500): Internal server error",
		},
		{
			name: "message takes precedence",
			err: &APIError{
				StatusCode:   403,
				Message:      "Access denied",
				ResponseBody: "ignored body",
			},
			want: "API error (status 403): Access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("APIError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_StatusChecks(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		isNotFound    bool
		isPermDenied  bool
		isConflict    bool
		isServerError bool
		isRetryable   bool
	}{
		{"404 Not Found", 404, true, false, false, false, false},
		{"403 Forbidden", 403, false, true, false, false, false},
		{"409 Conflict", 409, false, false, true, false, false},
		{"500 Server Error", 500, false, false, false, true, true},
		{"502 Bad Gateway", 502, false, false, false, true, true},
		{"503 Service Unavailable", 503, false, false, false, true, true},
		{"504 Gateway Timeout", 504, false, false, false, true, true},
		{"429 Too Many Requests", 429, false, false, false, false, true},
		{"400 Bad Request", 400, false, false, false, false, false},
		{"401 Unauthorized", 401, false, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}
			if err.IsNotFound() != tt.isNotFound {
				t.Errorf("IsNotFound() = %v, want %v", err.IsNotFound(), tt.isNotFound)
			}
			if err.IsPermissionDenied() != tt.isPermDenied {
				t.Errorf("IsPermissionDenied() = %v, want %v", err.IsPermissionDenied(), tt.isPermDenied)
			}
			if err.IsConflict() != tt.isConflict {
				t.Errorf("IsConflict() = %v, want %v", err.IsConflict(), tt.isConflict)
			}
			if err.IsServerError() != tt.isServerError {
				t.Errorf("IsServerError() = %v, want %v", err.IsServerError(), tt.isServerError)
			}
			if err.IsRetryable() != tt.isRetryable {
				t.Errorf("IsRetryable() = %v, want %v", err.IsRetryable(), tt.isRetryable)
			}
		})
	}
}
