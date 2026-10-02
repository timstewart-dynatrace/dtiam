package utils

// Permission decisions returned by DecidePermission.
const (
	DecisionYes         = "yes"
	DecisionNo          = "no"
	DecisionConditional = "conditional"
)

// PermissionDecision answers "can this entity do PERMISSION".
type PermissionDecision struct {
	Permission string `json:"permission" yaml:"permission"`
	// Decision is yes, no, or conditional: allowed only where the listed
	// conditions hold (a boundary, a policy parameter, a bound group).
	Decision   string `json:"decision" yaml:"decision"`
	Conditions []any  `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	// Reason explains a "no".
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// DecidePermission evaluates a permission against an effective-permissions
// list from the resolution API.
//
// An unconditional DENY wins over everything. An unconditional ALLOW is a yes.
// ALLOWs that only apply under conditions are "conditional", with the
// conditions returned so the caller can judge whether they hold. Anything
// else -- no grant, or only conditional DENYs -- is a no.
func DecidePermission(perms []map[string]any, permission string) PermissionDecision {
	d := PermissionDecision{Permission: permission, Decision: DecisionNo}

	var allowUnconditional, denyUnconditional bool
	var allowConditions []any
	for _, entry := range perms {
		if p, _ := entry["permission"].(string); p != permission {
			continue
		}
		effects, _ := entry["effects"].([]any)
		for _, e := range effects {
			effect, _ := e.(map[string]any)
			conditions, _ := effect["conditions"].([]any)
			switch effect["effect"] {
			case "DENY":
				if len(conditions) == 0 {
					denyUnconditional = true
				}
			case "ALLOW":
				if len(conditions) == 0 {
					allowUnconditional = true
				} else {
					allowConditions = append(allowConditions, conditions...)
				}
			}
		}
	}

	switch {
	case denyUnconditional:
		d.Reason = "explicitly denied by policy"
	case allowUnconditional:
		d.Decision = DecisionYes
	case len(allowConditions) > 0:
		d.Decision = DecisionConditional
		d.Conditions = allowConditions
	default:
		d.Reason = "no policy grants it"
	}
	return d
}
