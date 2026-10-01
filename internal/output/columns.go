package output

import (
	"fmt"
	"strings"
)

// Column defines a table column.
type Column struct {
	// Key is the field key to extract from data (supports dot notation).
	Key string
	// Header is the column header text.
	Header string
	// WideOnly indicates this column should only show in wide mode.
	WideOnly bool
	// Formatter is an optional custom formatter for the value.
	Formatter func(any) string
}

// GroupColumns returns columns for group resources.
func GroupColumns() []Column {
	return []Column{
		{Key: "uuid", Header: "UUID"},
		{Key: "name", Header: "NAME"},
		{Key: "description", Header: "DESCRIPTION"},
		{Key: "owner", Header: "OWNER", WideOnly: true},
		{Key: "createdAt", Header: "CREATED", WideOnly: true},
	}
}

// UserColumns returns columns for user resources.
func UserColumns() []Column {
	return []Column{
		{Key: "uid", Header: "UID"},
		{Key: "email", Header: "EMAIL"},
		{Key: "name", Header: "NAME"},
		{Key: "surname", Header: "SURNAME", WideOnly: true},
		{Key: "userStatus", Header: "STATUS"},
		{Key: "groups", Header: "GROUPS", Formatter: formatCount},
	}
}

// PolicyColumns returns columns for policy resources.
func PolicyColumns() []Column {
	return []Column{
		{Key: "uuid", Header: "UUID"},
		{Key: "name", Header: "NAME"},
		{Key: "description", Header: "DESCRIPTION"},
		{Key: "_level_type", Header: "LEVEL", WideOnly: true},
		{Key: "_level_id", Header: "LEVEL_ID", WideOnly: true},
	}
}

// BindingColumns returns columns for binding resources.
func BindingColumns() []Column {
	return []Column{
		{Key: "groupUuid", Header: "GROUP_UUID"},
		{Key: "policyUuid", Header: "POLICY_UUID"},
		{Key: "levelType", Header: "LEVEL_TYPE"},
		{Key: "levelId", Header: "LEVEL_ID"},
		{Key: "boundaries", Header: "BOUNDARIES", Formatter: formatCount},
	}
}

// BoundaryColumns returns columns for boundary resources.
func BoundaryColumns() []Column {
	return []Column{
		{Key: "uuid", Header: "UUID"},
		{Key: "name", Header: "NAME"},
		{Key: "description", Header: "DESCRIPTION"},
		{Key: "createdAt", Header: "CREATED", WideOnly: true},
	}
}

// EnvironmentColumns returns columns for environment resources.
//
// Field names verified live: the response carries active and url. There are no
// state or trial fields, so those columns always rendered blank.
func EnvironmentColumns() []Column {
	return []Column{
		{Key: "id", Header: "ID"},
		{Key: "name", Header: "NAME"},
		{Key: "active", Header: "ACTIVE"},
		{Key: "url", Header: "URL", WideOnly: true},
	}
}

// ServiceUserColumns returns columns for service user resources.
func ServiceUserColumns() []Column {
	return []Column{
		{Key: "uid", Header: "UID"},
		{Key: "name", Header: "NAME"},
		{Key: "description", Header: "DESCRIPTION"},
		{Key: "groups", Header: "GROUPS", Formatter: formatCount},
	}
}

// LimitColumns returns columns for account limit resources.
//
// Field names verified live: limitType, currentValue, limitValue. The earlier
// name/current/max keys matched nothing, so every column rendered blank.
// usage_percent is computed by the handler, not returned by the API.
func LimitColumns() []Column {
	return []Column{
		{Key: "limitType", Header: "LIMIT"},
		{Key: "currentValue", Header: "CURRENT"},
		{Key: "limitValue", Header: "MAX"},
		{Key: "usage_percent", Header: "USAGE %", Formatter: formatPercent},
	}
}

// SubscriptionColumns returns columns for subscription resources.
func SubscriptionColumns() []Column {
	return []Column{
		{Key: "uuid", Header: "UUID"},
		{Key: "name", Header: "NAME"},
		{Key: "type", Header: "TYPE"},
		{Key: "status", Header: "STATUS"},
		{Key: "startTime", Header: "START", WideOnly: true},
		{Key: "endTime", Header: "END", WideOnly: true},
	}
}

// TokenColumns returns columns for platform token resources.
//
// Field names verified against a live account: the response carries tokenId,
// expirationDate, and scope (singular). The earlier id/expiresIn/scopes keys
// matched nothing, so those columns always rendered blank.
func TokenColumns() []Column {
	return []Column{
		{Key: "tokenId", Header: "TOKEN ID"},
		{Key: "name", Header: "NAME"},
		{Key: "status", Header: "STATUS"},
		{Key: "expirationDate", Header: "EXPIRES"},
		{Key: "owner", Header: "OWNER", WideOnly: true},
		{Key: "createdBy", Header: "CREATED BY", WideOnly: true},
		{Key: "createdAt", Header: "CREATED", WideOnly: true},
		{Key: "scope", Header: "SCOPES", WideOnly: true, Formatter: FormatList},
	}
}

// AppColumns returns columns for app resources.
func AppColumns() []Column {
	return []Column{
		{Key: "id", Header: "ID"},
		{Key: "name", Header: "NAME"},
		{Key: "version", Header: "VERSION"},
		{Key: "description", Header: "DESCRIPTION", WideOnly: true},
	}
}

// SchemaColumns returns columns for schema resources.
func SchemaColumns() []Column {
	return []Column{
		{Key: "schemaId", Header: "SCHEMA ID"},
		{Key: "displayName", Header: "DISPLAY NAME"},
		{Key: "latestSchemaVersion", Header: "VERSION", WideOnly: true},
	}
}

// CapabilityColumns returns columns for subscription capability resources.
func CapabilityColumns() []Column {
	return []Column{
		{Key: "key", Header: "KEY"},
		{Key: "enabled", Header: "ENABLED"},
		{Key: "subscription", Header: "SUBSCRIPTION", WideOnly: true},
	}
}

// AuditColumns returns columns for account audit log entries.
func AuditColumns() []Column {
	// The default projection returns only timestamp, eventType, user, resource,
	// resourceName, eventProvider and eventId -- verified live. Everything else
	// (eventOutcome, resourceId, originAddress, authenticationType, eventReason)
	// requires --add-fields, so those stay in the wide set rather than
	// rendering as blank columns by default.
	return []Column{
		{Key: "timestamp", Header: "TIMESTAMP"},
		{Key: "eventType", Header: "EVENT"},
		{Key: "user", Header: "USER"},
		{Key: "resource", Header: "TYPE"},
		{Key: "resourceName", Header: "RESOURCE"},
		{Key: "eventProvider", Header: "PROVIDER", WideOnly: true},
		{Key: "eventId", Header: "EVENT ID", WideOnly: true},
		{Key: "eventOutcome", Header: "OUTCOME", WideOnly: true},
		{Key: "eventReason", Header: "REASON", WideOnly: true},
		{Key: "resourceId", Header: "RESOURCE ID", WideOnly: true},
		{Key: "originAddress", Header: "ORIGIN IP", WideOnly: true},
		{Key: "authenticationType", Header: "AUTH TYPE", WideOnly: true},
	}
}

// ReferencePermissionColumns returns columns for the reference data permission
// list: the permissions an account can grant.
func ReferencePermissionColumns() []Column {
	return []Column{
		{Key: "id", Header: "ID"},
		{Key: "description", Header: "DESCRIPTION"},
	}
}

// GroupPermissionColumns returns columns for direct permission grants on a group.
func GroupPermissionColumns() []Column {
	return []Column{
		{Key: "permissionName", Header: "PERMISSION"},
		{Key: "scopeType", Header: "SCOPE TYPE"},
		{Key: "scope", Header: "SCOPE"},
		{Key: "createdAt", Header: "CREATED", WideOnly: true},
		{Key: "updatedAt", Header: "UPDATED", WideOnly: true},
	}
}

// NotificationColumns returns columns for account notifications.
func NotificationColumns() []Column {
	return []Column{
		{Key: "dateTime", Header: "TIMESTAMP"},
		{Key: "type", Header: "TYPE"},
		{Key: "severity", Header: "SEVERITY"},
		{Key: "message", Header: "MESSAGE"},
		{Key: "subscriptionName", Header: "SUBSCRIPTION", WideOnly: true},
		{Key: "id", Header: "ID", WideOnly: true},
	}
}

// EnvironmentUsageColumns returns columns for per-environment subscription usage.
func EnvironmentUsageColumns() []Column {
	return []Column{
		{Key: "environmentId", Header: "ENVIRONMENT"},
		{Key: "capabilityKey", Header: "CAPABILITY"},
		{Key: "usage", Header: "USAGE"},
		{Key: "unit", Header: "UNIT"},
		{Key: "environmentName", Header: "NAME", WideOnly: true},
	}
}

// EnvironmentCostColumns returns columns for per-environment subscription cost.
func EnvironmentCostColumns() []Column {
	return []Column{
		{Key: "environmentId", Header: "ENVIRONMENT"},
		{Key: "capabilityKey", Header: "CAPABILITY"},
		{Key: "cost", Header: "COST"},
		{Key: "currency", Header: "CURRENCY"},
		{Key: "environmentName", Header: "NAME", WideOnly: true},
	}
}

// OrgLevelUserColumns returns columns for users from the environment-level
// Platform IAM API, whose field names differ from the account-level user API.
func OrgLevelUserColumns() []Column {
	return []Column{
		{Key: "uuid", Header: "UUID"},
		{Key: "email", Header: "EMAIL"},
		{Key: "name", Header: "NAME"},
		{Key: "surname", Header: "SURNAME", WideOnly: true},
	}
}

// OrgLevelGroupColumns returns columns for groups from the environment-level
// Platform IAM API.
func OrgLevelGroupColumns() []Column {
	return []Column{
		{Key: "uuid", Header: "UUID"},
		{Key: "name", Header: "NAME"},
		{Key: "description", Header: "DESCRIPTION", WideOnly: true},
		{Key: "owner", Header: "OWNER", WideOnly: true},
	}
}

// ContextColumns returns columns for context configuration.
func ContextColumns() []Column {
	return []Column{
		{Key: "name", Header: "NAME"},
		{Key: "account_uuid", Header: "ACCOUNT-UUID"},
		{Key: "credentials_ref", Header: "CREDENTIALS"},
		{Key: "current", Header: "CURRENT"},
	}
}

// CredentialColumns returns columns for credential configuration.
func CredentialColumns() []Column {
	return []Column{
		{Key: "name", Header: "NAME"},
		{Key: "client_id", Header: "CLIENT-ID"},
	}
}

// formatCount formats a slice or map as a count.
func formatCount(v any) string {
	switch val := v.(type) {
	case []any:
		return fmt.Sprintf("%d", len(val))
	case []string:
		return fmt.Sprintf("%d", len(val))
	case []map[string]any:
		return fmt.Sprintf("%d", len(val))
	case map[string]any:
		return fmt.Sprintf("%d", len(val))
	case int:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%d", int(val))
	case nil:
		return "0"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatPercent formats a number as a percentage.
func formatPercent(v any) string {
	switch val := v.(type) {
	case float64:
		return fmt.Sprintf("%.1f%%", val)
	case int:
		return fmt.Sprintf("%d%%", val)
	case nil:
		return "N/A"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatList formats a slice as a comma-separated list.
func FormatList(v any) string {
	switch val := v.(type) {
	case []any:
		strs := make([]string, len(val))
		for i, item := range val {
			strs[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(strs, ", ")
	case []string:
		return strings.Join(val, ", ")
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// FilterColumns returns columns filtered for wide mode.
func FilterColumns(columns []Column, wide bool) []Column {
	if wide {
		return columns
	}

	filtered := make([]Column, 0, len(columns))
	for _, col := range columns {
		if !col.WideOnly {
			filtered = append(filtered, col)
		}
	}
	return filtered
}
