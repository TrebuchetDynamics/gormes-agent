package contract

import contractslash "github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker/contract/slash"

// SlashArgument returns the free-form model argument after the slash command.
func SlashArgument(input string) string {
	return contractslash.Argument(input)
}
