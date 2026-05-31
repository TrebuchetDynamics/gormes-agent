package setup

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/truecolor"
)

func TruecolorDecision(env map[string]string) TruecolorResult {
	return truecolor.Decision(env)
}

func TerminalParityHints(env map[string]string, opts TerminalSetupOptions) []TerminalParityHint {
	var hints []TerminalParityHint
	if envvars.Value(env, envvars.TermProgram) == envvars.AppleTerminalProgram {
		hints = append(hints, TerminalParityHint{Key: "apple-terminal", Message: "Apple Terminal may need explicit truecolor configuration."})
	}
	if envvars.IsRemote(env) {
		hints = append(hints, TerminalParityHint{Key: "remote", Message: "Remote terminals may not support local clipboard integration."})
	}
	if envvars.Value(env, envvars.TMUX) != "" {
		hints = append(hints, TerminalParityHint{Key: "tmux", Message: "tmux requires OSC52 passthrough for clipboard queries."})
	}
	setupOpts := opts
	setupOpts.Env = env
	if ShouldPromptForTerminalSetup(setupOpts) {
		hints = append(hints, TerminalParityHint{Key: "ide-setup", Message: "VS Code-family terminal keybindings are not configured."})
	}
	return hints
}
