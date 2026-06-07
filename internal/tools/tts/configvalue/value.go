//go:build !slim

package configvalue

import (
	"fmt"
	"strings"
)

// Map coerces a generic config value into a string-keyed map.
func Map(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	default:
		return nil
	}
}

// LookupCaseInsensitive finds key in values with exact match first, then trimmed case-insensitive match.
func LookupCaseInsensitive(values map[string]any, key string) any {
	if values == nil {
		return nil
	}
	if value, ok := values[key]; ok {
		return value
	}
	for candidate, value := range values {
		if strings.EqualFold(strings.TrimSpace(candidate), key) {
			return value
		}
	}
	return nil
}

// FirstPresent returns the first present config value for keys, preserving zero values.
func FirstPresent(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if values == nil {
			return nil
		}
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

func String(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func Bool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func PositiveInt(value any) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &parsed); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func Float(value any) float64 {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case float64:
		return typed
	case float32:
		return float64(typed)
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%f", &parsed); err == nil {
			return parsed
		}
	}
	return 0
}
