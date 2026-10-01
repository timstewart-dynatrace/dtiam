package account

import (
	"reflect"
	"testing"
)

func TestUpperAll(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{name: "should return nil for nil", input: nil, want: nil},
		{name: "should return nil for empty", input: []string{}, want: nil},
		{name: "should upper-case values", input: []string{"budget", "cost"}, want: []string{"BUDGET", "COST"}},
		{name: "should trim surrounding space", input: []string{" warn "}, want: []string{"WARN"}},
		{name: "should leave upper-case values alone", input: []string{"SEVERE"}, want: []string{"SEVERE"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := upperAll(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("upperAll(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNotificationsCommandMetadata(t *testing.T) {
	if notificationsCmd.Short == "" || notificationsCmd.Long == "" || notificationsCmd.Example == "" {
		t.Error("Short, Long and Example are all required")
	}
	if notificationsCmd.RunE == nil {
		t.Error("commands must use RunE")
	}
}

func TestEnvironmentUsageAndCostRequireWindowFlags(t *testing.T) {
	// --start and --end are mandatory server-side, so cobra must enforce them
	// rather than letting the request fail with an opaque API error.
	for _, cmd := range []struct {
		name  string
		flags []string
	}{
		{name: "environment-usage", flags: []string{"start", "end"}},
		{name: "environment-cost", flags: []string{"start", "end"}},
	} {
		t.Run(cmd.name, func(t *testing.T) {
			target := environmentUsageCmd
			if cmd.name == "environment-cost" {
				target = environmentCostCmd
			}
			for _, name := range cmd.flags {
				flag := target.Flags().Lookup(name)
				if flag == nil {
					t.Fatalf("flag --%s is not defined", name)
				}
				if flag.Annotations == nil || len(flag.Annotations["cobra_annotation_bash_completion_one_required_flag"]) == 0 {
					t.Errorf("flag --%s is not marked required", name)
				}
			}
		})
	}
}
