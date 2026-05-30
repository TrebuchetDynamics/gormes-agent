package textvalue

import "strings"

// FirstNonEmptyTrimmed returns the first value whose trimmed form is non-empty.
// The returned value is trimmed so status/evidence fields do not preserve
// accidental surrounding whitespace.
func FirstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
