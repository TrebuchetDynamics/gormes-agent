package textlist

import "strings"

// Compact returns trimmed, non-empty values in their original order.
func Compact(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// Field names one required string config field for deterministic missing-field reports.
type Field struct {
	Name  string
	Value string
}

// MissingBlank returns field names whose values are blank after trimming.
func MissingBlank(fields []Field) []string {
	missing := []string{}
	for _, field := range fields {
		if strings.TrimSpace(field.Value) == "" {
			missing = append(missing, field.Name)
		}
	}
	return missing
}
