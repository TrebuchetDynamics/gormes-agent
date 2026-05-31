package textlimit

// TruncateMarkdownV2Safe bounds s to max runes and appends an ellipsis when
// truncated. If the cut would leave a dangling MarkdownV2 escape backslash, it
// backs up so the ellipsis is not parsed as part of an incomplete escape.
func TruncateMarkdownV2Safe(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	end := max - 1
	for end > 0 && runes[end-1] == '\\' {
		end--
	}
	return string(runes[:end]) + "…"
}
