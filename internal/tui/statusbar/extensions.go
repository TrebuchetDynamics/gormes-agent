package statusbar

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar/contextmeter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar/indicators"
)

// ContextBarWidth is the default width of the extension context bar.
const ContextBarWidth = contextmeter.BarWidth

// RenderFaceTicker returns a cycling status indicator based on state and frame.
func RenderFaceTicker(state string, frame int) string {
	return indicators.RenderFaceTicker(state, frame)
}

// ClampContextPercent normalizes context percentages before rendering bars or
// labels so both surfaces report the same bounded value.
func ClampContextPercent(pct float64) float64 {
	return contextmeter.ClampPercent(pct)
}

// RenderContextBar renders a filled █ and empty ░ bar representing the
// percentage with severity-based coloring semantics. It returns a 10-character
// string like "████░░░░░░" for 40%.
func RenderContextBar(pct float64) string {
	return indicators.RenderContextBar(pct)
}

// ContextBarSeverity returns the severity level for a given percentage.
func ContextBarSeverity(pct float64) HermesContextSeverity {
	return indicators.ContextBarSeverity(pct)
}

// RenderContextBarWithLabel renders a context bar with a percentage label.
func RenderContextBarWithLabel(pct float64) string {
	return indicators.RenderContextBarWithLabel(pct)
}
