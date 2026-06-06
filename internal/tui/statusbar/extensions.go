package statusbar

import (
	"fmt"
	"math"
	"strings"
)

// faceTickerIndicators maps task states to their cycling face indicators.
// These cycle based on frame number to provide visual feedback.
var faceTickerIndicators = []string{"⚕", "🌀", "🤔", "✨", "🍵", "🔮"}

// RenderFaceTicker returns a cycling status indicator based on state and frame.
// The indicator cycles through ⚕🌀🤔✨🍵🔮 based on frame mod 6. Different states
// may map to different starting positions in the cycle.
func RenderFaceTicker(state string, frame int) string {
	return faceTickerIndicators[faceTickerPosition(state, frame)]
}

func faceTickerPosition(state string, frame int) int {
	return positiveModulo(stateOffset(state)+frame, len(faceTickerIndicators))
}

func stateOffset(state string) int {
	state = normalizeFaceTickerState(state)
	switch state {
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
		for _, c := range state {
			hash += int(c)
		}
		return positiveModulo(hash, len(faceTickerIndicators))
	}
}

func normalizeFaceTickerState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

func positiveModulo(value, modulus int) int {
	if modulus <= 0 {
		return 0
	}
	position := value % modulus
	if position < 0 {
		position += modulus
	}
	return position
}

// ContextBarWidth is the default width of the extension context bar.
const ContextBarWidth = 10

// ClampContextPercent normalizes context percentages before rendering bars or
// labels so both surfaces report the same bounded value.
func ClampContextPercent(pct float64) float64 {
	if math.IsNaN(pct) || pct < 0 {
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
	filled := contextBarFilledCells(pct)
	return strings.Repeat("█", filled) + strings.Repeat("░", ContextBarWidth-filled)
}

func contextBarFilledCells(pct float64) int {
	pct = ClampContextPercent(pct)
	filled := int((pct/100)*float64(ContextBarWidth) + 0.5)
	if filled < 0 {
		return 0
	}
	if filled > ContextBarWidth {
		return ContextBarWidth
	}
	return filled
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
