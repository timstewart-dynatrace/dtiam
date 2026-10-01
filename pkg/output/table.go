package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// TableFormatter formats data as an ASCII table.
type TableFormatter struct {
	writer io.Writer
	plain  bool
}

// NewTableFormatter creates a new table formatter.
func NewTableFormatter(w io.Writer, plain bool) *TableFormatter {
	return &TableFormatter{
		writer: w,
		plain:  plain,
	}
}

// Format formats the data as a table.
func (f *TableFormatter) Format(data []map[string]any, columns []Column) error {
	if len(data) == 0 {
		fmt.Fprintln(f.writer, "No resources found.")
		return nil
	}

	table := tablewriter.NewWriter(f.writer)

	// Set headers
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}
	table.SetHeader(headers)

	// Configure table style
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	// Add rows
	for _, item := range data {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = extractValue(item, col)
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

// FormatSingle formats a single resource with key-value pairs.
func (f *TableFormatter) FormatSingle(data map[string]any, columns []Column) error {
	if data == nil {
		fmt.Fprintln(f.writer, "No resource found.")
		return nil
	}

	table := tablewriter.NewWriter(f.writer)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator(":")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding(" ")

	for _, col := range columns {
		value := extractValue(data, col)
		table.Append([]string{col.Header, value})
	}

	table.Render()
	return nil
}

// extractValue extracts a value from a map using a column definition.
func extractValue(data map[string]any, col Column) string {
	value := getNestedValue(data, col.Key)

	if col.Formatter != nil {
		return col.Formatter(value)
	}

	return formatValue(value)
}

// getNestedValue gets a value from a map using dot notation.
func getNestedValue(data map[string]any, key string) any {
	parts := strings.Split(key, ".")
	current := any(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			current = v[part]
		default:
			return nil
		}
	}

	return current
}

// formatValue formats a value as a string.
func formatValue(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		// Check if it's an integer
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case int, int64, int32:
		return fmt.Sprintf("%d", val)
	case []any:
		if len(val) == 0 {
			return ""
		}
		// Show first few items
		if len(val) <= 3 {
			strs := make([]string, len(val))
			for i, item := range val {
				strs[i] = fmt.Sprintf("%v", item)
			}
			return strings.Join(strs, ", ")
		}
		return fmt.Sprintf("%d items", len(val))
	case map[string]any:
		return fmt.Sprintf("{%d keys}", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}
