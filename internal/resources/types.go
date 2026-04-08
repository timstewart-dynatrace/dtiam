// Package resources types defines typed response structs with table struct tags
// for use with the struct-tag output system. These structs parallel the
// map[string]any data returned by handlers and can be used for type-safe output.
package resources

// Group represents a Dynatrace IAM group.
type Group struct {
	UUID        string `json:"uuid" table:"UUID"`
	Name        string `json:"name" table:"NAME"`
	Description string `json:"description" table:"DESCRIPTION"`
	Owner       string `json:"owner" table:"OWNER,wide"`
	CreatedAt   string `json:"createdAt" table:"CREATED,wide"`
}

// User represents a Dynatrace IAM user.
type User struct {
	UID        string `json:"uid" table:"UID"`
	Email      string `json:"email" table:"EMAIL"`
	Name       string `json:"name" table:"NAME"`
	Surname    string `json:"surname" table:"SURNAME,wide"`
	UserStatus string `json:"userStatus" table:"STATUS"`
	Groups     []any  `json:"groups" table:"-"`
	GroupCount int    `json:"-" table:"GROUPS"`
}

// Policy represents a Dynatrace IAM policy.
type Policy struct {
	UUID           string `json:"uuid" table:"UUID"`
	Name           string `json:"name" table:"NAME"`
	Description    string `json:"description" table:"DESCRIPTION"`
	LevelType      string `json:"-" table:"LEVEL,wide"`
	LevelID        string `json:"-" table:"LEVEL_ID,wide"`
	StatementQuery string `json:"statementQuery" table:"-"`
}

// Binding represents a Dynatrace IAM policy binding.
type Binding struct {
	GroupUUID  string `json:"groupUuid" table:"GROUP_UUID"`
	PolicyUUID string `json:"policyUuid" table:"POLICY_UUID"`
	LevelType  string `json:"levelType" table:"LEVEL_TYPE"`
	LevelID    string `json:"levelId" table:"LEVEL_ID"`
	Boundaries []any  `json:"boundaries" table:"-"`
	BoundaryCount int `json:"-" table:"BOUNDARIES"`
}

// Boundary represents a Dynatrace IAM environment boundary.
type Boundary struct {
	UUID        string `json:"uuid" table:"UUID"`
	Name        string `json:"name" table:"NAME"`
	Description string `json:"description" table:"DESCRIPTION"`
	CreatedAt   string `json:"createdAt" table:"CREATED,wide"`
}

// Environment represents a Dynatrace environment.
type Environment struct {
	ID    string `json:"id" table:"ID"`
	Name  string `json:"name" table:"NAME"`
	State string `json:"state" table:"STATE"`
	Trial bool   `json:"trial" table:"TRIAL,wide"`
}

// ServiceUser represents a Dynatrace service user (OAuth client).
type ServiceUser struct {
	UID         string `json:"uid" table:"UID"`
	Name        string `json:"name" table:"NAME"`
	Description string `json:"description" table:"DESCRIPTION"`
	Groups      []any  `json:"groups" table:"-"`
	GroupCount  int    `json:"-" table:"GROUPS"`
}

// Limit represents a Dynatrace account limit.
type Limit struct {
	Name         string  `json:"name" table:"NAME"`
	Current      int     `json:"current" table:"CURRENT"`
	Max          int     `json:"max" table:"MAX"`
	UsagePercent float64 `json:"usage_percent" table:"USAGE %"`
}

// Subscription represents a Dynatrace account subscription.
type Subscription struct {
	UUID      string `json:"uuid" table:"UUID"`
	Name      string `json:"name" table:"NAME"`
	Type      string `json:"type" table:"TYPE"`
	Status    string `json:"status" table:"STATUS"`
	StartTime string `json:"startTime" table:"START,wide"`
	EndTime   string `json:"endTime" table:"END,wide"`
}

// MapToGroup converts a map[string]any to a Group.
func MapToGroup(m map[string]any) Group {
	return Group{
		UUID:        stringFrom(m, "uuid"),
		Name:        stringFrom(m, "name"),
		Description: stringFrom(m, "description"),
		Owner:       stringFrom(m, "owner"),
		CreatedAt:   stringFrom(m, "createdAt"),
	}
}

// MapToUser converts a map[string]any to a User.
func MapToUser(m map[string]any) User {
	u := User{
		UID:        stringFrom(m, "uid"),
		Email:      stringFrom(m, "email"),
		Name:       stringFrom(m, "name"),
		Surname:    stringFrom(m, "surname"),
		UserStatus: stringFrom(m, "userStatus"),
	}
	if groups, ok := m["groups"].([]any); ok {
		u.Groups = groups
		u.GroupCount = len(groups)
	}
	return u
}

// MapToPolicy converts a map[string]any to a Policy.
func MapToPolicy(m map[string]any) Policy {
	return Policy{
		UUID:           stringFrom(m, "uuid"),
		Name:           stringFrom(m, "name"),
		Description:    stringFrom(m, "description"),
		LevelType:      stringFrom(m, "_level_type"),
		LevelID:        stringFrom(m, "_level_id"),
		StatementQuery: stringFrom(m, "statementQuery"),
	}
}

// MapToBinding converts a map[string]any to a Binding.
func MapToBinding(m map[string]any) Binding {
	b := Binding{
		GroupUUID:  stringFrom(m, "groupUuid"),
		PolicyUUID: stringFrom(m, "policyUuid"),
		LevelType:  stringFrom(m, "levelType"),
		LevelID:    stringFrom(m, "levelId"),
	}
	if boundaries, ok := m["boundaries"].([]any); ok {
		b.Boundaries = boundaries
		b.BoundaryCount = len(boundaries)
	}
	return b
}

// MapToBoundary converts a map[string]any to a Boundary.
func MapToBoundary(m map[string]any) Boundary {
	return Boundary{
		UUID:        stringFrom(m, "uuid"),
		Name:        stringFrom(m, "name"),
		Description: stringFrom(m, "description"),
		CreatedAt:   stringFrom(m, "createdAt"),
	}
}

// MapToEnvironment converts a map[string]any to an Environment.
func MapToEnvironment(m map[string]any) Environment {
	e := Environment{
		ID:    stringFrom(m, "id"),
		Name:  stringFrom(m, "name"),
		State: stringFrom(m, "state"),
	}
	if trial, ok := m["trial"].(bool); ok {
		e.Trial = trial
	}
	return e
}

// MapToServiceUser converts a map[string]any to a ServiceUser.
func MapToServiceUser(m map[string]any) ServiceUser {
	su := ServiceUser{
		UID:         stringFrom(m, "uid"),
		Name:        stringFrom(m, "name"),
		Description: stringFrom(m, "description"),
	}
	if groups, ok := m["groups"].([]any); ok {
		su.Groups = groups
		su.GroupCount = len(groups)
	}
	return su
}

// MapToLimit converts a map[string]any to a Limit.
func MapToLimit(m map[string]any) Limit {
	l := Limit{
		Name: stringFrom(m, "name"),
	}
	if current, ok := m["current"].(float64); ok {
		l.Current = int(current)
	}
	if max, ok := m["max"].(float64); ok {
		l.Max = int(max)
	}
	if pct, ok := m["usage_percent"].(float64); ok {
		l.UsagePercent = pct
	}
	return l
}

// MapToSubscription converts a map[string]any to a Subscription.
func MapToSubscription(m map[string]any) Subscription {
	return Subscription{
		UUID:      stringFrom(m, "uuid"),
		Name:      stringFrom(m, "name"),
		Type:      stringFrom(m, "type"),
		Status:    stringFrom(m, "status"),
		StartTime: stringFrom(m, "startTime"),
		EndTime:   stringFrom(m, "endTime"),
	}
}

// stringFrom safely extracts a string from a map.
func stringFrom(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
