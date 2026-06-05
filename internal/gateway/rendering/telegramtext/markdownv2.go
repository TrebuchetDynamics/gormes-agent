package telegramtext

import "strings"

// EscapeMarkdownV2 escapes all Telegram MarkdownV2 reserved characters plus
// backslash: _ * [ ] ( ) ~ ` > # + - = | { } . ! and \.
func EscapeMarkdownV2(text string) string {
	var b strings.Builder
	b.Grow(len(text) * 2) // worst case: every char escaped
	for _, r := range text {
		switch r {
		case '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
