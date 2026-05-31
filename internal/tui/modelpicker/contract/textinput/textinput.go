package textinput

import "strings"

// TrimBoundary trims whitespace at model-picker contract boundaries while
// preserving caller-owned casing and internal spacing.
func TrimBoundary(value string) string {
	return strings.TrimSpace(value)
}

// FirstNonEmpty returns the first non-empty already-normalized value.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
