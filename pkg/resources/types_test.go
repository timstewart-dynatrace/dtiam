package resources

import (
	"testing"
)

func TestMapToGroup(t *testing.T) {
	m := map[string]any{
		"uuid":        "abc-123",
		"name":        "Admins",
		"description": "Admin group",
		"owner":       "system",
		"createdAt":   "2025-01-01T00:00:00Z",
	}

	g := MapToGroup(m)
	if g.UUID != "abc-123" {
		t.Errorf("UUID = %q, want 'abc-123'", g.UUID)
	}
	if g.Name != "Admins" {
		t.Errorf("Name = %q, want 'Admins'", g.Name)
	}
	if g.Description != "Admin group" {
		t.Errorf("Description = %q, want 'Admin group'", g.Description)
	}
	if g.Owner != "system" {
		t.Errorf("Owner = %q, want 'system'", g.Owner)
	}
	if g.CreatedAt != "2025-01-01T00:00:00Z" {
		t.Errorf("CreatedAt = %q, want '2025-01-01T00:00:00Z'", g.CreatedAt)
	}
}

func TestMapToGroup_MissingFields(t *testing.T) {
	g := MapToGroup(map[string]any{"name": "partial"})
	if g.UUID != "" {
		t.Errorf("UUID should be empty, got %q", g.UUID)
	}
	if g.Name != "partial" {
		t.Errorf("Name = %q, want 'partial'", g.Name)
	}
}

func TestMapToUser(t *testing.T) {
	m := map[string]any{
		"uid":        "u-1",
		"email":      "alice@example.com",
		"name":       "Alice",
		"surname":    "Smith",
		"userStatus": "ACTIVE",
		"groups":     []any{"g1", "g2", "g3"},
	}

	u := MapToUser(m)
	if u.UID != "u-1" {
		t.Errorf("UID = %q, want 'u-1'", u.UID)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("Email = %q, want 'alice@example.com'", u.Email)
	}
	if u.GroupCount != 3 {
		t.Errorf("GroupCount = %d, want 3", u.GroupCount)
	}
}

func TestMapToUser_NoGroups(t *testing.T) {
	u := MapToUser(map[string]any{"uid": "u-1"})
	if u.GroupCount != 0 {
		t.Errorf("GroupCount should be 0, got %d", u.GroupCount)
	}
}

func TestMapToPolicy(t *testing.T) {
	m := map[string]any{
		"uuid":           "p-1",
		"name":           "ReadOnly",
		"description":    "Read-only policy",
		"_level_type":    "account",
		"_level_id":      "acc-123",
		"statementQuery": "ALLOW env:read;",
	}

	p := MapToPolicy(m)
	if p.UUID != "p-1" {
		t.Errorf("UUID = %q, want 'p-1'", p.UUID)
	}
	if p.LevelType != "account" {
		t.Errorf("LevelType = %q, want 'account'", p.LevelType)
	}
	if p.StatementQuery != "ALLOW env:read;" {
		t.Errorf("StatementQuery = %q, want 'ALLOW env:read;'", p.StatementQuery)
	}
}

func TestMapToBinding(t *testing.T) {
	m := map[string]any{
		"groupUuid":  "g-1",
		"policyUuid": "p-1",
		"levelType":  "account",
		"levelId":    "acc-123",
		"boundaries": []any{"b-1", "b-2"},
	}

	b := MapToBinding(m)
	if b.GroupUUID != "g-1" {
		t.Errorf("GroupUUID = %q, want 'g-1'", b.GroupUUID)
	}
	if b.BoundaryCount != 2 {
		t.Errorf("BoundaryCount = %d, want 2", b.BoundaryCount)
	}
}

func TestMapToBinding_NoBoundaries(t *testing.T) {
	b := MapToBinding(map[string]any{"groupUuid": "g-1"})
	if b.BoundaryCount != 0 {
		t.Errorf("BoundaryCount should be 0, got %d", b.BoundaryCount)
	}
}

func TestMapToBoundary(t *testing.T) {
	m := map[string]any{
		"uuid":        "b-1",
		"name":        "Production",
		"description": "Prod boundary",
		"createdAt":   "2025-06-01T00:00:00Z",
	}

	b := MapToBoundary(m)
	if b.UUID != "b-1" {
		t.Errorf("UUID = %q, want 'b-1'", b.UUID)
	}
	if b.Name != "Production" {
		t.Errorf("Name = %q, want 'Production'", b.Name)
	}
}

func TestMapToEnvironment(t *testing.T) {
	m := map[string]any{
		"id":    "env-1",
		"name":  "Production",
		"state": "ACTIVE",
		"trial": true,
	}

	e := MapToEnvironment(m)
	if e.ID != "env-1" {
		t.Errorf("ID = %q, want 'env-1'", e.ID)
	}
	if !e.Trial {
		t.Error("Trial should be true")
	}
}

func TestMapToEnvironment_NoTrial(t *testing.T) {
	e := MapToEnvironment(map[string]any{"id": "env-1"})
	if e.Trial {
		t.Error("Trial should default to false")
	}
}

func TestMapToServiceUser(t *testing.T) {
	m := map[string]any{
		"uid":         "su-1",
		"name":        "automation",
		"description": "CI/CD service user",
		"groups":      []any{"g-1"},
	}

	su := MapToServiceUser(m)
	if su.UID != "su-1" {
		t.Errorf("UID = %q, want 'su-1'", su.UID)
	}
	if su.GroupCount != 1 {
		t.Errorf("GroupCount = %d, want 1", su.GroupCount)
	}
}

func TestMapToLimit(t *testing.T) {
	m := map[string]any{
		"name":          "groups",
		"current":       float64(5),
		"max":           float64(500),
		"usage_percent": float64(1.0),
	}

	l := MapToLimit(m)
	if l.Name != "groups" {
		t.Errorf("Name = %q, want 'groups'", l.Name)
	}
	if l.Current != 5 {
		t.Errorf("Current = %d, want 5", l.Current)
	}
	if l.Max != 500 {
		t.Errorf("Max = %d, want 500", l.Max)
	}
	if l.UsagePercent != 1.0 {
		t.Errorf("UsagePercent = %f, want 1.0", l.UsagePercent)
	}
}

func TestMapToSubscription(t *testing.T) {
	m := map[string]any{
		"uuid":      "sub-1",
		"name":      "DPS",
		"type":      "SaaS",
		"status":    "ACTIVE",
		"startTime": "2025-01-01",
		"endTime":   "2026-01-01",
	}

	s := MapToSubscription(m)
	if s.UUID != "sub-1" {
		t.Errorf("UUID = %q, want 'sub-1'", s.UUID)
	}
	if s.Type != "SaaS" {
		t.Errorf("Type = %q, want 'SaaS'", s.Type)
	}
	if s.StartTime != "2025-01-01" {
		t.Errorf("StartTime = %q, want '2025-01-01'", s.StartTime)
	}
}

func TestStringFrom_Missing(t *testing.T) {
	m := map[string]any{"name": "test"}
	if got := stringFrom(m, "nonexistent"); got != "" {
		t.Errorf("stringFrom missing key = %q, want empty", got)
	}
}

func TestStringFrom_WrongType(t *testing.T) {
	m := map[string]any{"count": 42}
	if got := stringFrom(m, "count"); got != "" {
		t.Errorf("stringFrom wrong type = %q, want empty", got)
	}
}
