package commandline

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Name normalizes a slash command token/name for lookup. It trims bot mention
// suffixes and ignores trailing arguments.
func Name(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "／") {
		_, size := utf8.DecodeRuneInString(key)
		key = key[size:]
		if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "／") {
			return ""
		}
	}
	if i := strings.IndexFunc(key, isCommandSpace); i >= 0 {
		key = key[:i]
	}
	if i := strings.IndexByte(key, '@'); i >= 0 {
		suffix := key[i+1:]
		if strings.TrimSpace(suffix) == "" || strings.Contains(suffix, "@") || strings.IndexFunc(suffix, isCommandSpace) >= 0 || !validBotMentionSuffix(suffix) {
			return ""
		}
		key = key[:i]
	}
	if strings.ContainsFunc(key, unsafeCommandNameRune) {
		return ""
	}
	return key
}

func unsafeCommandNameRune(r rune) bool {
	if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
		return true
	}
	switch {
	case r >= 0x200b && r <= 0x200f:
		return true
	case r >= 0x2028 && r <= 0x202e:
		return true
	case r >= 0x2060 && r <= 0x2069:
		return true
	case r == 0xfeff || r == 0xfffc:
		return true
	case r >= 0xfff9 && r <= 0xfffb:
		return true
	default:
		return false
	}
}

func validBotMentionSuffix(suffix string) bool {
	for _, r := range suffix {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
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
	commandName := Name(command)
	if commandName == "" {
		return "", false
	}
	token, args := Split(input)
	if token == "" || !isSlashCommandToken(token) || Name(token) != commandName {
		return "", false
	}
	return args, true
}

func isSlashCommandToken(token string) bool {
	return strings.HasPrefix(token, "/") || strings.HasPrefix(token, "／")
}

func isCommandSpace(r rune) bool {
	return unicode.IsSpace(r)
}
