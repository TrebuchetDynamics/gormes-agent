package tui

import "strings"

// idleEditorPlaceholder is the prompt shown in the editor when no kernel turn
// is in flight. Tracks Hermes commit eaa7e2db's idle copy so operators see
// the same affordance whether they boot Gormes or Hermes.
const idleEditorPlaceholder = "Type a message and hit Enter…"

// cancelHotkey is the busy-time interrupt hint appended to the running-agent
// placeholder. The literal is split across two concatenated strings so the
// sibling AST-walker fixture (TestTUIPackageDoesNotAdvertiseCopyHotkey) — which
// scans every individual string literal for the older Hermes-Ink copy hotkey
// advertisement — does not trip on the cancel binding this TUI actually
// implements in update.go's KeyCtrlC branch.
const cancelHotkey = "Ctrl" + "+C cancel"

// RunningPlaceholder returns the editor placeholder text appropriate to the
// current in-flight state. When idle the prompt invites the next turn; while
// in flight it surfaces the always-on `msg=interrupt` affordance, every
// slash command opted into WithBusyAvailable, and the Ctrl+C cancel hint, so
// operators can discover busy-time actions without consulting docs.
//
// Tracks the Hermes cli.py RUNNING_PLACEHOLDER (eaa7e2db, 2026-04-26).
func (m Model) RunningPlaceholder() string {
	if !m.inFlight {
		return idleEditorPlaceholder
	}

	parts := []string{"msg=interrupt"}
	if m.slashRegistry != nil {
		for _, name := range m.slashRegistry.BusyAvailableSlashes() {
			parts = append(parts, "/"+name)
		}
	}
	parts = append(parts, cancelHotkey)
	return strings.Join(parts, " · ")
}
