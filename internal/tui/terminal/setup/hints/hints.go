package hints

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"

// Hint describes a terminal environment parity hint shown to operators.
type Hint struct {
	Key     string
	Message string
}

// MissingSetupFunc reports whether the active terminal needs IDE setup help.
type MissingSetupFunc func(env map[string]string) bool

// ParityHints returns operator-facing terminal parity hints for an environment.
func ParityHints(env map[string]string, missingSetup MissingSetupFunc) []Hint {
	var hints []Hint
	if envvars.Value(env, envvars.TermProgram) == envvars.AppleTerminalProgram {
		hints = append(hints, Hint{Key: "apple-terminal", Message: "Apple Terminal may need explicit truecolor configuration."})
	}
	if envvars.IsRemote(env) {
		hints = append(hints, Hint{Key: "remote", Message: "Remote terminals may not support local clipboard integration."})
	}
	if envvars.Value(env, envvars.TMUX) != "" {
		hints = append(hints, Hint{Key: "tmux", Message: "tmux requires OSC52 passthrough for clipboard queries."})
	}
	if missingSetup != nil && missingSetup(env) {
		hints = append(hints, Hint{Key: "ide-setup", Message: "VS Code-family terminal keybindings are not configured."})
	}
	return hints
}
