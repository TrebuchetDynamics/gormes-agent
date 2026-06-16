package indicators

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar/contextmeter"
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

// RenderContextBar renders a filled █ and empty ░ bar representing the
// percentage with severity-based coloring semantics. It returns a 10-character
// string like "████░░░░░░" for 40%.
func RenderContextBar(pct float64) string {
	filled := contextmeter.FilledCells(pct)
	return strings.Repeat("█", filled) + strings.Repeat("░", contextmeter.BarWidth-filled)
}

// ContextBarSeverity returns the severity level for a given percentage.
func ContextBarSeverity(pct float64) contextmeter.Severity {
	return contextmeter.SeverityForFloat(pct)
}

// RenderContextBarWithLabel renders a context bar with a percentage label.
func RenderContextBarWithLabel(pct float64) string {
	pct = contextmeter.ClampPercent(pct)
	bar := RenderContextBar(pct)
	return fmt.Sprintf("[%s] %d%%", bar, int(pct))
}
