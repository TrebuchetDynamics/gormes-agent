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
	end := markdownSafeTruncateEnd(runes, max-1)
	return string(runes[:end]) + "…"
}

func markdownSafeTruncateEnd(runes []rune, end int) int {
	if end <= 0 || end > len(runes) || trailingBackslashCount(runes[:end])%2 == 0 {
		return end
	}
	return end - 1
}

func trailingBackslashCount(runes []rune) int {
	count := 0
	for i := len(runes) - 1; i >= 0 && runes[i] == '\\'; i-- {
		count++
	}
	return count
}
