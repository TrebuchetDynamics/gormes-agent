package rendering

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/markdown"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// FormatTarget specifies the messaging platform for markdown rendering.
type FormatTarget = markdown.FormatTarget

const (
	FormatTargetTelegram = markdown.FormatTargetTelegram
	FormatTargetDiscord  = markdown.FormatTargetDiscord
	FormatTargetSlack    = markdown.FormatTargetSlack
)

// FormatFinalMarkdown renders the final assistant text as formatted markdown
// for the given platform target. Handles headings, lists, code blocks,
// blockquotes, tables, bold, italic, strikethrough, spoiler, links, and code.
func FormatFinalMarkdown(text string, target FormatTarget) string {
	return markdown.FormatFinalMarkdown(text, target)
}

// FormatFinalDiscordText renders final assistant text for Discord markdown.
func FormatFinalDiscordText(text string) string {
	return markdown.FormatFinalDiscordText(text)
}

// FormatFinalSlackText renders final assistant text for Slack mrkdwn.
func FormatFinalSlackText(text string) string {
	return markdown.FormatFinalSlackText(text)
}

// FormatFinalDiscord renders the final assistant message for Discord.
func FormatFinalDiscord(f kernel.RenderFrame) string {
	return FormatFinalDiscordText(FinalAssistantText(f))
}

// FormatFinalSlack renders the final assistant message for Slack.
func FormatFinalSlack(f kernel.RenderFrame) string {
	return FormatFinalSlackText(FinalAssistantText(f))
}
