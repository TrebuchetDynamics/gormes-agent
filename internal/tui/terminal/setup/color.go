package setup

import "strings"

func TruecolorDecision(env map[string]string) TruecolorResult {
	if envHas(env, "NO_COLOR") {
		return TruecolorResult{Evidence: "tui_terminal_truecolor_disabled"}
	}
	switch strings.ToLower(strings.TrimSpace(envValue(env, "HERMES_TUI_TRUECOLOR"))) {
	case "1", "true", "yes", "on":
		set := map[string]string{"FORCE_COLOR": "3"}
		if envValue(env, "COLORTERM") == "" {
			set["COLORTERM"] = "truecolor"
		}
		return TruecolorResult{
			Force: true,
			Set:   set,
		}
	case "0", "false", "no", "off":
		return TruecolorResult{Evidence: "tui_terminal_truecolor_disabled"}
	default:
		if shouldDowngradeAppleTerminalTruecolor(env) {
			unset := []string{"COLORTERM"}
			if envValue(env, "FORCE_COLOR") == "3" {
				unset = append(unset, "FORCE_COLOR")
			}
			return TruecolorResult{Unset: unset, Evidence: "tui_terminal_truecolor_downgraded"}
		}
		return TruecolorResult{}
	}
}

func shouldDowngradeAppleTerminalTruecolor(env map[string]string) bool {
	return envValue(env, "TERM_PROGRAM") == "Apple_Terminal" && terminalAdvertisesTruecolor(env)
}

func terminalAdvertisesTruecolor(env map[string]string) bool {
	switch strings.ToLower(envValue(env, "COLORTERM")) {
	case "truecolor", "24bit":
		return true
	}
	return envValue(env, "FORCE_COLOR") == "3"
}

func TerminalParityHints(env map[string]string, opts TerminalSetupOptions) []TerminalParityHint {
	var hints []TerminalParityHint
	if envValue(env, "TERM_PROGRAM") == "Apple_Terminal" {
		hints = append(hints, TerminalParityHint{Key: "apple-terminal", Message: "Apple Terminal may need explicit truecolor configuration."})
	}
	if isRemoteTerminal(env) {
		hints = append(hints, TerminalParityHint{Key: "remote", Message: "Remote terminals may not support local clipboard integration."})
	}
	if envValue(env, "TMUX") != "" {
		hints = append(hints, TerminalParityHint{Key: "tmux", Message: "tmux requires OSC52 passthrough for clipboard queries."})
	}
	setupOpts := opts
	setupOpts.Env = env
	if ShouldPromptForTerminalSetup(setupOpts) {
		hints = append(hints, TerminalParityHint{Key: "ide-setup", Message: "VS Code-family terminal keybindings are not configured."})
	}
	return hints
}
