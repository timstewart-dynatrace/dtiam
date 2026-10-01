// Package output provides output formatting for CLI commands.
package output

import (
	"fmt"
	"strings"
)

// Format represents the output format type.
type Format string

const (
	// FormatTable is the default table format.
	FormatTable Format = "table"
	// FormatWide is an extended table format with additional columns.
	FormatWide Format = "wide"
	// FormatJSON outputs data as JSON.
	FormatJSON Format = "json"
	// FormatYAML outputs data as YAML.
	FormatYAML Format = "yaml"
	// FormatCSV outputs data as CSV.
	FormatCSV Format = "csv"
	// FormatPlain outputs raw data without formatting.
	FormatPlain Format = "plain"
)

// ParseFormat parses a string into a Format.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "table", "":
		return FormatTable, nil
	case "wide":
		return FormatWide, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	case "csv":
		return FormatCSV, nil
	case "plain":
		return FormatPlain, nil
	default:
		return "", fmt.Errorf("unknown output format: %s", s)
	}
}

// String returns the string representation of the format.
func (f Format) String() string {
	return string(f)
}

// AllFormats returns all valid format options.
func AllFormats() []string {
	return []string{"table", "wide", "json", "yaml", "csv", "plain"}
}
