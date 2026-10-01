// Package diff compares a desired resource spec against the live resource.
package diff

import (
	"fmt"
	"sort"
	"strings"
)

// ChangeKind classifies what would happen to one field or resource.
type ChangeKind string

const (
	// ChangeCreate means the resource does not exist yet.
	ChangeCreate ChangeKind = "create"

	// ChangeUpdate means the resource exists and at least one field differs.
	ChangeUpdate ChangeKind = "update"

	// ChangeNone means the resource exists and matches the spec.
	ChangeNone ChangeKind = "none"
)

// FieldChange is a single field that differs between the spec and the live
// resource.
type FieldChange struct {
	Field string `json:"field" yaml:"field"`
	Want  string `json:"want" yaml:"want"`
	Have  string `json:"have" yaml:"have"`
}

// ResourceDiff is the comparison result for one resource.
type ResourceDiff struct {
	Kind    string        `json:"kind" yaml:"kind"`
	Name    string        `json:"name" yaml:"name"`
	Change  ChangeKind    `json:"change" yaml:"change"`
	Fields  []FieldChange `json:"fields,omitempty" yaml:"fields,omitempty"`
	Message string        `json:"message,omitempty" yaml:"message,omitempty"`
}

// HasChanges reports whether applying this resource would change anything.
func (d ResourceDiff) HasChanges() bool {
	return d.Change == ChangeCreate || d.Change == ChangeUpdate
}

// Compare diffs a desired spec against the live resource.
//
// live being nil means the resource does not exist, which is reported as a
// create rather than as every field differing — the distinction matters because
// a create is expected on first apply while an update is not.
//
// Only fields present in the spec are compared. The live resource carries
// server-managed fields the spec will never mention (uuid, createdAt, owner),
// and reporting those as differences would drown the real changes.
func Compare(kind, name string, spec, live map[string]any) ResourceDiff {
	d := ResourceDiff{Kind: kind, Name: name}

	if live == nil {
		d.Change = ChangeCreate
		d.Message = "does not exist; would be created"
		return d
	}

	for _, field := range sortedKeys(spec) {
		want := spec[field]

		have, present := live[field]
		if !present {
			// A field the server does not return at all is reported with an
			// empty "have" rather than skipped, since applying it would still
			// be a change.
			d.Fields = append(d.Fields, FieldChange{
				Field: field,
				Want:  formatValue(want),
				Have:  "",
			})
			continue
		}

		if !valuesEqual(want, have) {
			d.Fields = append(d.Fields, FieldChange{
				Field: field,
				Want:  formatValue(want),
				Have:  formatValue(have),
			})
		}
	}

	if len(d.Fields) == 0 {
		d.Change = ChangeNone
		d.Message = "up to date"
		return d
	}

	d.Change = ChangeUpdate
	d.Message = fmt.Sprintf("%d field(s) differ", len(d.Fields))
	return d
}

// valuesEqual compares two decoded JSON/YAML values.
//
// It compares formatted representations rather than using reflect.DeepEqual,
// because the same logical value arrives with different Go types depending on
// its source: a YAML file yields int, the API yields float64, and DeepEqual
// would call 5 and 5.0 different. Treating those as a change would make every
// numeric field look modified on every run.
func valuesEqual(a, b any) bool {
	return formatValue(a) == formatValue(b)
}

// formatValue renders a value for display and comparison.
func formatValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case bool:
		return fmt.Sprintf("%t", val)
	case float64:
		// Render a whole float without a decimal part so it matches the int form.
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case float32:
		return formatValue(float64(val))
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case []any:
		// Lists are compared order-insensitively: the API returns group members,
		// scopes and zones in an order the caller does not control, so comparing
		// order would report spurious changes.
		parts := make([]string, 0, len(val))
		for _, item := range val {
			parts = append(parts, formatValue(item))
		}
		sort.Strings(parts)
		return "[" + strings.Join(parts, ", ") + "]"
	case []string:
		parts := append([]string(nil), val...)
		sort.Strings(parts)
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		parts := make([]string, 0, len(val))
		for _, k := range sortedKeys(val) {
			parts = append(parts, k+"="+formatValue(val[k]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// sortedKeys returns a map's keys in sorted order, so output is reproducible.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Summary counts the diffs by change kind.
type Summary struct {
	Create int `json:"create" yaml:"create"`
	Update int `json:"update" yaml:"update"`
	None   int `json:"none" yaml:"none"`
}

// Summarize counts a set of diffs.
func Summarize(diffs []ResourceDiff) Summary {
	var s Summary
	for _, d := range diffs {
		switch d.Change {
		case ChangeCreate:
			s.Create++
		case ChangeUpdate:
			s.Update++
		case ChangeNone:
			s.None++
		}
	}
	return s
}

// HasChanges reports whether any resource would change.
func (s Summary) HasChanges() bool {
	return s.Create > 0 || s.Update > 0
}

// String renders the summary as a one-line description.
func (s Summary) String() string {
	return fmt.Sprintf("%d to create, %d to update, %d unchanged", s.Create, s.Update, s.None)
}
