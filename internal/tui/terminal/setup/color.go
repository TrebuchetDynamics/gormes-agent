package setup

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func TruecolorDecision(env map[string]string) TruecolorResult {
	if envvars.Has(env, envvars.NoColor) {
		return TruecolorResult{Evidence: "tui_terminal_truecolor_disabled"}
	}
	switch strings.ToLower(strings.TrimSpace(envvars.Value(env, envvars.HermesTUITruecolor))) {
	case "1", "true", "yes", "on":
		set := map[string]string{envvars.ForceColor: "3"}
		if envvars.Value(env, envvars.ColorTerm) == "" {
			set[envvars.ColorTerm] = "truecolor"
		}
		return TruecolorResult{
			Force: true,
			Set:   set,
		}
	case "0", "false", "no", "off":
		return TruecolorResult{Evidence: "tui_terminal_truecolor_disabled"}
	default:
		if shouldDowngradeAppleTerminalTruecolor(env) {
			unset := []string{envvars.ColorTerm}
			if envvars.Value(env, envvars.ForceColor) == "3" {
				unset = append(unset, envvars.ForceColor)
			}
			return TruecolorResult{Unset: unset, Evidence: "tui_terminal_truecolor_downgraded"}
		}
		return TruecolorResult{}
	}
}

func shouldDowngradeAppleTerminalTruecolor(env map[string]string) bool {
	return envvars.Value(env, envvars.TermProgram) == "Apple_Terminal" && terminalAdvertisesTruecolor(env)
}

func terminalAdvertisesTruecolor(env map[string]string) bool {
	switch strings.ToLower(envvars.Value(env, envvars.ColorTerm)) {
	case "truecolor", "24bit":
		return true
	}
	return envvars.Value(env, envvars.ForceColor) == "3"
}

func TerminalParityHints(env map[string]string, opts TerminalSetupOptions) []TerminalParityHint {
	var hints []TerminalParityHint
	if envvars.Value(env, envvars.TermProgram) == "Apple_Terminal" {
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
