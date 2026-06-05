package commandline

import (
	"strings"
	"unicode"
)

// Name normalizes a slash command token/name for lookup. It trims bot mention
// suffixes and ignores trailing arguments.
func Name(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	key = strings.TrimPrefix(key, "/")
	if i := strings.IndexFunc(key, isCommandSpace); i >= 0 {
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
		if isCommandSpace(r) {
			return trimmed[:i], strings.TrimSpace(trimmed[i:])
		}
	}
	return trimmed, ""
}

// PayloadIfCommand returns the trimmed argument payload when input starts with
// the named slash command. Bot mention suffixes on the command token are
// ignored. Non-command payloads return ok=false so callers can preserve their
// existing already-split payload behavior explicitly.
func PayloadIfCommand(input, command string) (payload string, ok bool) {
	token, args := Split(input)
	if token == "" || Name(token) != Name(command) {
		return "", false
	}
	return args, true
}

func isCommandSpace(r rune) bool {
	return unicode.IsSpace(r)
}
