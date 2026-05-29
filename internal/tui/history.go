package tui

import drafthistory "github.com/TrebuchetDynamics/gormes-agent/internal/tui/history"

// HermesHistory is re-exported from the draft history module to preserve the public TUI seam.
type HermesHistory = drafthistory.HermesHistory

// NewHermesHistory returns an empty Hermes composer draft history.
func NewHermesHistory() *HermesHistory {
	return drafthistory.NewHermesHistory()
}
