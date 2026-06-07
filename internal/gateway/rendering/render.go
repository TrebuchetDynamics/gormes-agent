package rendering

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/renderframe"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/browserartifacts"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/pagination"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/providererrors"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/telegramtext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/textlimit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/toolprogress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const (
	maxMessageLen       = pagination.MaxMessageLen
	streamPreviewCursor = " ▉"
)

// MaxMessageLen is the maximum platform message length used by gateway renderers.
const MaxMessageLen = maxMessageLen

// ToolProgressStatus is the channel-neutral lifecycle state for structured
// tool progress. Text-only channels can ignore it; first-party channels such
// as Navivox can render it as native UI instead of assistant prose.
type ToolProgressStatus = toolprogress.Status

const (
	ToolProgressStarted  = toolprogress.Started
	ToolProgressUpdated  = toolprogress.Updated
	ToolProgressFinished = toolprogress.Finished
	ToolProgressFailed   = toolprogress.Failed
)

// ToolProgressEvent carries redacted, bounded tool-progress evidence. It must
// never include raw tool arguments, stdout, credentials, or full logs.
type ToolProgressEvent = toolprogress.Event

// FormatStreamPlain renders a streaming frame as plain text.
func FormatStreamPlain(f kernel.RenderFrame) string {
	body := f.DraftText
	if streamPreviewCursorActive(f) {
		body += streamPreviewCursor
	}
	tail := ""
	if f.Phase == kernel.PhaseReconnecting {
		tail += "\n\nreconnecting…"
	}
	return truncate(body + tail)
}

// FormatToolProgressPlain renders the persistent Hermes-style tool progress
// transcript for gateway platforms that can edit progress messages.
func FormatToolProgressPlain(f kernel.RenderFrame) string {
	return toolprogress.FormatPlain(f)
}

// FormatToolProgressPlainMode renders tool progress with Hermes gateway
// display.tool_progress semantics for the compact progress transcript.
func FormatToolProgressPlainMode(f kernel.RenderFrame, mode string) string {
	return toolprogress.FormatPlainMode(f, mode)
}

// FormatToolProgressEvents extracts bounded structured tool progress for
// first-party channels. Unlike text progress, summaries deliberately avoid raw
// tool arguments so URLs, command lines, and credentials cannot become chat
// prose.
func FormatToolProgressEvents(f kernel.RenderFrame, mode, requestID string) []ToolProgressEvent {
	return toolprogress.Events(f, mode, requestID)
}

// FormatFinalPlain returns the final assistant text from render history.
func FormatFinalPlain(f kernel.RenderFrame) string {
	return FormatFinalPlainText(FinalAssistantText(f))
}

// FinalAssistantText returns the raw final assistant text from render history.
func FinalAssistantText(f kernel.RenderFrame) string {
	return renderframe.LastAssistantText(f)
}

func FormatFinalPlainText(text string) string {
	if strings.TrimSpace(text) == "" {
		return "(empty reply)"
	}
	return text
}

// FormatErrorPlain renders a terminal error frame.
func FormatErrorPlain(f kernel.RenderFrame) string {
	text := "❌ " + sanitizeProviderErrorText(f.LastError)
	if f.LastError == "" {
		text = "❌ cancelled"
	}
	return truncate(text)
}

func escapeTelegramMarkdownV2(text string) string {
	return telegramtext.EscapeMarkdownV2(text)
}

// FormatStreamTelegram renders a streaming frame using Telegram MarkdownV2.
func FormatStreamTelegram(f kernel.RenderFrame) string {
	body := escapeTelegramMarkdownV2(f.DraftText)
	if streamPreviewCursorActive(f) {
		body += streamPreviewCursor
	}
	tail := ""
	if f.Phase == kernel.PhaseReconnecting {
		tail += "\n\n_reconnecting…_"
	}
	return truncate(body + tail)
}

// FormatToolProgressTelegram renders plain tool-progress text for Telegram.
// Does NOT use MarkdownV2 escaping because tool progress contains URLs, file
// paths, and code snippets that would be broken by _ ( ) - escaping. The
// Bot.Send fallback path strips MarkdownV2 if Telegram rejects the parse.
func FormatToolProgressTelegram(f kernel.RenderFrame) string {
	return FormatToolProgressTelegramMode(f, "all")
}

func FormatToolProgressTelegramMode(f kernel.RenderFrame, mode string) string {
	progress := FormatToolProgressPlainMode(f, mode)
	if strings.TrimSpace(progress) == "" {
		return ""
	}
	// Tool progress is plain text with URLs, paths, and emoji — NOT markdown.
	// Escaping for MarkdownV2 would break URLs (_ → \_) and make backslashes
	// visible. Just truncate; the bot's sendWithParseFallback handles any
	// MarkdownV2 parse failures by retrying without parse_mode.
	return truncate(progress)
}

// FormatFinalTelegram renders the final assistant message for Telegram.
func FormatFinalTelegram(f kernel.RenderFrame) string {
	return FormatFinalTelegramText(FinalAssistantText(f))
}

func FormatFinalTelegramText(text string) string {
	return FormatFinalMarkdown(text, FormatTargetTelegram)
}

// FormatBrowserArtifactTelegram renders a browser result envelope as a bounded
// Telegram MarkdownV2 artifact notice. It keeps browser/tool progress separate
// from final assistant text and never exposes local screenshot paths or raw
// persisted artifact bytes.
func FormatBrowserArtifactTelegram(envelope tools.BrowserResultEnvelope) string {
	return browserartifacts.Telegram(envelope)
}

// FormatErrorTelegram renders an error frame for Telegram MarkdownV2.
func FormatErrorTelegram(f kernel.RenderFrame) string {
	text := "❌ " + sanitizeProviderErrorText(f.LastError)
	if f.LastError == "" {
		text = "❌ cancelled"
	}
	return truncate(tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, text))
}

func streamPreviewCursorActive(f kernel.RenderFrame) bool {
	if strings.TrimSpace(f.DraftText) == "" {
		return false
	}
	switch f.Phase {
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseFinalizing, kernel.PhaseReconnecting:
		return true
	default:
		return false
	}
}

func ToolProgressSummary(name string, status ToolProgressStatus) string {
	return toolprogress.Summary(name, status)
}

func NormalizeGatewayToolProgressMode(mode string) string {
	return toolprogress.NormalizeMode(mode)
}

func normalizeGatewayToolProgressMode(mode string) string {
	return NormalizeGatewayToolProgressMode(mode)
}

func sanitizeProviderErrorText(s string) string {
	return providererrors.SanitizeText(s)
}

func truncate(s string) string {
	return textlimit.TruncateMarkdownV2Safe(s, maxMessageLen)
}

func PaginatePlainText(s string) []string {
	return pagination.PlainText(s)
}

func PaginateTelegramText(s string) []string {
	return pagination.TelegramText(s)
}

func paginatePlainText(s string) []string {
	return PaginatePlainText(s)
}

func paginateTelegramText(s string) []string {
	return PaginateTelegramText(s)
}
