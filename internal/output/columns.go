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
func EnvironmentColumns() []Column {
	return []Column{
		{Key: "id", Header: "ID"},
		{Key: "name", Header: "NAME"},
		{Key: "state", Header: "STATE"},
		{Key: "trial", Header: "TRIAL", WideOnly: true},
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

// LimitColumns returns columns for limit resources.
func LimitColumns() []Column {
	return []Column{
		{Key: "name", Header: "NAME"},
		{Key: "current", Header: "CURRENT"},
		{Key: "max", Header: "MAX"},
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
func TokenColumns() []Column {
	return []Column{
		{Key: "id", Header: "ID"},
		{Key: "name", Header: "NAME"},
		{Key: "expiresIn", Header: "EXPIRES"},
		{Key: "scopes", Header: "SCOPES", WideOnly: true, Formatter: FormatList},
		{Key: "createdAt", Header: "CREATED", WideOnly: true},
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
