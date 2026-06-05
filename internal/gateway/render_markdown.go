package gateway

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// FormatTarget specifies the messaging platform for markdown rendering.
type FormatTarget = rendering.FormatTarget

const (
	FormatTargetTelegram = rendering.FormatTargetTelegram
	FormatTargetDiscord  = rendering.FormatTargetDiscord
	FormatTargetSlack    = rendering.FormatTargetSlack
)

// FormatFinalMarkdown renders the final assistant text as formatted markdown
// for the given platform target.
func FormatFinalMarkdown(text string, target FormatTarget) string {
	return rendering.FormatFinalMarkdown(text, target)
}

// FormatFinalDiscordText renders final assistant text for Discord markdown.
func FormatFinalDiscordText(text string) string { return rendering.FormatFinalDiscordText(text) }

// FormatFinalSlackText renders final assistant text for Slack mrkdwn.
func FormatFinalSlackText(text string) string { return rendering.FormatFinalSlackText(text) }

// FormatFinalDiscord renders the final assistant message for Discord.
func FormatFinalDiscord(f kernel.RenderFrame) string { return rendering.FormatFinalDiscord(f) }

// FormatFinalSlack renders the final assistant message for Slack.
func FormatFinalSlack(f kernel.RenderFrame) string { return rendering.FormatFinalSlack(f) }
