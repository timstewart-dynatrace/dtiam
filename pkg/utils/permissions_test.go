package utils

import (
	"testing"
)

func TestParseStatementQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []Permission
	}{
		{
			name:  "simple allow statement",
			query: "ALLOW settings:objects:read",
			expected: []Permission{
				{Effect: "ALLOW", Action: "settings:objects:read", Description: "Read settings objects"},
			},
		},
		{
			name:  "deny statement",
			query: "DENY account:users:write",
			expected: []Permission{
				{Effect: "DENY", Action: "account:users:write", Description: "Write account users"},
			},
		},
		{
			name:  "multiple actions via semicolons",
			query: "ALLOW settings:objects:read; ALLOW settings:objects:write",
			expected: []Permission{
				{Effect: "ALLOW", Action: "settings:objects:read", Description: "Read settings objects"},
				{Effect: "ALLOW", Action: "settings:objects:write", Description: "Write settings objects"},
			},
		},
		{
			name:  "statement with conditions",
			query: "ALLOW settings:objects:read WHERE environmentId = \"env123\"",
			expected: []Permission{
				{Effect: "ALLOW", Action: "settings:objects:read", Description: "Read settings objects", Conditions: "environmentId = \"env123\""},
			},
		},
		{
			name:  "semicolon-separated statements",
			query: "ALLOW settings:objects:read; DENY account:users:write",
			expected: []Permission{
				{Effect: "ALLOW", Action: "settings:objects:read", Description: "Read settings objects"},
				{Effect: "DENY", Action: "account:users:write", Description: "Write account users"},
			},
		},
		{
			name:     "empty query",
			query:    "",
			expected: nil,
		},
		{
			name:     "invalid format - no effect",
			query:    "settings:objects:read",
			expected: nil,
		},
		{
			name:  "lowercase effect is parsed",
			query: "allow settings:objects:read",
			expected: []Permission{
				{Effect: "ALLOW", Action: "settings:objects:read", Description: "Read settings objects"},
			},
		},
		{
			name:  "unknown action gets action as description",
			query: "ALLOW custom:action:foo",
			expected: []Permission{
				{Effect: "ALLOW", Action: "custom:action:foo", Description: "custom:action:foo"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStatementQuery(tt.query)

			if len(result) != len(tt.expected) {
				t.Errorf("ParseStatementQuery() returned %d permissions, want %d", len(result), len(tt.expected))
				return
			}

			for i, perm := range result {
				exp := tt.expected[i]
				if perm.Effect != exp.Effect {
					t.Errorf("ParseStatementQuery()[%d].Effect = %v, want %v", i, perm.Effect, exp.Effect)
				}
				if perm.Action != exp.Action {
					t.Errorf("ParseStatementQuery()[%d].Action = %v, want %v", i, perm.Action, exp.Action)
				}
				if perm.Description != exp.Description {
					t.Errorf("ParseStatementQuery()[%d].Description = %v, want %v", i, perm.Description, exp.Description)
				}
				if perm.Conditions != exp.Conditions {
					t.Errorf("ParseStatementQuery()[%d].Conditions = %v, want %v", i, perm.Conditions, exp.Conditions)
				}
			}
		})
	}
}

func TestPermissionPatterns(t *testing.T) {
	// Test that known patterns are correctly described
	knownPatterns := map[string]string{
		"settings:objects:read":    "Read settings objects",
		"settings:objects:write":   "Write settings objects",
		"settings:schemas:read":    "Read settings schemas",
		"environment:roles:manage": "Manage environment roles",
		"account:users:read":       "Read account users",
		"account:users:write":      "Write account users",
		"account:groups:read":      "Read account groups",
		"account:groups:write":     "Write account groups",
		"account:policies:read":    "Read account policies",
		"account:policies:write":   "Write account policies",
	}

	for pattern, expectedDesc := range knownPatterns {
		query := "ALLOW " + pattern
		result := ParseStatementQuery(query)
		if len(result) != 1 {
			t.Errorf("ParseStatementQuery(%q) returned %d permissions, want 1", query, len(result))
			continue
		}
		if result[0].Description != expectedDesc {
			t.Errorf("Permission pattern %q description = %q, want %q", pattern, result[0].Description, expectedDesc)
		}
	}
}

func TestPermission_Fields(t *testing.T) {
	perm := Permission{
		Effect:      "ALLOW",
		Action:      "settings:objects:read",
		Description: "Read settings objects",
		Conditions:  "environmentId = \"env123\"",
	}

	if perm.Effect != "ALLOW" {
		t.Errorf("Permission.Effect = %v, want ALLOW", perm.Effect)
	}
	if perm.Action != "settings:objects:read" {
		t.Errorf("Permission.Action = %v, want settings:objects:read", perm.Action)
	}
	if perm.Description != "Read settings objects" {
		t.Errorf("Permission.Description = %v, want Read settings objects", perm.Description)
	}
	if perm.Conditions != "environmentId = \"env123\"" {
		t.Errorf("Permission.Conditions = %v, want environmentId = \"env123\"", perm.Conditions)
	}
}

func TestEffectivePermission_Sources(t *testing.T) {
	perm := EffectivePermission{
		Effect:      "ALLOW",
		Action:      "settings:objects:read",
		Description: "Read settings objects",
		Sources: []PermissionSource{
			{Group: "Admins", Policy: "Admin Policy"},
			{Group: "Editors", Policy: "Editor Policy", Boundary: "boundary-123"},
		},
	}

	if perm.Effect != "ALLOW" {
		t.Errorf("EffectivePermission.Effect = %v, want ALLOW", perm.Effect)
	}
	if perm.Action != "settings:objects:read" {
		t.Errorf("EffectivePermission.Action = %v, want settings:objects:read", perm.Action)
	}
	if perm.Description != "Read settings objects" {
		t.Errorf("EffectivePermission.Description = %v, want Read settings objects", perm.Description)
	}
	if len(perm.Sources) != 2 {
		t.Errorf("EffectivePermission.Sources len = %d, want 2", len(perm.Sources))
	}

	if perm.Sources[0].Group != "Admins" {
		t.Errorf("EffectivePermission.Sources[0].Group = %v, want Admins", perm.Sources[0].Group)
	}
	if perm.Sources[1].Boundary != "boundary-123" {
		t.Errorf("EffectivePermission.Sources[1].Boundary = %v, want boundary-123", perm.Sources[1].Boundary)
	}
}

func TestMatrixResult_Fields(t *testing.T) {
	result := MatrixResult{
		Permissions:     []string{"ALLOW:settings:read", "DENY:account:write"},
		Policies:        []string{"Policy1", "Policy2"},
		Groups:          []string{"Group1"},
		PolicyCount:     2,
		GroupCount:      1,
		PermissionCount: 2,
		Matrix:          []map[string]any{{"policy_name": "Policy1"}},
	}

	if len(result.Permissions) != 2 {
		t.Errorf("MatrixResult.Permissions len = %d, want 2", len(result.Permissions))
	}
	if len(result.Policies) != 2 {
		t.Errorf("MatrixResult.Policies len = %d, want 2", len(result.Policies))
	}
	if len(result.Groups) != 1 {
		t.Errorf("MatrixResult.Groups len = %d, want 1", len(result.Groups))
	}
	if result.PolicyCount != 2 {
		t.Errorf("MatrixResult.PolicyCount = %d, want 2", result.PolicyCount)
	}
	if result.GroupCount != 1 {
		t.Errorf("MatrixResult.GroupCount = %d, want 1", result.GroupCount)
	}
	if result.PermissionCount != 2 {
		t.Errorf("MatrixResult.PermissionCount = %d, want 2", result.PermissionCount)
	}
	if len(result.Matrix) != 1 {
		t.Errorf("MatrixResult.Matrix len = %d, want 1", len(result.Matrix))
	}
}
