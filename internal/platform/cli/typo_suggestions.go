package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands"

// TypoSuggestion is the pre-Cobra extension point for deterministic,
// secret-safe guidance on removed command spellings.
func TypoSuggestion(args []string) (string, bool) { return commands.TypoSuggestion(args) }
