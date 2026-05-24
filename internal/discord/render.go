package discord

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/trace"
)

const maxDiscordText = 2000

func formatStream(f kernel.RenderFrame) string {
	parts := make([]string, 0, 3)
	if text := strings.TrimSpace(f.DraftText); text != "" {
		parts = append(parts, text)
	}
	if text := formatToolTrace(f.SoulEvents); text != "" {
		parts = append(parts, text)
	}
	if f.Phase == kernel.PhaseReconnecting {
		parts = append(parts, "reconnecting...")
	}
	if len(parts) == 0 {
		return "⏳"
	}
	return truncateDiscord(strings.Join(parts, "\n\n"))
}

func formatToolTrace(events []kernel.SoulEntry) string {
	texts := make([]string, 0, len(events))
	for _, event := range events {
		texts = append(texts, event.Text)
	}
	return trace.FormatBlock(texts)
}

func formatFinal(f kernel.RenderFrame) string {
	for i := len(f.History) - 1; i >= 0; i-- {
		if f.History[i].Role == "assistant" {
			return truncateDiscord(f.History[i].Content)
		}
	}
	return "(empty reply)"
}

func formatError(f kernel.RenderFrame) string {
	text := f.LastError
	if strings.TrimSpace(text) == "" {
		text = "cancelled"
	}
	return truncateDiscord("❌ " + text)
}

func truncateDiscord(s string) string {
	runes := []rune(s)
	if len(runes) <= maxDiscordText {
		return s
	}
	return string(runes[:maxDiscordText-1]) + "…"
}

func stripSelfMention(text, selfID string) string {
	if selfID == "" {
		return strings.TrimSpace(text)
	}
	replacer := strings.NewReplacer("<@"+selfID+">", "", "<@!"+selfID+">", "")
	return strings.TrimSpace(replacer.Replace(text))
}

func hasSelfMention(text, selfID string) bool {
	if selfID == "" {
		return false
	}
	return strings.Contains(text, "<@"+selfID+">") || strings.Contains(text, "<@!"+selfID+">")
}
