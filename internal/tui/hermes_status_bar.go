package tui

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar"
	"github.com/charmbracelet/lipgloss"
)

// HermesStatusModel is the pure input to the Hermes-compatible status bar
// renderer. It carries the data Hermes' cli.py:_get_status_bar_snapshot
// produces in Python — model name, token usage, session duration in seconds,
// and per-prompt elapsed seconds — without coupling to Bubble Tea, providers,
// or wall-clock IO.
type HermesStatusModel = statusbar.HermesModel

// HermesStatusContextSeverity classifies the context-usage percent into the
// same five buckets Hermes' status bar uses to colour the percentage label.
type HermesStatusContextSeverity = statusbar.HermesContextSeverity

const (
	HermesStatusContextDim      HermesStatusContextSeverity = statusbar.HermesContextDim
	HermesStatusContextGood     HermesStatusContextSeverity = statusbar.HermesContextGood
	HermesStatusContextWarn     HermesStatusContextSeverity = statusbar.HermesContextWarn
	HermesStatusContextBad      HermesStatusContextSeverity = statusbar.HermesContextBad
	HermesStatusContextCritical HermesStatusContextSeverity = statusbar.HermesContextCritical
)

// HermesStatusBarContextSeverity mirrors Hermes' _status_bar_context_style:
// nil → dim, <50 good, 50–80 warn, 81–94 bad, ≥95 critical.
func HermesStatusBarContextSeverity(percent *int) HermesStatusContextSeverity {
	return statusbar.HermesContextSeverityFor(percent)
}

// RenderHermesStatusBar renders the single-line Hermes-compatible status rule
// for the given width. The row follows current Hermes Ink's StatusRule shape:
// a leading rule/status segment, bar-separated model/usage details, and an
// optional cwd label on the right. Width tiers keep the row readable on small
// terminals and output is trimmed with an ellipsis so it never wraps.
func RenderHermesStatusBar(model HermesStatusModel, width int) string {
	return renderHermesStatusBar(model, width)
}

func RenderHermesStatusBarWithSkin(model HermesStatusModel, width int, skin HermesSkin) string {
	line := renderHermesStatusBar(model, width)
	if line == "" {
		return ""
	}
	styles := SkinStylesFor(skin)
	line = styleHermesStatusBarSegments(line, model, width, styles)
	return styles.Status.Width(width).Render(line)
}

func styleHermesStatusBarSegments(line string, model HermesStatusModel, width int, styles SkinStyles) string {
	percent := hermesStatusPercent(model.ContextTokens, model.ContextLength)
	if percent != nil {
		percentLabel := fmt.Sprintf("%d%%", *percent)
		segment := percentLabel
		if width >= 76 {
			segment = fmt.Sprintf("[%s] %s", hermesStatusContextBar(*percent), percentLabel)
		}
		line = strings.Replace(line, segment, hermesStatusContextStyle(styles, percent).Render(segment), 1)
	}
	if model.HasPromptElapsed && width >= 76 {
		elapsed := hermesStatusPromptElapsed(model.PromptElapsed, model.PromptLive)
		line = strings.Replace(line, elapsed, styles.Warn.Background(styles.Status.GetBackground()).Render(elapsed), 1)
	}
	if cwd := strings.TrimSpace(model.CWDLabel); cwd != "" && width >= 76 {
		line = strings.Replace(line, cwd, styles.Dim.Background(styles.Status.GetBackground()).Render(cwd), 1)
	}
	return line
}

func hermesStatusContextStyle(styles SkinStyles, percent *int) lipgloss.Style {
	var style lipgloss.Style
	switch HermesStatusBarContextSeverity(percent) {
	case HermesStatusContextGood:
		style = styles.Good
	case HermesStatusContextWarn:
		style = styles.Warn
	case HermesStatusContextBad:
		style = styles.Bad
	case HermesStatusContextCritical:
		style = styles.Critical
	default:
		style = styles.Dim
	}
	return style.Background(styles.Status.GetBackground())
}

func renderHermesStatusBar(model HermesStatusModel, width int) string {
	return statusbar.RenderHermes(model, width)
}

func hermesStatusModelLabel(name, effort string, fast bool) string {
	return statusbar.HermesModelLabel(name, effort, fast)
}

func hermesStatusContextBar(percent int) string {
	return statusbar.HermesContextBar(percent)
}

func hermesStatusDurationLabel(seconds int64) string {
	return statusbar.HermesDurationLabel(seconds)
}

func hermesStatusPercent(tokens, length int) *int {
	return statusbar.HermesPercent(tokens, length)
}

func hermesStatusFormatTokenCount(value int) string {
	return statusbar.HermesFormatTokenCount(value)
}

func hermesStatusFormatContextLength(tokens int) string {
	return statusbar.HermesFormatContextLength(tokens)
}

func hermesStatusPromptElapsed(seconds int64, live bool) string {
	return statusbar.HermesPromptElapsed(seconds, live)
}

func hermesStatusTrimToWidth(text string, maxWidth int) string {
	return statusbar.HermesTrimToWidth(text, maxWidth)
}
