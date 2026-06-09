package pagination

import "fmt"

const MaxMessageLen = 4000

func PlainText(s string) []string {
	return paginateOutboundText(s, plainPageMarker)
}

func TelegramText(s string) []string {
	return paginateOutboundText(s, telegramPageMarker)
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
