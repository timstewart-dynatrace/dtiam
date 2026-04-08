package output

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// tableFieldInfo holds metadata about a struct field for table output.
type tableFieldInfo struct {
	header   string
	indices  []int // field path for embedded structs
	wideOnly bool
}

// hasAnyTableTag checks if any field (including embedded structs) has a table tag.
func hasAnyTableTag(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			if hasAnyTableTag(f.Type) {
				return true
			}
			continue
		}
		if _, ok := f.Tag.Lookup("table"); ok {
			return true
		}
	}
	return false
}

// getTableFields extracts table field metadata from a struct type.
// If no fields have a `table` tag, all exported fields are included as fallback.
// Fields with `table:"-"` are always excluded.
func getTableFields(t reflect.Type, wide bool) []tableFieldInfo {
	// Unwrap pointer
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	// First pass: check if any field (including embedded) has a table tag
	hasTableTags := hasAnyTableTag(t)

	var fields []tableFieldInfo
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// Handle embedded structs before export check (embedded types may be unexported)
		if f.Anonymous {
			embedded := getTableFields(f.Type, wide)
			for _, ef := range embedded {
				// Prepend parent index
				indices := make([]int, 0, 1+len(ef.indices))
				indices = append(indices, i)
				indices = append(indices, ef.indices...)
				ef.indices = indices
				fields = append(fields, ef)
			}
			continue
		}

		if !f.IsExported() {
			continue
		}

		tag, ok := f.Tag.Lookup("table")
		if hasTableTags {
			if !ok || tag == "-" {
				continue
			}
			parts := strings.Split(tag, ",")
			header := parts[0]
			wideOnly := len(parts) > 1 && parts[1] == "wide"
			if wideOnly && !wide {
				continue
			}
			fields = append(fields, tableFieldInfo{
				header:   header,
				indices:  []int{i},
				wideOnly: wideOnly,
			})
		} else {
			// Fallback: use field name as header
			fields = append(fields, tableFieldInfo{
				header:  strings.ToUpper(f.Name),
				indices: []int{i},
			})
		}
	}

	return fields
}

// getFieldByPath navigates nested struct fields via index path.
func getFieldByPath(v reflect.Value, indices []int) reflect.Value {
	for _, idx := range indices {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		v = v.Field(idx)
	}
	return v
}

// formatStructValue converts a reflected value to a display string.
func formatStructValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	// Handle interfaces
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if f == float64(int64(f)) {
			return fmt.Sprintf("%d", int64(f))
		}
		return fmt.Sprintf("%.2f", f)
	case reflect.Slice, reflect.Array:
		if v.IsNil() || v.Len() == 0 {
			return ""
		}
		if v.Len() <= 3 {
			strs := make([]string, v.Len())
			for i := 0; i < v.Len(); i++ {
				strs[i] = fmt.Sprintf("%v", v.Index(i).Interface())
			}
			return strings.Join(strs, ", ")
		}
		return fmt.Sprintf("%d items", v.Len())
	case reflect.Map:
		if v.IsNil() || v.Len() == 0 {
			return ""
		}
		return fmt.Sprintf("{%d keys}", v.Len())
	case reflect.Struct:
		// Handle time.Time
		if t, ok := v.Interface().(time.Time); ok {
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02 15:04:05")
		}
		return fmt.Sprintf("%v", v.Interface())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// StructPrint prints a single struct as a table row using the Printer's format.
func (p *Printer) StructPrint(v any) error {
	return p.StructPrintList([]any{v})
}

// StructPrintList prints a slice of structs as a table using `table` struct tags.
// It accepts []T, []*T, or []any where each element is the same struct type.
func (p *Printer) StructPrintList(data any) error {
	// For JSON/YAML, delegate directly — struct tags handle marshaling
	switch p.format {
	case FormatJSON, FormatPlain:
		return p.printJSON(data)
	case FormatYAML:
		return p.printYAML(data)
	}

	// Reflect into the slice
	rv := reflect.ValueOf(data)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice {
		return fmt.Errorf("StructPrintList expects a slice, got %T", data)
	}
	if rv.Len() == 0 {
		fmt.Fprintln(p.writer, "No resources found.")
		return nil
	}

	// Get the struct type from the first element
	first := rv.Index(0)
	if first.Kind() == reflect.Interface {
		first = first.Elem()
	}
	if first.Kind() == reflect.Ptr {
		first = first.Elem()
	}
	if first.Kind() != reflect.Struct {
		return fmt.Errorf("StructPrintList elements must be structs, got %s", first.Kind())
	}

	wide := p.format == FormatWide
	fields := getTableFields(first.Type(), wide)
	if len(fields) == 0 {
		return fmt.Errorf("no table fields found on %s", first.Type().Name())
	}

	// Build columns and data for the existing table formatter
	columns := make([]Column, len(fields))
	for i, f := range fields {
		columns[i] = Column{Header: f.header}
	}

	items := make([]map[string]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i)
		if elem.Kind() == reflect.Interface {
			elem = elem.Elem()
		}
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		row := make(map[string]any, len(fields))
		for j, f := range fields {
			val := getFieldByPath(elem, f.indices)
			row[columns[j].Header] = formatStructValue(val)
		}
		items[i] = row
	}

	// Use Column.Key = Header since we already formatted values
	for i := range columns {
		columns[i].Key = columns[i].Header
	}

	if p.format == FormatCSV {
		return p.printCSV(items, columns)
	}

	formatter := NewTableFormatter(p.writer, p.plain)
	return formatter.Format(items, columns)
}
