package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HermesStatusModel is the pure input to the Hermes-compatible status bar
// renderer. It carries the data Hermes' cli.py:_get_status_bar_snapshot
// produces in Python — model name, token usage, session duration in seconds,
// and per-prompt elapsed seconds — without coupling to Bubble Tea, providers,
// or wall-clock IO.
type HermesStatusModel struct {
	ModelName       string
	ContextTokens   int
	ContextLength   int
	SessionDuration int64
	// PromptElapsed is the per-prompt elapsed time in seconds. Only rendered
	// in the wide tier when HasPromptElapsed is true.
	PromptElapsed    int64
	PromptLive       bool
	HasPromptElapsed bool
}

// HermesStatusContextSeverity classifies the context-usage percent into the
// same five buckets Hermes' status bar uses to colour the percentage label.
type HermesStatusContextSeverity int

const (
	HermesStatusContextDim HermesStatusContextSeverity = iota
	HermesStatusContextGood
	HermesStatusContextWarn
	HermesStatusContextBad
	HermesStatusContextCritical
)

// HermesStatusBarContextSeverity mirrors Hermes' _status_bar_context_style:
// nil → dim, <50 good, 50–80 warn, 81–94 bad, ≥95 critical.
func HermesStatusBarContextSeverity(percent *int) HermesStatusContextSeverity {
	if percent == nil {
		return HermesStatusContextDim
	}
	p := *percent
	switch {
	case p >= 95:
		return HermesStatusContextCritical
	case p > 80:
		return HermesStatusContextBad
	case p >= 50:
		return HermesStatusContextWarn
	default:
		return HermesStatusContextGood
	}
}

// RenderHermesStatusBar renders the single-line Hermes-compatible footer for
// the given width. Width tiers match cli.py:_build_status_bar_text:
// <52 columns drops to model+duration, <76 columns adds context percent, and
// ≥76 columns shows the full model/context/percent/duration/prompt-elapsed
// layout. Output is trimmed with an ellipsis so it never wraps.
func RenderHermesStatusBar(model HermesStatusModel, width int) string {
	if width <= 0 {
		return ""
	}
	short := hermesStatusShortModel(model.ModelName)
	duration := hermesStatusDurationLabel(model.SessionDuration)
	percent := hermesStatusPercent(model.ContextTokens, model.ContextLength)
	percentLabel := "--"
	if percent != nil {
		percentLabel = fmt.Sprintf("%d%%", *percent)
	}

	var line string
	switch {
	case width < 52:
		line = fmt.Sprintf("⚕ %s · %s", short, duration)
	case width < 76:
		line = fmt.Sprintf("⚕ %s · %s · %s", short, percentLabel, duration)
	default:
		var contextLabel string
		if model.ContextLength > 0 {
			contextLabel = fmt.Sprintf("%s/%s",
				hermesStatusFormatTokenCount(model.ContextTokens),
				hermesStatusFormatContextLength(model.ContextLength),
			)
		} else {
			contextLabel = "ctx --"
		}
		parts := []string{
			"⚕ " + short,
			contextLabel,
			percentLabel,
			duration,
		}
		if model.HasPromptElapsed {
			parts = append(parts, hermesStatusPromptElapsed(model.PromptElapsed, model.PromptLive))
		}
		line = strings.Join(parts, " │ ")
	}

	return hermesStatusTrimToWidth(line, width)
}

func hermesStatusShortModel(name string) string {
	if name == "" {
		return "unknown"
	}
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	if strings.HasSuffix(name, ".gguf") {
		name = strings.TrimSuffix(name, ".gguf")
	}
	if lipgloss.Width(name) > 26 {
		name = hermesStatusTrimToWidth(name, 26)
	}
	return name
}

func hermesStatusDurationLabel(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutesF := float64(seconds) / 60
	if minutesF < 60 {
		return fmt.Sprintf("%dm", roundHalfAwayFromZero(minutesF))
	}
	hoursF := minutesF / 60
	if hoursF < 24 {
		hours := int(hoursF)
		remMin := int(minutesF) % 60
		if remMin == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, remMin)
	}
	days := hoursF / 24
	return fmt.Sprintf("%.1fd", days)
}

func roundHalfAwayFromZero(v float64) int {
	if v >= 0 {
		return int(v + 0.5)
	}
	return int(v - 0.5)
}

func hermesStatusPercent(tokens, length int) *int {
	if length <= 0 {
		return nil
	}
	pct := int((float64(tokens) / float64(length) * 100) + 0.5)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return &pct
}

func hermesStatusFormatTokenCount(value int) string {
	abs := value
	sign := ""
	if abs < 0 {
		abs = -abs
		sign = "-"
	}
	if abs < 1_000 {
		return fmt.Sprintf("%s%d", sign, abs)
	}
	for _, unit := range []struct {
		threshold int
		suffix    string
	}{
		{1_000_000_000, "B"},
		{1_000_000, "M"},
		{1_000, "K"},
	} {
		if abs >= unit.threshold {
			scaled := float64(abs) / float64(unit.threshold)
			var text string
			switch {
			case scaled < 10:
				text = fmt.Sprintf("%.2f", scaled)
			case scaled < 100:
				text = fmt.Sprintf("%.1f", scaled)
			default:
				text = fmt.Sprintf("%.0f", scaled)
			}
			text = trimTrailingZeros(text)
			return sign + text + unit.suffix
		}
	}
	return fmt.Sprintf("%s%d", sign, abs)
}

func hermesStatusFormatContextLength(tokens int) string {
	if tokens >= 1_000_000 {
		val := float64(tokens) / 1_000_000
		rounded := float64(int(val + 0.5))
		if abs(val-rounded) < 0.05 {
			return fmt.Sprintf("%dM", int(rounded))
		}
		return fmt.Sprintf("%.1fM", val)
	}
	if tokens >= 1_000 {
		val := float64(tokens) / 1_000
		rounded := float64(int(val + 0.5))
		if abs(val-rounded) < 0.05 {
			return fmt.Sprintf("%dK", int(rounded))
		}
		return fmt.Sprintf("%.1fK", val)
	}
	return fmt.Sprintf("%d", tokens)
}

func hermesStatusPromptElapsed(seconds int64, live bool) string {
	emoji := "⏲"
	if live {
		emoji = "⏱"
	}
	if seconds <= 0 {
		return emoji + " 0s"
	}
	days := seconds / 86400
	rem := seconds % 86400
	hours := rem / 3600
	rem %= 3600
	minutes := rem / 60
	secs := rem % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%s %dd %dh %dm", emoji, days, hours, minutes)
	case hours > 0:
		if secs > 0 {
			return fmt.Sprintf("%s %dh %dm %ds", emoji, hours, minutes, secs)
		}
		return fmt.Sprintf("%s %dh %dm", emoji, hours, minutes)
	case minutes > 0:
		if secs > 0 {
			return fmt.Sprintf("%s %dm %ds", emoji, minutes, secs)
		}
		return fmt.Sprintf("%s %dm", emoji, minutes)
	default:
		return fmt.Sprintf("%s %ds", emoji, secs)
	}
}

func hermesStatusTrimToWidth(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= maxWidth {
		return text
	}
	const ellipsis = "..."
	ellipsisWidth := lipgloss.Width(ellipsis)
	if maxWidth <= ellipsisWidth {
		out := ""
		w := 0
		for _, r := range ellipsis {
			rw := lipgloss.Width(string(r))
			if w+rw > maxWidth {
				break
			}
			out += string(r)
			w += rw
		}
		return out
	}
	var b strings.Builder
	used := 0
	for _, r := range text {
		rw := lipgloss.Width(string(r))
		if used+rw+ellipsisWidth > maxWidth {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return strings.TrimRight(b.String(), " \t") + ellipsis
}

func trimTrailingZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
