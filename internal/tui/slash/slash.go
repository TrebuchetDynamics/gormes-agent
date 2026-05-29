package slash

import "strings"

// NormalizeName returns a registry key for a slash command name. Callers may
// pass either "save" or "/save"; matching stays case-insensitive.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimPrefix(name, "/"))
}

// InvocationArgs returns the raw argument tail after the first whitespace-
// delimited command token, trimmed for downstream command adapters.
func InvocationArgs(input string) string {
	trimmed := strings.TrimSpace(input)
	for i, r := range trimmed {
		switch r {
		case ' ', '\t', '\r', '\n':
			return strings.TrimSpace(trimmed[i:])
		}
	}
	return ""
}
