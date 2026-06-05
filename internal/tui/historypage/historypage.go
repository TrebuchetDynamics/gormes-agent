package historypage

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/transientpage"
)

// SlashResult is the behavior-only result for /history.
type SlashResult struct {
	Page          transientpage.State
	Open          bool
	StatusMessage string
}

// HandleSlash builds the transient history page or reports an empty transcript.
func HandleSlash(history []llm.Message, input string) SlashResult {
	page, ok := Build(history, PreviewLimit(input))
	if !ok {
		return SlashResult{StatusMessage: "no conversation yet"}
	}
	return SlashResult{Page: page, Open: true, StatusMessage: "history opened"}
}

func PreviewLimit(input string) int {
	fields := strings.Fields(strings.TrimSpace(input))
	preview := 400
	if len(fields) > 1 {
		if n, err := strconv.Atoi(fields[1]); err == nil && n > 0 {
			preview = n
		}
	}
	if preview < 80 {
		preview = 80
	}
	return preview
}

func Build(history []llm.Message, preview int) (transientpage.State, bool) {
	if preview < 80 {
		preview = 80
	}
	items := make([]llm.Message, 0, len(history))
	for _, msg := range history {
		switch msg.Role {
		case "user", "assistant":
			items = append(items, msg)
		}
	}
	if len(items) == 0 {
		return transientpage.State{}, false
	}

	blocks := make([]string, 0, len(items))
	for i, msg := range items {
		name := "Gormes"
		if msg.Role == "user" {
			name = "You"
		}
		body := strings.TrimSpace(MessageText(msg))
		if body == "" {
			if len(msg.ToolCalls) > 0 {
				body = fmt.Sprintf("(%d tool calls)", len(msg.ToolCalls))
			} else {
				body = "(empty)"
			}
		}
		body = truncateEllipsis(body, preview+1)
		blocks = append(blocks, fmt.Sprintf("[%s #%d]\n%s", name, i+1, body))
	}
	return transientpage.State{Title: "History", Body: strings.Join(blocks, "\n\n")}, true
}

func MessageText(msg llm.Message) string {
	if strings.TrimSpace(msg.Content) != "" {
		return msg.Content
	}
	if len(msg.ContentParts) == 0 {
		return ""
	}
	parts := make([]string, 0, len(msg.ContentParts))
	for _, part := range msg.ContentParts {
		if strings.TrimSpace(part.Text) != "" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func truncateEllipsis(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	runes := []rune(s)
	return string(runes[:n-1]) + "…"
}
