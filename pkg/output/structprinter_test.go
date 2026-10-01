package output

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Test structs for struct printer

type testResource struct {
	Name        string `table:"NAME"`
	ID          string `table:"ID"`
	Status      string `table:"STATUS"`
	Description string `table:"DESCRIPTION,wide"`
	Internal    string `table:"-"`
}

type testNoTags struct {
	Name  string
	Value int
}

type testBase struct {
	BaseField string `table:"BASE"`
	Hidden    string `table:"-"`
}

type testEmbedded struct {
	testBase
	Extra string `table:"EXTRA"`
}

type testWithPointer struct {
	Name  string  `table:"NAME"`
	Value *string `table:"VALUE"`
}

type testWithTime struct {
	Name    string    `table:"NAME"`
	Created time.Time `table:"CREATED"`
}

type testWithSlice struct {
	Name  string   `table:"NAME"`
	Tags  []string `table:"TAGS"`
	Items []int    `table:"ITEMS"`
}

func TestGetTableFields_WithTags(t *testing.T) {
	fields := getTableFields(typeOf(testResource{}), false)
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields in normal mode, got %d", len(fields))
	}
	if fields[0].header != "NAME" {
		t.Errorf("field 0 header = %q, want NAME", fields[0].header)
	}
	if fields[1].header != "ID" {
		t.Errorf("field 1 header = %q, want ID", fields[1].header)
	}
	if fields[2].header != "STATUS" {
		t.Errorf("field 2 header = %q, want STATUS", fields[2].header)
	}
}

func TestGetTableFields_WideMode(t *testing.T) {
	fields := getTableFields(typeOf(testResource{}), true)
	if len(fields) != 4 {
		t.Fatalf("expected 4 fields in wide mode, got %d", len(fields))
	}
	if fields[3].header != "DESCRIPTION" {
		t.Errorf("field 3 header = %q, want DESCRIPTION", fields[3].header)
	}
	if !fields[3].wideOnly {
		t.Error("DESCRIPTION should be marked wideOnly")
	}
}

func TestGetTableFields_NoTags(t *testing.T) {
	fields := getTableFields(typeOf(testNoTags{}), false)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fallback fields, got %d", len(fields))
	}
	if fields[0].header != "NAME" {
		t.Errorf("fallback field 0 = %q, want NAME", fields[0].header)
	}
	if fields[1].header != "VALUE" {
		t.Errorf("fallback field 1 = %q, want VALUE", fields[1].header)
	}
}

func TestGetTableFields_Embedded(t *testing.T) {
	fields := getTableFields(typeOf(testEmbedded{}), false)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields (BASE + EXTRA), got %d", len(fields))
	}
	if fields[0].header != "BASE" {
		t.Errorf("embedded field 0 = %q, want BASE", fields[0].header)
	}
	if fields[1].header != "EXTRA" {
		t.Errorf("embedded field 1 = %q, want EXTRA", fields[1].header)
	}
}

func TestGetTableFields_NonStruct(t *testing.T) {
	fields := getTableFields(typeOf("not a struct"), false)
	if fields != nil {
		t.Errorf("expected nil for non-struct, got %v", fields)
	}
}

func TestStructPrintList_Table(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	data := []testResource{
		{Name: "alpha", ID: "a-1", Status: "active", Description: "first", Internal: "secret"},
		{Name: "beta", ID: "b-2", Status: "inactive", Description: "second", Internal: "hidden"},
	}

	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("StructPrintList error: %v", err)
	}

	output := buf.String()
	// Should include visible fields
	if !strings.Contains(output, "alpha") {
		t.Error("output should contain 'alpha'")
	}
	if !strings.Contains(output, "active") {
		t.Error("output should contain 'active'")
	}
	// Should NOT include hidden or wide-only fields
	if strings.Contains(output, "secret") {
		t.Error("output should NOT contain 'secret' (table:\"-\")")
	}
	if strings.Contains(output, "first") {
		t.Error("output should NOT contain 'first' (wide-only in normal mode)")
	}
}

func TestStructPrintList_Wide(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatWide, false)
	p.SetWriter(&buf)

	data := []testResource{
		{Name: "alpha", ID: "a-1", Status: "active", Description: "first", Internal: "secret"},
	}

	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("StructPrintList error: %v", err)
	}

	output := buf.String()
	// Wide mode should include DESCRIPTION
	if !strings.Contains(output, "first") {
		t.Error("wide mode should contain 'first' (DESCRIPTION)")
	}
	// Still should NOT include hidden
	if strings.Contains(output, "secret") {
		t.Error("wide mode should NOT contain 'secret' (table:\"-\")")
	}
}

func TestStructPrintList_JSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatJSON, false)
	p.SetWriter(&buf)

	data := []testResource{
		{Name: "alpha", ID: "a-1", Status: "active"},
	}

	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("StructPrintList JSON error: %v", err)
	}

	// Verify it's valid JSON
	var result []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON output not valid: %v", err)
	}
}

func TestStructPrintList_YAML(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatYAML, false)
	p.SetWriter(&buf)

	data := []testResource{
		{Name: "alpha", ID: "a-1"},
	}

	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("StructPrintList YAML error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name: alpha") {
		t.Errorf("YAML should contain field values, got %q", output)
	}
}

func TestStructPrintList_CSV(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatCSV, false)
	p.SetWriter(&buf)

	data := []testResource{
		{Name: "alpha", ID: "a-1", Status: "active"},
	}

	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("StructPrintList CSV error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("CSV expected 2 lines (header + data), got %d", len(lines))
	}
	if !strings.Contains(lines[0], "NAME") {
		t.Errorf("CSV header should contain NAME, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "alpha") {
		t.Errorf("CSV data should contain alpha, got %q", lines[1])
	}
}

func TestStructPrintList_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	data := []testResource{}

	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("StructPrintList error: %v", err)
	}

	if !strings.Contains(buf.String(), "No resources found") {
		t.Error("empty slice should print 'No resources found'")
	}
}

func TestStructPrintList_PointerSlice(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	data := []*testResource{
		{Name: "alpha", ID: "a-1", Status: "active"},
	}

	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("StructPrintList pointer slice error: %v", err)
	}

	if !strings.Contains(buf.String(), "alpha") {
		t.Error("pointer slice output should contain 'alpha'")
	}
}

func TestStructPrint_Single(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	r := testResource{Name: "alpha", ID: "a-1", Status: "active"}
	if err := p.StructPrint(r); err != nil {
		t.Fatalf("StructPrint error: %v", err)
	}

	if !strings.Contains(buf.String(), "alpha") {
		t.Error("single print should contain 'alpha'")
	}
}

func TestStructPrintList_NotSlice(t *testing.T) {
	p := NewPrinter(FormatTable, false)
	err := p.StructPrintList("not a slice")
	if err == nil {
		t.Error("expected error for non-slice input")
	}
}

func TestFormatStructValue_Pointer(t *testing.T) {
	val := "hello"
	r := testWithPointer{Name: "test", Value: &val}

	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	if err := p.StructPrintList([]testWithPointer{r}); err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Error("should contain pointer value 'hello'")
	}
}

func TestFormatStructValue_NilPointer(t *testing.T) {
	r := testWithPointer{Name: "test", Value: nil}

	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	if err := p.StructPrintList([]testWithPointer{r}); err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should not panic, nil pointer renders as empty
	if !strings.Contains(buf.String(), "test") {
		t.Error("should contain 'test'")
	}
}

func TestFormatStructValue_Time(t *testing.T) {
	ts := time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC)
	r := testWithTime{Name: "test", Created: ts}

	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	if err := p.StructPrintList([]testWithTime{r}); err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(buf.String(), "2025-03-15") {
		t.Error("should contain formatted time")
	}
}

func TestFormatStructValue_ZeroTime(t *testing.T) {
	r := testWithTime{Name: "test"}

	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	if err := p.StructPrintList([]testWithTime{r}); err != nil {
		t.Fatalf("error: %v", err)
	}
	// Zero time should render as empty, not "0001-01-01"
	if strings.Contains(buf.String(), "0001") {
		t.Error("zero time should not show '0001'")
	}
}

func TestFormatStructValue_Slices(t *testing.T) {
	tests := []struct {
		name     string
		resource testWithSlice
		contains string
	}{
		{
			"small slice shows items",
			testWithSlice{Name: "a", Tags: []string{"go", "cli"}},
			"go, cli",
		},
		{
			"large slice shows count",
			testWithSlice{Name: "b", Items: []int{1, 2, 3, 4, 5}},
			"5 items",
		},
		{
			"nil slice shows empty",
			testWithSlice{Name: "c"},
			"c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(FormatTable, false)
			p.SetWriter(&buf)

			if err := p.StructPrintList([]testWithSlice{tt.resource}); err != nil {
				t.Fatalf("error: %v", err)
			}
			if !strings.Contains(buf.String(), tt.contains) {
				t.Errorf("output should contain %q, got %q", tt.contains, buf.String())
			}
		})
	}
}

func TestStructPrintList_EmbeddedStruct(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	data := []testEmbedded{
		{testBase: testBase{BaseField: "base-val", Hidden: "hidden-val"}, Extra: "extra-val"},
	}

	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "base-val") {
		t.Error("should contain embedded field 'base-val'")
	}
	if !strings.Contains(output, "extra-val") {
		t.Error("should contain 'extra-val'")
	}
	if strings.Contains(output, "hidden-val") {
		t.Error("should NOT contain hidden embedded field 'hidden-val'")
	}
}

func TestFormatStructValue_Bool(t *testing.T) {
	type testBool struct {
		Name   string `table:"NAME"`
		Active bool   `table:"ACTIVE"`
	}

	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	data := []testBool{
		{Name: "a", Active: true},
		{Name: "b", Active: false},
	}
	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "true") {
		t.Error("should contain 'true'")
	}
	if !strings.Contains(output, "false") {
		t.Error("should contain 'false'")
	}
}

func TestFormatStructValue_Float(t *testing.T) {
	type testFloat struct {
		Name  string  `table:"NAME"`
		Score float64 `table:"SCORE"`
	}

	var buf bytes.Buffer
	p := NewPrinter(FormatTable, false)
	p.SetWriter(&buf)

	data := []testFloat{
		{Name: "int-like", Score: 42.0},
		{Name: "decimal", Score: 3.14},
	}
	if err := p.StructPrintList(data); err != nil {
		t.Fatalf("error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "42") {
		t.Error("should contain '42' for integer-like float")
	}
	if !strings.Contains(output, "3.14") {
		t.Error("should contain '3.14' for decimal float")
	}
}

// typeOf is a helper to get reflect.Type without importing reflect in test.
func typeOf(v any) reflect.Type {
	return reflect.TypeOf(v)
}
