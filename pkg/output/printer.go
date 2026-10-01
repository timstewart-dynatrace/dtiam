package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Printer handles all output formatting.
type Printer struct {
	format Format
	plain  bool
	writer io.Writer
}

// NewPrinter creates a new printer with the specified format.
func NewPrinter(format Format, plain bool) *Printer {
	return &Printer{
		format: format,
		plain:  plain,
		writer: os.Stdout,
	}
}

// SetWriter sets the output writer.
func (p *Printer) SetWriter(w io.Writer) {
	p.writer = w
}

// Print prints data using the configured format.
func (p *Printer) Print(data any, columns []Column) error {
	switch p.format {
	case FormatJSON, FormatPlain:
		return p.printJSON(data)
	case FormatYAML:
		return p.printYAML(data)
	case FormatCSV:
		return p.printCSV(data, columns)
	case FormatTable, FormatWide:
		return p.printTable(data, columns)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintSingle prints a single resource.
func (p *Printer) PrintSingle(data map[string]any, columns []Column) error {
	switch p.format {
	case FormatJSON, FormatPlain:
		return p.printJSON(data)
	case FormatYAML:
		return p.printYAML(data)
	case FormatCSV:
		return p.printCSV([]map[string]any{data}, columns)
	case FormatTable, FormatWide:
		formatter := NewTableFormatter(p.writer, p.plain)
		return formatter.FormatSingle(data, FilterColumns(columns, p.format == FormatWide))
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// printJSON prints data as JSON.
func (p *Printer) printJSON(data any) error {
	encoder := json.NewEncoder(p.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printYAML prints data as YAML.
func (p *Printer) printYAML(data any) error {
	encoder := yaml.NewEncoder(p.writer)
	encoder.SetIndent(2)
	return encoder.Encode(data)
}

// printCSV prints data as CSV.
func (p *Printer) printCSV(data any, columns []Column) error {
	// Convert data to slice of maps
	items, err := toSliceOfMaps(data)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	writer := csv.NewWriter(p.writer)
	defer writer.Flush()

	// Use all columns for CSV (including wide-only)
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data rows
	for _, item := range items {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = extractValue(item, col)
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// printTable prints data as an ASCII table.
func (p *Printer) printTable(data any, columns []Column) error {
	items, err := toSliceOfMaps(data)
	if err != nil {
		return err
	}

	formatter := NewTableFormatter(p.writer, p.plain)
	filteredColumns := FilterColumns(columns, p.format == FormatWide)
	return formatter.Format(items, filteredColumns)
}

// toSliceOfMaps converts various data types to []map[string]any.
func toSliceOfMaps(data any) ([]map[string]any, error) {
	switch v := data.(type) {
	case []map[string]any:
		return v, nil
	case map[string]any:
		return []map[string]any{v}, nil
	case []any:
		result := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				result = append(result, m)
			}
		}
		return result, nil
	case nil:
		return []map[string]any{}, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to []map[string]any", data)
	}
}

// PrintMessage prints a message to the writer.
func (p *Printer) PrintMessage(format string, args ...any) {
	fmt.Fprintf(p.writer, format+"\n", args...)
}

// PrintError prints an error message to stderr.
func (p *Printer) PrintError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// PrintSuccess prints a success message.
func (p *Printer) PrintSuccess(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if p.plain {
		fmt.Fprintln(p.writer, msg)
	} else {
		fmt.Fprintln(p.writer, "\033[32m"+msg+"\033[0m") // Green
	}
}

// PrintWarning prints a warning message.
func (p *Printer) PrintWarning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if p.plain {
		fmt.Fprintln(p.writer, "Warning: "+msg)
	} else {
		fmt.Fprintln(p.writer, "\033[33mWarning: "+msg+"\033[0m") // Yellow
	}
}

// PrintKeyValue prints a key-value pair.
func (p *Printer) PrintKeyValue(key, value string) {
	fmt.Fprintf(p.writer, "%s: %s\n", key, value)
}

// PrintList prints a list of items.
func (p *Printer) PrintList(items []string) {
	for _, item := range items {
		fmt.Fprintf(p.writer, "  - %s\n", item)
	}
}

// PrintDetail prints detailed information about a resource.
func (p *Printer) PrintDetail(data map[string]any) error {
	// Priority keys to show first
	priorityKeys := []string{"uuid", "uid", "id", "name", "email", "description"}

	// Print priority keys first
	for _, key := range priorityKeys {
		if val, ok := data[key]; ok {
			p.PrintKeyValue(strings.ToUpper(key), formatValue(val))
		}
	}

	// Print remaining keys
	for key, val := range data {
		// Skip priority keys (already printed)
		skip := false
		for _, pk := range priorityKeys {
			if key == pk {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Handle nested structures
		switch v := val.(type) {
		case map[string]any:
			fmt.Fprintf(p.writer, "\n%s:\n", strings.ToUpper(key))
			for k, vv := range v {
				fmt.Fprintf(p.writer, "  %s: %s\n", k, formatValue(vv))
			}
		case []any:
			fmt.Fprintf(p.writer, "\n%s: (%d items)\n", strings.ToUpper(key), len(v))
			for i, item := range v {
				if i >= 10 {
					fmt.Fprintf(p.writer, "  ... and %d more\n", len(v)-10)
					break
				}
				fmt.Fprintf(p.writer, "  - %s\n", formatValue(item))
			}
		default:
			p.PrintKeyValue(strings.ToUpper(key), formatValue(val))
		}
	}

	return nil
}

// PrintAny prints any data structure as JSON or YAML based on format.
func (p *Printer) PrintAny(data any) error {
	switch p.format {
	case FormatJSON, FormatPlain, FormatTable, FormatWide, FormatCSV:
		return p.printJSON(data)
	case FormatYAML:
		return p.printYAML(data)
	default:
		return p.printJSON(data)
	}
}
