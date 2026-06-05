package gateway

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const maxMessageLen = rendering.MaxMessageLen

// FormatStreamPlain renders a streaming frame as plain text.
func FormatStreamPlain(f kernel.RenderFrame) string { return rendering.FormatStreamPlain(f) }

// FormatToolProgressPlain renders the persistent Hermes-style tool progress
// transcript for gateway platforms that can edit progress messages.
func FormatToolProgressPlain(f kernel.RenderFrame) string {
	return rendering.FormatToolProgressPlain(f)
}

// FormatToolProgressPlainMode renders tool progress with Hermes gateway
// display.tool_progress semantics for the compact progress transcript.
func FormatToolProgressPlainMode(f kernel.RenderFrame, mode string) string {
	return rendering.FormatToolProgressPlainMode(f, mode)
}

// FormatToolProgressEvents extracts bounded structured tool progress for
// first-party channels.
func FormatToolProgressEvents(f kernel.RenderFrame, mode, requestID string) []ToolProgressEvent {
	return rendering.FormatToolProgressEvents(f, mode, requestID)
}

// FormatFinalPlain returns the final assistant text from render history.
func FormatFinalPlain(f kernel.RenderFrame) string { return rendering.FormatFinalPlain(f) }

// FinalAssistantText returns the raw final assistant text from render history.
func FinalAssistantText(f kernel.RenderFrame) string { return rendering.FinalAssistantText(f) }

func FormatFinalPlainText(text string) string { return rendering.FormatFinalPlainText(text) }

// FormatErrorPlain renders a terminal error frame.
func FormatErrorPlain(f kernel.RenderFrame) string { return rendering.FormatErrorPlain(f) }

func FormatStreamTelegram(f kernel.RenderFrame) string { return rendering.FormatStreamTelegram(f) }

func FormatToolProgressTelegram(f kernel.RenderFrame) string {
	return rendering.FormatToolProgressTelegram(f)
}

func FormatToolProgressTelegramMode(f kernel.RenderFrame, mode string) string {
	return rendering.FormatToolProgressTelegramMode(f, mode)
}

// FormatFinalTelegram renders the final assistant message for Telegram.
func FormatFinalTelegram(f kernel.RenderFrame) string { return rendering.FormatFinalTelegram(f) }

func FormatFinalTelegramText(text string) string { return rendering.FormatFinalTelegramText(text) }

// FormatBrowserArtifactTelegram renders a browser result envelope as a bounded
// Telegram MarkdownV2 message.
func FormatBrowserArtifactTelegram(envelope tools.BrowserResultEnvelope) string {
	return rendering.FormatBrowserArtifactTelegram(envelope)
}

func FormatErrorTelegram(f kernel.RenderFrame) string { return rendering.FormatErrorTelegram(f) }

func normalizeGatewayToolProgressMode(mode string) string {
	return rendering.NormalizeGatewayToolProgressMode(mode)
}

func toolProgressSummary(name string, status ToolProgressStatus) string {
	return rendering.ToolProgressSummary(name, status)
}

func paginatePlainText(s string) []string { return rendering.PaginatePlainText(s) }

func paginateTelegramText(s string) []string { return rendering.PaginateTelegramText(s) }
