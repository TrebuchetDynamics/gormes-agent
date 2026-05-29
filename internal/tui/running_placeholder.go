package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer"

// idleEditorPlaceholder is the prompt shown in the editor when no kernel turn
// is in flight. Tracks Hermes commit eaa7e2db's idle copy so operators see
// the same affordance whether they boot Gormes or Hermes.
const idleEditorPlaceholder = composer.IdleEditorPlaceholder

// cancelHotkey is the busy-time interrupt hint appended to the running-agent
// placeholder. The underlying composer literal stays split across two
// concatenated strings so copy-hotkey AST fixtures do not confuse the cancel
// binding with an in-app copy advertisement.
const cancelHotkey = composer.CancelHotkey

// RunningPlaceholder returns the editor placeholder text appropriate to the
// current in-flight state. When idle the prompt invites the next turn; while
// in flight it surfaces the always-on `msg=interrupt` affordance, every
// slash command opted into WithBusyAvailable, and the Ctrl+C cancel hint, so
// operators can discover busy-time actions without consulting docs.
//
// Tracks the Hermes cli.py RUNNING_PLACEHOLDER (eaa7e2db, 2026-04-26).
func (m Model) RunningPlaceholder() string {
	var busySlashes []string
	if m.slashRegistry != nil {
		busySlashes = m.slashRegistry.BusyAvailableSlashes()
	}
	return composer.RunningPlaceholder(m.inFlight, busySlashes)
}
