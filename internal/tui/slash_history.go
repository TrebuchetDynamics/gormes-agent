package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func historySlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "history: TUI unavailable"}
	}
	page, ok := BuildHistoryPage(model.frame.History, historyPreviewLimit(input))
	if !ok {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "no conversation yet"}
	}
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: "history opened"}
}

func historyPreviewLimit(input string) int {
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

func BuildHistoryPage(history []hermes.Message, preview int) (TransientPageState, bool) {
	if preview < 80 {
		preview = 80
	}
	items := make([]hermes.Message, 0, len(history))
	for _, msg := range history {
		switch msg.Role {
		case "user", "assistant":
			items = append(items, msg)
		}
	}
	if len(items) == 0 {
		return TransientPageState{}, false
	}

	blocks := make([]string, 0, len(items))
	for i, msg := range items {
		name := "Gormes"
		if msg.Role == "user" {
			name = "You"
		}
		body := strings.TrimSpace(historyMessageText(msg))
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
	return TransientPageState{Title: "History", Body: strings.Join(blocks, "\n\n")}, true
}

func historyMessageText(msg hermes.Message) string {
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
