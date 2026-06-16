package placeholder

import "strings"

// IdleEditorPlaceholder is the prompt shown in the editor when no kernel turn
// is in flight. Tracks Hermes commit eaa7e2db's idle copy so operators see
// the same affordance whether they boot Gormes or Hermes.
const IdleEditorPlaceholder = "Type a message and hit Enter…"

// CancelHotkey is the busy-time interrupt hint appended to the running-agent
// placeholder.
const CancelHotkey = "Ctrl" + "+C cancel"

// RunningPlaceholder returns the editor placeholder text appropriate to the
// current in-flight state. When idle the prompt invites the next turn; while
// in flight it surfaces the always-on interrupt affordance, caller-provided
// busy slash commands, and the Ctrl+C cancel hint.
func RunningPlaceholder(inFlight bool, busySlashes []string) string {
	if !inFlight {
		return IdleEditorPlaceholder
	}

	parts := []string{"msg=interrupt"}
	for _, name := range busySlashes {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		parts = append(parts, "/"+strings.TrimPrefix(name, "/"))
	}
	parts = append(parts, CancelHotkey)
	return strings.Join(parts, " · ")
}
