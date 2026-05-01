// Package tui — Hermes-compatible status bar extensions.
//
// This module provides FaceTicker and ContextBar renderers that extend
// the HermesStatusBar with animated indicators and severity-colored
// context usage bars.
package tui

import (
	"fmt"
	"strings"
)

// faceTickerIndicators maps task states to their cycling face indicators.
// These cycle based on frame number to provide visual feedback.
var faceTickerIndicators = []string{"⚕", "🌀", "🤔", "✨", "🍵", "🔮"}

// RenderFaceTicker returns a cycling status indicator based on state and frame.
// The indicator cycles through ⚕🌀🤔✨🍵🔮 based on frame mod 6.
// Different states may map to different starting positions in the cycle.
func RenderFaceTicker(state string, frame int) string {
	// Map state to starting position in the cycle
	startOffset := stateOffset(state)

	// Calculate actual position in cycle
	position := (startOffset + frame) % len(faceTickerIndicators)
	if position < 0 {
		position = 0
	}

	return faceTickerIndicators[position]
}

// stateOffset maps a state string to an offset in the face cycle.
// This allows different states to start at different points in the animation.
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
		// For unknown states, hash to a consistent offset
		hash := 0
		for _, c := range strings.ToLower(state) {
			hash += int(c)
		}
		return hash % len(faceTickerIndicators)
	}
}

// ContextBarWidth is the default width of the context bar.
const ContextBarWidth = 10

// RenderContextBar renders a filled █ and empty ░ bar representing
// the percentage with severity-based coloring semantics.
// Returns a 10-character string like "████░░░░░░" for 40%.
func RenderContextBar(pct float64) string {
	// Clamp percentage to [0, 100]
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

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
// This mirrors the HermesStatusBarContextSeverity thresholds.
func ContextBarSeverity(pct float64) HermesStatusContextSeverity {
	switch {
	case pct >= 95:
		return HermesStatusContextCritical
	case pct > 80:
		return HermesStatusContextBad
	case pct >= 50:
		return HermesStatusContextWarn
	default:
		return HermesStatusContextGood
	}
}

// RenderContextBarWithLabel renders a context bar with a percentage label.
// Format: "[████░░░░░░] 40%"
func RenderContextBarWithLabel(pct float64) string {
	bar := RenderContextBar(pct)
	return fmt.Sprintf("[%s] %d%%", bar, int(pct))
}
