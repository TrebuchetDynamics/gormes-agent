// Package tui — Hermes-compatible status bar extensions.
//
// This module provides FaceTicker and ContextBar renderers that extend
// the HermesStatusBar with animated indicators and severity-colored
// context usage bars.
package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar"

// ContextBarWidth is the default width of the context bar.
const ContextBarWidth = statusbar.ContextBarWidth

// RenderFaceTicker returns a cycling status indicator based on state and frame.
// The indicator cycles through ⚕🌀🤔✨🍵🔮 based on frame mod 6.
// Different states may map to different starting positions in the cycle.
func RenderFaceTicker(state string, frame int) string {
	return statusbar.RenderFaceTicker(state, frame)
}

// RenderContextBar renders a filled █ and empty ░ bar representing
// the percentage with severity-based coloring semantics.
// Returns a 10-character string like "████░░░░░░" for 40%.
func RenderContextBar(pct float64) string {
	return statusbar.RenderContextBar(pct)
}

// ContextBarSeverity returns the severity level for a given percentage.
// This mirrors the HermesStatusBarContextSeverity thresholds.
func ContextBarSeverity(pct float64) HermesStatusContextSeverity {
	return statusbar.ContextBarSeverity(pct)
}

// RenderContextBarWithLabel renders a context bar with a percentage label.
// Format: "[████░░░░░░] 40%"
func RenderContextBarWithLabel(pct float64) string {
	return statusbar.RenderContextBarWithLabel(pct)
}
