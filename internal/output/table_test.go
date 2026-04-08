package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTableFormatter_Format(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewTableFormatter(buf, false)

	columns := []Column{
		{Key: "id", Header: "ID"},
		{Key: "name", Header: "NAME"},
	}
	data := []map[string]any{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
	}

	if err := f.Format(data, columns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ID") {
		t.Error("table missing ID header")
	}
	if !strings.Contains(out, "NAME") {
		t.Error("table missing NAME header")
	}
	if !strings.Contains(out, "Alice") {
		t.Error("table missing data value Alice")
	}
	if !strings.Contains(out, "Bob") {
		t.Error("table missing data value Bob")
	}
}

func TestTableFormatter_Format_EmptyData(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewTableFormatter(buf, false)

	columns := []Column{
		{Key: "id", Header: "ID"},
	}

	if err := f.Format([]map[string]any{}, columns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	if !strings.Contains(buf.String(), "No resources found") {
		t.Errorf("empty data should show 'No resources found', got: %q", buf.String())
	}
}

func TestTableFormatter_FormatSingle(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewTableFormatter(buf, false)

	columns := []Column{
		{Key: "id", Header: "ID"},
		{Key: "name", Header: "NAME"},
	}
	data := map[string]any{"id": "abc-123", "name": "Test"}

	if err := f.FormatSingle(data, columns); err != nil {
		t.Fatalf("FormatSingle() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ID") {
		t.Error("single format missing ID label")
	}
	if !strings.Contains(out, "abc-123") {
		t.Error("single format missing id value")
	}
	if !strings.Contains(out, "Test") {
		t.Error("single format missing name value")
	}
}

func TestTableFormatter_FormatSingle_Nil(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewTableFormatter(buf, false)

	columns := []Column{
		{Key: "id", Header: "ID"},
	}

	if err := f.FormatSingle(nil, columns); err != nil {
		t.Fatalf("FormatSingle() error: %v", err)
	}

	if !strings.Contains(buf.String(), "No resource found") {
		t.Errorf("nil data should show 'No resource found', got: %q", buf.String())
	}
}

func TestExtractValue(t *testing.T) {
	tests := []struct {
		name   string
		data   map[string]any
		col    Column
		want   string
	}{
		{
			name: "simple key",
			data: map[string]any{"name": "Alice"},
			col:  Column{Key: "name", Header: "NAME"},
			want: "Alice",
		},
		{
			name: "nested dot notation",
			data: map[string]any{"meta": map[string]any{"version": "1.0"}},
			col:  Column{Key: "meta.version", Header: "VERSION"},
			want: "1.0",
		},
		{
			name: "missing key returns empty",
			data: map[string]any{"name": "Alice"},
			col:  Column{Key: "missing", Header: "MISSING"},
			want: "",
		},
		{
			name: "custom formatter",
			data: map[string]any{"items": []any{"a", "b"}},
			col: Column{
				Key:    "items",
				Header: "ITEMS",
				Formatter: func(v any) string {
					if arr, ok := v.([]any); ok {
						return strings.Join(func() []string {
							s := make([]string, len(arr))
							for i, a := range arr {
								s[i] = a.(string)
							}
							return s
						}(), "+")
					}
					return ""
				},
			},
			want: "a+b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractValue(tt.data, tt.col)
			if got != tt.want {
				t.Errorf("extractValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetNestedValue(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		key  string
		want any
	}{
		{
			name: "top-level key",
			data: map[string]any{"name": "Alice"},
			key:  "name",
			want: "Alice",
		},
		{
			name: "nested key",
			data: map[string]any{"a": map[string]any{"b": map[string]any{"c": "deep"}}},
			key:  "a.b.c",
			want: "deep",
		},
		{
			name: "missing nested key",
			data: map[string]any{"a": map[string]any{"b": "val"}},
			key:  "a.x.y",
			want: nil,
		},
		{
			name: "non-map intermediate",
			data: map[string]any{"a": "string-not-map"},
			key:  "a.b",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNestedValue(tt.data, tt.key)
			if got != tt.want {
				t.Errorf("getNestedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"float64 integer", float64(42), "42"},
		{"float64 decimal", float64(3.14), "3.14"},
		{"int", 7, "7"},
		{"empty slice", []any{}, ""},
		{"small slice", []any{"a", "b"}, "a, b"},
		{"large slice", []any{"a", "b", "c", "d"}, "4 items"},
		{"map", map[string]any{"k1": "v1", "k2": "v2"}, "{2 keys}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.input)
			if got != tt.want {
				t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
