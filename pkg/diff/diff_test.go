package diff

import "testing"

func TestCompare_MissingResourceIsACreate(t *testing.T) {
	// nil live means "does not exist". Reporting that as a create rather than as
	// every field differing matters: a create is expected on first apply.
	d := Compare("Group", "New", map[string]any{"name": "New", "description": "x"}, nil)
	if d.Change != ChangeCreate {
		t.Errorf("Change = %q, want create", d.Change)
	}
	if len(d.Fields) != 0 {
		t.Errorf("a create should not enumerate fields, got %v", d.Fields)
	}
	if !d.HasChanges() {
		t.Error("HasChanges() = false for a create")
	}
}

func TestCompare_IdenticalIsNoChange(t *testing.T) {
	spec := map[string]any{"name": "A", "description": "d"}
	live := map[string]any{"name": "A", "description": "d", "uuid": "abc"}

	d := Compare("Group", "A", spec, live)
	if d.Change != ChangeNone {
		t.Errorf("Change = %q, want none (fields: %v)", d.Change, d.Fields)
	}
	if d.HasChanges() {
		t.Error("HasChanges() = true for an identical resource")
	}
}

func TestCompare_IgnoresServerManagedFields(t *testing.T) {
	// The live resource carries fields a spec will never mention. Reporting them
	// would bury the real changes.
	spec := map[string]any{"name": "A"}
	live := map[string]any{
		"name": "A", "uuid": "abc", "createdAt": "2020-01-01", "owner": "LOCAL", "hidden": false,
	}

	d := Compare("Group", "A", spec, live)
	if d.Change != ChangeNone {
		t.Errorf("Change = %q, want none; server fields leaked in as %v", d.Change, d.Fields)
	}
}

func TestCompare_DetectsFieldDifference(t *testing.T) {
	spec := map[string]any{"name": "A", "description": "new"}
	live := map[string]any{"name": "A", "description": "old"}

	d := Compare("Group", "A", spec, live)
	if d.Change != ChangeUpdate {
		t.Fatalf("Change = %q, want update", d.Change)
	}
	if len(d.Fields) != 1 {
		t.Fatalf("got %d field changes, want 1: %v", len(d.Fields), d.Fields)
	}
	f := d.Fields[0]
	if f.Field != "description" || f.Want != "new" || f.Have != "old" {
		t.Errorf("got %+v", f)
	}
}

func TestCompare_FieldAbsentFromLiveIsAChange(t *testing.T) {
	spec := map[string]any{"description": "set it"}
	live := map[string]any{"name": "A"}

	d := Compare("Group", "A", spec, live)
	if d.Change != ChangeUpdate {
		t.Fatalf("Change = %q, want update", d.Change)
	}
	if d.Fields[0].Have != "" {
		t.Errorf("Have = %q, want empty for an absent field", d.Fields[0].Have)
	}
}

func TestCompare_NumericTypesAcrossSources(t *testing.T) {
	// A YAML file decodes 5 as int; the API returns it as float64. Treating
	// those as different would make every numeric field look modified.
	spec := map[string]any{"limit": 5}
	live := map[string]any{"limit": float64(5)}

	if d := Compare("X", "x", spec, live); d.Change != ChangeNone {
		t.Errorf("Change = %q, want none; int 5 and float64 5 must compare equal (%v)", d.Change, d.Fields)
	}
}

func TestCompare_ListOrderIsIgnored(t *testing.T) {
	// The API returns members, scopes and zones in an order the caller does not
	// control, so comparing order would report spurious changes.
	spec := map[string]any{"zones": []any{"a", "b", "c"}}
	live := map[string]any{"zones": []any{"c", "a", "b"}}

	if d := Compare("Boundary", "b", spec, live); d.Change != ChangeNone {
		t.Errorf("Change = %q, want none; list order must not matter (%v)", d.Change, d.Fields)
	}
}

func TestCompare_ListContentDifferenceIsDetected(t *testing.T) {
	spec := map[string]any{"zones": []any{"a", "b"}}
	live := map[string]any{"zones": []any{"a", "c"}}

	if d := Compare("Boundary", "b", spec, live); d.Change != ChangeUpdate {
		t.Errorf("Change = %q, want update for genuinely different list contents", d.Change)
	}
}

func TestCompare_FieldsAreReportedInStableOrder(t *testing.T) {
	// Go map iteration is random; output must be reproducible so a diff can be
	// compared across runs.
	spec := map[string]any{"z": "1", "a": "1", "m": "1"}
	live := map[string]any{"z": "2", "a": "2", "m": "2"}

	first := Compare("X", "x", spec, live)
	for i := 0; i < 20; i++ {
		again := Compare("X", "x", spec, live)
		for j := range first.Fields {
			if first.Fields[j].Field != again.Fields[j].Field {
				t.Fatalf("field order is not stable: %v vs %v", first.Fields, again.Fields)
			}
		}
	}
	if first.Fields[0].Field != "a" {
		t.Errorf("fields are not sorted: %v", first.Fields)
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{name: "nil renders empty", input: nil, want: ""},
		{name: "string passes through", input: "x", want: "x"},
		{name: "bool", input: true, want: "true"},
		{name: "whole float has no decimal point", input: float64(5), want: "5"},
		{name: "fractional float keeps precision", input: 5.5, want: "5.5"},
		{name: "int", input: 7, want: "7"},
		{name: "list is sorted", input: []any{"b", "a"}, want: "[a, b]"},
		{name: "string slice is sorted", input: []string{"b", "a"}, want: "[a, b]"},
		{name: "map keys are sorted", input: map[string]any{"b": "2", "a": "1"}, want: "{a=1, b=2}"},
		{name: "nested list in map", input: map[string]any{"k": []any{"z", "y"}}, want: "{k=[y, z]}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatValue(tt.input); got != tt.want {
				t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	diffs := []ResourceDiff{
		{Change: ChangeCreate},
		{Change: ChangeCreate},
		{Change: ChangeUpdate},
		{Change: ChangeNone},
	}
	s := Summarize(diffs)
	if s.Create != 2 || s.Update != 1 || s.None != 1 {
		t.Errorf("got %+v", s)
	}
	if !s.HasChanges() {
		t.Error("HasChanges() = false despite creates and updates")
	}
	if s.String() != "2 to create, 1 to update, 1 unchanged" {
		t.Errorf("String() = %q", s.String())
	}
}

func TestSummarize_NoChanges(t *testing.T) {
	s := Summarize([]ResourceDiff{{Change: ChangeNone}, {Change: ChangeNone}})
	if s.HasChanges() {
		t.Error("HasChanges() = true when everything is unchanged")
	}
}

func TestSummarize_Empty(t *testing.T) {
	s := Summarize(nil)
	if s.HasChanges() {
		t.Error("HasChanges() = true for no diffs")
	}
}
