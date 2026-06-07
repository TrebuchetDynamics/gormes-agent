package statusbar

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar/hermes"

// HermesModel is the pure input to the Hermes-compatible status bar renderer.
type HermesModel = hermes.Model

// HermesContextSeverity classifies context usage for Hermes-compatible status bars.
type HermesContextSeverity = hermes.ContextSeverity

const (
	HermesContextDim      HermesContextSeverity = hermes.ContextDim
	HermesContextGood     HermesContextSeverity = hermes.ContextGood
	HermesContextWarn     HermesContextSeverity = hermes.ContextWarn
	HermesContextBad      HermesContextSeverity = hermes.ContextBad
	HermesContextCritical HermesContextSeverity = hermes.ContextCritical
)

func HermesContextSeverityFor(percent *int) HermesContextSeverity {
	return hermes.ContextSeverityFor(percent)
}

func RenderHermes(model HermesModel, width int) string {
	return hermes.Render(model, width)
}

func HermesModelLabel(name, effort string, fast bool) string {
	return hermes.ModelLabel(name, effort, fast)
}

func HermesContextBar(percent int) string {
	return hermes.ContextBar(percent)
}

func HermesDurationLabel(seconds int64) string {
	return hermes.DurationLabel(seconds)
}

func HermesPercent(tokens, length int) *int {
	return hermes.Percent(tokens, length)
}

func HermesFormatTokenCount(value int) string {
	return hermes.FormatTokenCount(value)
}

func HermesFormatContextLength(tokens int) string {
	return hermes.FormatContextLength(tokens)
}

func HermesPromptElapsed(seconds int64, live bool) string {
	return hermes.PromptElapsed(seconds, live)
}

func HermesTrimToWidth(text string, maxWidth int) string {
	return hermes.TrimToWidth(text, maxWidth)
}
