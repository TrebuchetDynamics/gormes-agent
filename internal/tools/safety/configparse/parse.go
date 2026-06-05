package configparse

import (
	"strconv"
	"strings"
)

// BoolLike parses a bool-like value from various Go types.
func BoolLike(raw any, fallback bool) bool {
	if raw == nil {
		return fallback
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	case int:
		return v != 0
	case int64:
		return v != 0
	}
	return fallback
}

// BoolLikeWithEvidence parses a bool-like value and returns evidence if invalid.
func BoolLikeWithEvidence(raw any, fallback bool) (bool, string) {
	if raw == nil {
		return fallback, ""
	}
	switch v := raw.(type) {
	case bool:
		return v, ""
	case string:
		trimmed := strings.TrimSpace(v)
		for len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
			trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		}
		for len(trimmed) >= 2 && trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'' {
			trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		}
		switch strings.ToLower(trimmed) {
		case "true", "1", "yes", "on":
			return true, ""
		case "false", "0", "no", "off":
			return false, ""
		}
		return fallback, "invalid_bool_string"
	case int:
		return v != 0, ""
	case int64:
		return v != 0, ""
	}
	return fallback, "invalid_bool_type"
}

// IntLike parses an int-like value from various Go types.
func IntLike(raw any, fallback int) int {
	if raw == nil {
		return fallback
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case string:
		trimmed := strings.TrimSpace(v)
		if parsed, err := strconv.Atoi(trimmed); err == nil {
			return parsed
		}
	}
	return fallback
}
