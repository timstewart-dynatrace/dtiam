package output

import (
	"testing"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{"empty string defaults to table", "", FormatTable, false},
		{"table", "table", FormatTable, false},
		{"TABLE uppercase", "TABLE", FormatTable, false},
		{"wide", "wide", FormatWide, false},
		{"json", "json", FormatJSON, false},
		{"yaml", "yaml", FormatYAML, false},
		{"yml alias", "yml", FormatYAML, false},
		{"csv", "csv", FormatCSV, false},
		{"plain", "plain", FormatPlain, false},
		{"invalid format", "invalid", "", true},
		{"xml not supported", "xml", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFormat(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseFormat(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormat_String(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatTable, "table"},
		{FormatWide, "wide"},
		{FormatJSON, "json"},
		{FormatYAML, "yaml"},
		{FormatCSV, "csv"},
		{FormatPlain, "plain"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("Format.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllFormats(t *testing.T) {
	formats := AllFormats()

	expected := []string{"table", "wide", "json", "yaml", "csv", "plain"}

	if len(formats) != len(expected) {
		t.Errorf("AllFormats() returned %d formats, want %d", len(formats), len(expected))
	}

	// Create a map for easy lookup
	formatMap := make(map[string]bool)
	for _, f := range formats {
		formatMap[f] = true
	}

	for _, e := range expected {
		if !formatMap[e] {
			t.Errorf("AllFormats() missing expected format: %s", e)
		}
	}
}
