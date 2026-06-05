package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/ansitext"

// StripANSIForTUI removes terminal control sequences before text enters
// cursor/source-of-truth calculations.
func StripANSIForTUI(s string) string {
	return ansitext.StripForTUI(s)
}

// SanitizeANSIForRender keeps SGR color sequences but strips cursor movement,
// OSC strings, malformed CSI, and C0 controls that can corrupt renderer state.
func SanitizeANSIForRender(s string) string {
	return ansitext.SanitizeForRender(s)
}

func HasANSI(s string) bool {
	return ansitext.HasANSI(s)
}
