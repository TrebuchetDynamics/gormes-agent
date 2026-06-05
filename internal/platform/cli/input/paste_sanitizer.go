package input

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/input/sanitization"

// StripLeakedBracketedPasteWrappers removes bracketed-paste markers that
// terminals can leak into CLI buffers while preserving ordinary inline text.
func StripLeakedBracketedPasteWrappers(text string) string {
	return sanitization.StripLeakedBracketedPasteWrappers(text)
}
