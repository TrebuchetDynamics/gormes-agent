package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/input"

// StripLeakedBracketedPasteWrappers removes bracketed-paste markers that
// terminals can leak into CLI buffers while preserving ordinary inline text.
func StripLeakedBracketedPasteWrappers(text string) string {
	return input.StripLeakedBracketedPasteWrappers(text)
}
