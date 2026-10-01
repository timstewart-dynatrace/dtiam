// Package utils provides shared utility functions.
package utils

// StringFrom safely extracts a string value from a map.
// Returns empty string if the key is missing or the value is not a string.
func StringFrom(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key].(string)
	if !ok {
		return ""
	}
	return v
}

// IntFrom safely extracts an int value from a map.
// Returns 0 if the key is missing or the value is not numeric.
// Handles both int and float64 (JSON unmarshaling produces float64).
func IntFrom(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}

// Float64From safely extracts a float64 value from a map.
// Returns 0 if the key is missing or the value is not numeric.
func Float64From(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// BoolFrom safely extracts a bool value from a map.
// Returns false if the key is missing or the value is not a bool.
func BoolFrom(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key].(bool)
	if !ok {
		return false
	}
	return v
}

// SliceFrom safely extracts a []any value from a map.
// Returns nil if the key is missing or the value is not a slice.
func SliceFrom(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, ok := m[key].([]any)
	if !ok {
		return nil
	}
	return v
}

// MapFrom safely extracts a map[string]any value from a map.
// Returns nil if the key is missing or the value is not a map.
func MapFrom(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, ok := m[key].(map[string]any)
	if !ok {
		return nil
	}
	return v
}

// StringSliceFrom safely extracts a []string from a map.
// Handles both []string and []any containing strings.
func StringSliceFrom(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	switch v := m[key].(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
