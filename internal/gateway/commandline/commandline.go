package commandline

import "strings"

// Name normalizes a slash command token/name for lookup. It trims bot mention
// suffixes and ignores trailing arguments.
func Name(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	key = strings.TrimPrefix(key, "/")
	if i := strings.IndexAny(key, " \t\r\n"); i >= 0 {
		key = key[:i]
	}
	if i := strings.IndexByte(key, '@'); i >= 0 {
		key = key[:i]
	}
	return key
}

// Split separates the leading command token from the rest of a command line.
func Split(input string) (token, args string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", ""
	}
	for i, r := range trimmed {
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' {
			return trimmed[:i], strings.TrimSpace(trimmed[i:])
		}
	}
	return trimmed, ""
}
