package composer

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer/placeholder"

const IdleEditorPlaceholder = placeholder.IdleEditorPlaceholder
const CancelHotkey = placeholder.CancelHotkey

func RunningPlaceholder(inFlight bool, busySlashes []string) string {
	return placeholder.RunningPlaceholder(inFlight, busySlashes)
}
