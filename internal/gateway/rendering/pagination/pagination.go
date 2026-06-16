package pagination

import (
	"fmt"
	"strings"
)

const MaxMessageLen = 4000

func PlainText(s string) []string {
	return paginateOutboundText(s, plainPageMarker)
}

func TelegramText(s string) []string {
	pages := paginateOutboundText(s, telegramPageMarker)
	if len(pages) == 1 {
		pages[0] = removeDanglingMarkdownEscape(pages[0])
		return pages
	}
	for i, page := range pages {
		pages[i] = sanitizeTelegramPage(page)
	}
	return pages
}

func plainText(s string) []string {
	return PlainText(s)
}

func telegramText(s string) []string {
	return TelegramText(s)
}

func paginateOutboundText(s string, marker func(int, int) string) []string {
	if marker == nil {
		marker = plainPageMarker
	}
	if len([]rune(s)) <= MaxMessageLen {
		return []string{s}
	}
	const markerReserve = 32
	bodyLimit := MaxMessageLen - markerReserve
	if bodyLimit < 1 {
		bodyLimit = MaxMessageLen / 2
	}
	chunks := splitOutboundText(s, bodyLimit)
	if len(chunks) <= 1 {
		return chunks
	}
	pages := make([]string, len(chunks))
	total := len(chunks)
	for i, chunk := range chunks {
		pages[i] = chunk + marker(i+1, total)
	}
	return pages
}

func splitOutboundText(s string, limit int) []string {
	if limit <= 0 || len([]rune(s)) <= limit {
		return []string{s}
	}
	var chunks []string
	remaining := []rune(s)
	for len(remaining) > limit {
		split := outboundSplitIndex(remaining, limit)
		if split <= 0 || split > len(remaining) {
			split = limit
		}
		split = markdownSafeSplitIndex(remaining, split)
		chunks = append(chunks, string(remaining[:split]))
		remaining = remaining[split:]
	}
	if len(remaining) > 0 {
		chunks = append(chunks, string(remaining))
	}
	return chunks
}

func outboundSplitIndex(runes []rune, limit int) int {
	if len(runes) <= limit {
		return len(runes)
	}
	window := runes[:limit]
	if idx := lastRuneIndex(window, '\n'); idx >= limit/2 {
		return idx + 1
	}
	if idx := lastRuneIndex(window, ' '); idx >= limit/2 {
		return idx + 1
	}
	return limit
}

func markdownSafeSplitIndex(runes []rune, split int) int {
	if split <= 0 || split > len(runes) || trailingBackslashCount(runes[:split])%2 == 0 {
		return split
	}
	if split == 1 && len(runes) > 1 {
		return 2
	}
	return split - 1
}

func sanitizeTelegramPage(page string) string {
	body, marker, ok := splitTelegramPageMarker(page)
	if ok {
		return removeDanglingMarkdownEscape(body) + marker
	}
	return removeDanglingMarkdownEscape(page)
}

func splitTelegramPageMarker(page string) (body, marker string, ok bool) {
	idx := strings.LastIndex(page, "\n\n\\(")
	if idx < 0 {
		return "", "", false
	}
	candidate := page[idx:]
	if !validTelegramPageMarker(candidate) {
		return "", "", false
	}
	return page[:idx], candidate, true
}

func validTelegramPageMarker(marker string) bool {
	if !strings.HasPrefix(marker, "\n\n\\(") || !strings.HasSuffix(marker, "\\)") {
		return false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(marker, "\n\n\\("), "\\)")
	left, right, ok := strings.Cut(inner, "/")
	return ok && decimalText(left) && decimalText(right)
}

func decimalText(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func removeDanglingMarkdownEscape(s string) string {
	runes := []rune(s)
	if len(runes) > 0 && trailingBackslashCount(runes)%2 == 1 {
		return string(runes[:len(runes)-1])
	}
	return s
}

func trailingBackslashCount(runes []rune) int {
	count := 0
	for i := len(runes) - 1; i >= 0 && runes[i] == '\\'; i-- {
		count++
	}
	return count
}

func lastRuneIndex(runes []rune, needle rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == needle {
			return i
		}
	}
	return -1
}

func plainPageMarker(page, total int) string {
	return fmt.Sprintf("\n\n(%d/%d)", page, total)
}

func telegramPageMarker(page, total int) string {
	return fmt.Sprintf("\n\n\\(%d/%d\\)", page, total)
}
