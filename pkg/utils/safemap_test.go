package utils

import (
	"testing"
)

func TestStringFrom(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want string
	}{
		{"valid string", map[string]any{"name": "test"}, "name", "test"},
		{"missing key", map[string]any{"name": "test"}, "other", ""},
		{"nil map", nil, "name", ""},
		{"wrong type int", map[string]any{"name": 42}, "name", ""},
		{"wrong type bool", map[string]any{"name": true}, "name", ""},
		{"empty string", map[string]any{"name": ""}, "name", ""},
		{"nil value", map[string]any{"name": nil}, "name", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringFrom(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("StringFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIntFrom(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want int
	}{
		{"int value", map[string]any{"count": 42}, "count", 42},
		{"float64 value", map[string]any{"count": float64(42)}, "count", 42},
		{"int64 value", map[string]any{"count": int64(42)}, "count", 42},
		{"missing key", map[string]any{}, "count", 0},
		{"nil map", nil, "count", 0},
		{"wrong type string", map[string]any{"count": "42"}, "count", 0},
		{"nil value", map[string]any{"count": nil}, "count", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntFrom(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("IntFrom() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFloat64From(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want float64
	}{
		{"float64 value", map[string]any{"val": 3.14}, "val", 3.14},
		{"int value", map[string]any{"val": 42}, "val", 42.0},
		{"missing key", map[string]any{}, "val", 0},
		{"nil map", nil, "val", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Float64From(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("Float64From() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestBoolFrom(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want bool
	}{
		{"true value", map[string]any{"flag": true}, "flag", true},
		{"false value", map[string]any{"flag": false}, "flag", false},
		{"missing key", map[string]any{}, "flag", false},
		{"nil map", nil, "flag", false},
		{"wrong type", map[string]any{"flag": "true"}, "flag", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoolFrom(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("BoolFrom() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSliceFrom(t *testing.T) {
	tests := []struct {
		name    string
		m       map[string]any
		key     string
		wantLen int
		wantNil bool
	}{
		{"valid slice", map[string]any{"items": []any{"a", "b"}}, "items", 2, false},
		{"empty slice", map[string]any{"items": []any{}}, "items", 0, false},
		{"missing key", map[string]any{}, "items", 0, true},
		{"nil map", nil, "items", 0, true},
		{"wrong type", map[string]any{"items": "not a slice"}, "items", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SliceFrom(tt.m, tt.key)
			if tt.wantNil && got != nil {
				t.Errorf("SliceFrom() = %v, want nil", got)
			}
			if !tt.wantNil && len(got) != tt.wantLen {
				t.Errorf("SliceFrom() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestMapFrom(t *testing.T) {
	inner := map[string]any{"nested": "value"}
	tests := []struct {
		name    string
		m       map[string]any
		key     string
		wantNil bool
	}{
		{"valid map", map[string]any{"data": inner}, "data", false},
		{"missing key", map[string]any{}, "data", true},
		{"nil map", nil, "data", true},
		{"wrong type", map[string]any{"data": "not a map"}, "data", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapFrom(tt.m, tt.key)
			if tt.wantNil && got != nil {
				t.Errorf("MapFrom() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("MapFrom() = nil, want non-nil")
			}
		})
	}
}

func TestStringSliceFrom(t *testing.T) {
	tests := []struct {
		name    string
		m       map[string]any
		key     string
		want    []string
		wantNil bool
	}{
		{"[]string value", map[string]any{"tags": []string{"a", "b"}}, "tags", []string{"a", "b"}, false},
		{"[]any with strings", map[string]any{"tags": []any{"a", "b"}}, "tags", []string{"a", "b"}, false},
		{"[]any mixed", map[string]any{"tags": []any{"a", 42}}, "tags", []string{"a"}, false},
		{"missing key", map[string]any{}, "tags", nil, true},
		{"nil map", nil, "tags", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringSliceFrom(tt.m, tt.key)
			if tt.wantNil {
				if got != nil {
					t.Errorf("StringSliceFrom() = %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("StringSliceFrom() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("StringSliceFrom()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}
