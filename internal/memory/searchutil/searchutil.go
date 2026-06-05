package searchutil

import "strings"

// SameChatKey compares chat keys while treating source names as
// case-insensitive and preserving chat IDs exactly.
func SameChatKey(a, b string) bool {
	aSource, aID, aOK := splitChatKey(a)
	bSource, bID, bOK := splitChatKey(b)
	if !aOK || !bOK {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	return strings.EqualFold(aSource, bSource) && aID == bID
}

func splitChatKey(chatKey string) (string, string, bool) {
	source, chatID, ok := strings.Cut(strings.TrimSpace(chatKey), ":")
	if !ok || strings.TrimSpace(source) == "" || strings.TrimSpace(chatID) == "" {
		return "", "", false
	}
	return strings.TrimSpace(source), strings.TrimSpace(chatID), true
}

// SanitizeFTS5Pattern keeps the memory/session search query inside the small
// subset accepted by the local FTS5 MATCH builders.
func SanitizeFTS5Pattern(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == ' ', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte(' ')
		}
	}
	// Collapse runs of spaces.
	out := b.String()
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return strings.TrimSpace(out)
}
