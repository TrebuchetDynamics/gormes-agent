package textvalue

import "strings"

// FirstNonEmptyTrimmed returns the first string with non-whitespace content,
// trimmed for stable config, routing, and delivery fallback values.
func FirstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
