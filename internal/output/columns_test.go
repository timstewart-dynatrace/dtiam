package output

import (
	"testing"
)

func TestGroupColumns(t *testing.T) {
	cols := GroupColumns()
	assertColumnHeaders(t, cols, []string{"UUID", "NAME", "DESCRIPTION", "OWNER", "CREATED"})
	assertColumnKeys(t, cols, []string{"uuid", "name", "description", "owner", "createdAt"})
}

func TestUserColumns(t *testing.T) {
	cols := UserColumns()
	assertColumnHeaders(t, cols, []string{"UID", "EMAIL", "NAME", "SURNAME", "STATUS", "GROUPS"})
	assertColumnKeys(t, cols, []string{"uid", "email", "name", "surname", "userStatus", "groups"})
}

func TestPolicyColumns(t *testing.T) {
	cols := PolicyColumns()
	assertColumnHeaders(t, cols, []string{"UUID", "NAME", "DESCRIPTION", "LEVEL", "LEVEL_ID"})
}

func TestBindingColumns(t *testing.T) {
	cols := BindingColumns()
	assertColumnHeaders(t, cols, []string{"GROUP_UUID", "POLICY_UUID", "LEVEL_TYPE", "LEVEL_ID", "BOUNDARIES"})
}

func TestBoundaryColumns(t *testing.T) {
	cols := BoundaryColumns()
	assertColumnHeaders(t, cols, []string{"UUID", "NAME", "DESCRIPTION", "CREATED"})
}

func TestEnvironmentColumns(t *testing.T) {
	cols := EnvironmentColumns()
	assertColumnHeaders(t, cols, []string{"ID", "NAME", "STATE", "TRIAL"})
}

func TestServiceUserColumns(t *testing.T) {
	cols := ServiceUserColumns()
	assertColumnHeaders(t, cols, []string{"UID", "NAME", "DESCRIPTION", "GROUPS"})
}

func TestLimitColumns(t *testing.T) {
	cols := LimitColumns()
	assertColumnHeaders(t, cols, []string{"NAME", "CURRENT", "MAX", "USAGE %"})
}

func TestSubscriptionColumns(t *testing.T) {
	cols := SubscriptionColumns()
	assertColumnHeaders(t, cols, []string{"UUID", "NAME", "TYPE", "STATUS", "START", "END"})
}

func TestTokenColumns(t *testing.T) {
	cols := TokenColumns()
	assertColumnHeaders(t, cols, []string{"ID", "NAME", "EXPIRES", "SCOPES", "CREATED"})
}

func TestAppColumns(t *testing.T) {
	cols := AppColumns()
	assertColumnHeaders(t, cols, []string{"ID", "NAME", "VERSION", "DESCRIPTION"})
}

func TestSchemaColumns(t *testing.T) {
	cols := SchemaColumns()
	assertColumnHeaders(t, cols, []string{"SCHEMA ID", "DISPLAY NAME", "VERSION"})
}

func TestFilterColumns(t *testing.T) {
	cols := []Column{
		{Key: "id", Header: "ID"},
		{Key: "name", Header: "NAME"},
		{Key: "detail", Header: "DETAIL", WideOnly: true},
		{Key: "extra", Header: "EXTRA", WideOnly: true},
	}

	t.Run("normal mode excludes wide-only", func(t *testing.T) {
		filtered := FilterColumns(cols, false)
		if len(filtered) != 2 {
			t.Fatalf("expected 2 columns, got %d", len(filtered))
		}
		for _, c := range filtered {
			if c.WideOnly {
				t.Errorf("wide-only column %q should not appear in normal mode", c.Header)
			}
		}
	})

	t.Run("wide mode includes all columns", func(t *testing.T) {
		filtered := FilterColumns(cols, true)
		if len(filtered) != 4 {
			t.Fatalf("expected 4 columns in wide mode, got %d", len(filtered))
		}
	})

	t.Run("no wide-only columns returns all", func(t *testing.T) {
		noCols := []Column{
			{Key: "a", Header: "A"},
			{Key: "b", Header: "B"},
		}
		filtered := FilterColumns(noCols, false)
		if len(filtered) != 2 {
			t.Fatalf("expected 2 columns, got %d", len(filtered))
		}
	})
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"[]any with items", []any{"a", "b", "c"}, "3"},
		{"[]any empty", []any{}, "0"},
		{"int", 5, "5"},
		{"float64", float64(10), "10"},
		{"nil", nil, "0"},
		{"string fallback", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCount(tt.input)
			if got != tt.want {
				t.Errorf("formatCount(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"float64", float64(75.5), "75.5%"},
		{"float64 zero", float64(0), "0.0%"},
		{"int", 100, "100%"},
		{"nil", nil, "N/A"},
		{"string fallback", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPercent(tt.input)
			if got != tt.want {
				t.Errorf("formatPercent(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatList(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"[]any", []any{"read", "write", "admin"}, "read, write, admin"},
		{"[]any empty", []any{}, ""},
		{"[]string", []string{"a", "b"}, "a, b"},
		{"nil", nil, ""},
		{"other type", 42, "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatList(tt.input)
			if got != tt.want {
				t.Errorf("FormatList(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// assertColumnHeaders checks that columns have the expected headers in order.
func assertColumnHeaders(t *testing.T, cols []Column, expected []string) {
	t.Helper()
	if len(cols) != len(expected) {
		t.Fatalf("expected %d columns, got %d", len(expected), len(cols))
	}
	for i, want := range expected {
		if cols[i].Header != want {
			t.Errorf("column[%d].Header = %q, want %q", i, cols[i].Header, want)
		}
	}
}

// assertColumnKeys checks that columns have the expected keys in order.
func assertColumnKeys(t *testing.T, cols []Column, expected []string) {
	t.Helper()
	if len(cols) != len(expected) {
		t.Fatalf("expected %d columns, got %d", len(expected), len(cols))
	}
	for i, want := range expected {
		if cols[i].Key != want {
			t.Errorf("column[%d].Key = %q, want %q", i, cols[i].Key, want)
		}
	}
}
