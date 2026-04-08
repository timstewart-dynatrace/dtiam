package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func testColumns() []Column {
	return []Column{
		{Key: "id", Header: "ID"},
		{Key: "name", Header: "NAME"},
		{Key: "detail", Header: "DETAIL", WideOnly: true},
	}
}

func testData() []map[string]any {
	return []map[string]any{
		{"id": "abc-123", "name": "Alpha", "detail": "extra-a"},
		{"id": "def-456", "name": "Beta", "detail": "extra-b"},
	}
}

func newTestPrinter(format Format, plain bool) (*Printer, *bytes.Buffer) {
	p := NewPrinter(format, plain)
	buf := &bytes.Buffer{}
	p.SetWriter(buf)
	return p, buf
}

func TestPrinter_Print_JSON(t *testing.T) {
	p, buf := newTestPrinter(FormatJSON, false)
	data := testData()

	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print() error: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if result[0]["name"] != "Alpha" {
		t.Errorf("expected name Alpha, got %v", result[0]["name"])
	}
}

func TestPrinter_Print_YAML(t *testing.T) {
	p, buf := newTestPrinter(FormatYAML, false)
	data := testData()

	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print() error: %v", err)
	}

	var result []map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid YAML: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestPrinter_Print_CSV(t *testing.T) {
	p, buf := newTestPrinter(FormatCSV, false)
	data := testData()

	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 CSV lines (header + 2 data), got %d", len(lines))
	}

	// Header should contain all columns (including wide-only for CSV)
	header := lines[0]
	for _, col := range testColumns() {
		if !strings.Contains(header, col.Header) {
			t.Errorf("CSV header missing %q", col.Header)
		}
	}

	// Data rows should contain values
	if !strings.Contains(lines[1], "abc-123") {
		t.Errorf("first data row missing id abc-123: %s", lines[1])
	}
}

func TestPrinter_Print_Table(t *testing.T) {
	p, buf := newTestPrinter(FormatTable, false)
	data := testData()

	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print() error: %v", err)
	}

	out := buf.String()
	// Table should contain non-wide headers
	if !strings.Contains(out, "ID") {
		t.Error("table output missing ID header")
	}
	if !strings.Contains(out, "NAME") {
		t.Error("table output missing NAME header")
	}
	// Wide-only column should NOT appear in table mode
	if strings.Contains(out, "DETAIL") {
		t.Error("table output should not contain wide-only DETAIL column")
	}
	// Data should be present
	if !strings.Contains(out, "Alpha") {
		t.Error("table output missing data value Alpha")
	}
}

func TestPrinter_Print_Wide(t *testing.T) {
	p, buf := newTestPrinter(FormatWide, false)
	data := testData()

	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print() error: %v", err)
	}

	out := buf.String()
	// Wide mode should include wide-only columns
	if !strings.Contains(out, "DETAIL") {
		t.Error("wide output missing DETAIL column")
	}
	if !strings.Contains(out, "extra-a") {
		t.Error("wide output missing detail value extra-a")
	}
}

func TestPrinter_PrintSingle(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		check  func(t *testing.T, out string)
	}{
		{
			name:   "json format",
			format: FormatJSON,
			check: func(t *testing.T, out string) {
				var result map[string]any
				if err := json.Unmarshal([]byte(out), &result); err != nil {
					t.Fatalf("not valid JSON: %v", err)
				}
				if result["name"] != "Alpha" {
					t.Errorf("expected name Alpha, got %v", result["name"])
				}
			},
		},
		{
			name:   "table format",
			format: FormatTable,
			check: func(t *testing.T, out string) {
				if !strings.Contains(out, "Alpha") {
					t.Error("table single output missing Alpha")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, buf := newTestPrinter(tt.format, false)
			data := testData()[0]
			if err := p.PrintSingle(data, testColumns()); err != nil {
				t.Fatalf("PrintSingle() error: %v", err)
			}
			tt.check(t, buf.String())
		})
	}
}

func TestPrinter_PrintDetail(t *testing.T) {
	p, buf := newTestPrinter(FormatTable, false)
	data := map[string]any{
		"uuid":        "abc-123",
		"name":        "Test Resource",
		"description": "A test",
		"custom":      "value",
	}

	if err := p.PrintDetail(data); err != nil {
		t.Fatalf("PrintDetail() error: %v", err)
	}

	out := buf.String()
	// Priority keys should appear
	if !strings.Contains(out, "UUID") {
		t.Error("PrintDetail missing priority key UUID")
	}
	if !strings.Contains(out, "NAME") {
		t.Error("PrintDetail missing priority key NAME")
	}
	if !strings.Contains(out, "abc-123") {
		t.Error("PrintDetail missing uuid value")
	}
	if !strings.Contains(out, "CUSTOM") {
		t.Error("PrintDetail missing non-priority key CUSTOM")
	}
}

func TestPrinter_PrintDetail_NestedData(t *testing.T) {
	p, buf := newTestPrinter(FormatTable, false)
	data := map[string]any{
		"name":   "Test",
		"nested": map[string]any{"key1": "val1"},
		"items":  []any{"a", "b", "c"},
	}

	if err := p.PrintDetail(data); err != nil {
		t.Fatalf("PrintDetail() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "NESTED") {
		t.Error("PrintDetail missing nested map section")
	}
	if !strings.Contains(out, "ITEMS") {
		t.Error("PrintDetail missing items section")
	}
	if !strings.Contains(out, "3 items") {
		t.Error("PrintDetail missing item count")
	}
}

func TestPrinter_PrintAny(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		data   any
	}{
		{"json format", FormatJSON, map[string]any{"key": "value"}},
		{"table falls back to json", FormatTable, map[string]any{"key": "value"}},
		{"yaml format", FormatYAML, map[string]any{"key": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, buf := newTestPrinter(tt.format, false)
			if err := p.PrintAny(tt.data); err != nil {
				t.Fatalf("PrintAny() error: %v", err)
			}
			if buf.Len() == 0 {
				t.Error("PrintAny() produced no output")
			}
		})
	}
}

func TestPrinter_PrintMessage(t *testing.T) {
	p, buf := newTestPrinter(FormatTable, false)
	p.PrintMessage("hello %s", "world")

	if got := buf.String(); got != "hello world\n" {
		t.Errorf("PrintMessage() = %q, want %q", got, "hello world\n")
	}
}

func TestPrinter_PrintSuccess(t *testing.T) {
	tests := []struct {
		name  string
		plain bool
		want  string
	}{
		{"with color", false, "\033[32mOperation completed\033[0m\n"},
		{"plain mode", true, "Operation completed\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, buf := newTestPrinter(FormatTable, tt.plain)
			p.PrintSuccess("Operation completed")
			if got := buf.String(); got != tt.want {
				t.Errorf("PrintSuccess() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrinter_PrintWarning(t *testing.T) {
	tests := []struct {
		name  string
		plain bool
		want  string
	}{
		{"with color", false, "\033[33mWarning: watch out\033[0m\n"},
		{"plain mode", true, "Warning: watch out\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, buf := newTestPrinter(FormatTable, tt.plain)
			p.PrintWarning("watch out")
			if got := buf.String(); got != tt.want {
				t.Errorf("PrintWarning() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrinter_PrintError(t *testing.T) {
	// PrintError writes to os.Stderr, not the writer.
	// We can only verify it doesn't panic.
	p, _ := newTestPrinter(FormatTable, false)
	p.PrintError("something went wrong: %s", "details")
	// No assertion on output since it goes to os.Stderr.
}

func TestPrinter_PrintKeyValue(t *testing.T) {
	p, buf := newTestPrinter(FormatTable, false)
	p.PrintKeyValue("Name", "Alice")

	if got := buf.String(); got != "Name: Alice\n" {
		t.Errorf("PrintKeyValue() = %q, want %q", got, "Name: Alice\n")
	}
}

func TestPrinter_PlainMode(t *testing.T) {
	// When plain=true and format=FormatPlain, Print should output JSON
	p, buf := newTestPrinter(FormatPlain, true)
	data := testData()

	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print() error: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("plain mode output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
}

func TestPrinter_EmptyData(t *testing.T) {
	p, buf := newTestPrinter(FormatTable, false)
	data := []map[string]any{}

	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print() error: %v", err)
	}

	if !strings.Contains(buf.String(), "No resources found") {
		t.Errorf("empty data should show 'No resources found', got: %q", buf.String())
	}
}

func TestPrinter_EmptyData_CSV(t *testing.T) {
	p, buf := newTestPrinter(FormatCSV, false)
	data := []map[string]any{}

	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print() error: %v", err)
	}

	// CSV with empty data should produce no output
	if buf.Len() != 0 {
		t.Errorf("CSV empty data should produce no output, got: %q", buf.String())
	}
}

func TestPrinter_EmptyData_JSON(t *testing.T) {
	p, buf := newTestPrinter(FormatJSON, false)
	data := []map[string]any{}

	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print() error: %v", err)
	}

	// JSON should output an empty array
	var result []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}
