package statusbar

import (
	"fmt"
	"strings"
)

// faceTickerIndicators maps task states to their cycling face indicators.
// These cycle based on frame number to provide visual feedback.
var faceTickerIndicators = []string{"⚕", "🌀", "🤔", "✨", "🍵", "🔮"}

// RenderFaceTicker returns a cycling status indicator based on state and frame.
// The indicator cycles through ⚕🌀🤔✨🍵🔮 based on frame mod 6. Different states
// may map to different starting positions in the cycle.
func RenderFaceTicker(state string, frame int) string {
	startOffset := stateOffset(state)
	position := (startOffset + frame) % len(faceTickerIndicators)
	if position < 0 {
		position = 0
	}
	return faceTickerIndicators[position]
}

func stateOffset(state string) int {
	switch strings.ToLower(state) {
	case "idle":
		return 0
	case "reasoning":
		return 1
	case "working":
		return 2
	case "waiting":
		return 3
	case "ready":
		return 4
	case "break":
		return 5
	case "magic":
		return 0
	case "error":
		return 2
	default:
		hash := 0
		for _, c := range strings.ToLower(state) {
			hash += int(c)
		}
		return hash % len(faceTickerIndicators)
	}
}

// ContextBarWidth is the default width of the extension context bar.
const ContextBarWidth = 10

// ClampContextPercent normalizes context percentages before rendering bars or
// labels so both surfaces report the same bounded value.
func ClampContextPercent(pct float64) float64 {
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// RenderContextBar renders a filled █ and empty ░ bar representing the
// percentage with severity-based coloring semantics. It returns a 10-character
// string like "████░░░░░░" for 40%.
func RenderContextBar(pct float64) string {
	pct = ClampContextPercent(pct)

	width := ContextBarWidth
	filled := int((pct / 100) * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}

	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// ContextBarSeverity returns the severity level for a given percentage.
func ContextBarSeverity(pct float64) HermesContextSeverity {
	pct = ClampContextPercent(pct)
	switch {
	case pct >= 95:
		return HermesContextCritical
	case pct > 80:
		return HermesContextBad
	case pct >= 50:
		return HermesContextWarn
	default:
		return HermesContextGood
	}
}

// RenderContextBarWithLabel renders a context bar with a percentage label.
func RenderContextBarWithLabel(pct float64) string {
	pct = ClampContextPercent(pct)
	bar := RenderContextBar(pct)
	return fmt.Sprintf("[%s] %d%%", bar, int(pct))
}
