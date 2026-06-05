package setup

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/hints"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/truecolor"
)

func TruecolorDecision(env map[string]string) TruecolorResult {
	return truecolor.Decision(env)
}

func TerminalParityHints(env map[string]string, opts TerminalSetupOptions) []TerminalParityHint {
	return hints.ParityHints(env, func(env map[string]string) bool {
		setupOpts := opts
		setupOpts.Env = env
		return ShouldPromptForTerminalSetup(setupOpts)
	})
}
