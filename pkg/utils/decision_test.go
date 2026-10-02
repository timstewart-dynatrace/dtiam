package utils

import "testing"

func TestDecidePermission(t *testing.T) {
	allow := func(conds ...any) map[string]any {
		return map[string]any{"effect": "ALLOW", "conditions": conds}
	}
	deny := func(conds ...any) map[string]any {
		return map[string]any{"effect": "DENY", "conditions": conds}
	}
	entry := func(perm string, effects ...map[string]any) map[string]any {
		es := make([]any, len(effects))
		for i, e := range effects {
			es[i] = e
		}
		return map[string]any{"permission": perm, "effects": es}
	}
	cond := map[string]any{"name": "iam:boundGroup", "operator": "EQ", "values": []any{"g1"}}

	tests := []struct {
		name  string
		perms []map[string]any
		want  string
	}{
		{name: "should say yes to an unconditional allow", perms: []map[string]any{entry("a:b:read", allow())}, want: DecisionYes},
		{name: "should say no when nothing grants it", perms: []map[string]any{entry("a:b:write", allow())}, want: DecisionNo},
		{name: "should say conditional when every allow has conditions", perms: []map[string]any{entry("a:b:read", allow(cond))}, want: DecisionConditional},
		{name: "should prefer an unconditional allow over a conditional one", perms: []map[string]any{entry("a:b:read", allow(cond), allow())}, want: DecisionYes},
		{name: "should let an unconditional deny win", perms: []map[string]any{entry("a:b:read", allow(), deny())}, want: DecisionNo},
		{name: "should not let a conditional deny override an allow", perms: []map[string]any{entry("a:b:read", allow(), deny(cond))}, want: DecisionYes},
		{name: "should say no to an empty list", perms: nil, want: DecisionNo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecidePermission(tt.perms, "a:b:read")
			if got.Decision != tt.want {
				t.Errorf("Decision = %q, want %q (%+v)", got.Decision, tt.want, got)
			}
			if tt.want == DecisionConditional && len(got.Conditions) == 0 {
				t.Error("a conditional decision must report its conditions")
			}
		})
	}
}
