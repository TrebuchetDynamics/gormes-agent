package markdown

import "strings"

// JoinOrDash returns a comma-separated list or '-' for empty values.
func JoinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

// JoinCodeOrDash returns a comma-separated list of inline-code values or '-'
// for empty values.
func JoinCodeOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "`"+value+"`")
	}
	return strings.Join(quoted, ", ")
}

// Cell escapes text for markdown table cells and returns '-' for empty values.
func Cell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	if s == "" {
		return "-"
	}
	return s
}
