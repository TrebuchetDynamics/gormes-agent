package truecolor

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

type Result struct {
	Force    bool
	Set      map[string]string
	Unset    []string
	Evidence string
}

func Decision(env map[string]string) Result {
	if envvars.Has(env, envvars.NoColor) {
		return Result{Evidence: "tui_terminal_truecolor_disabled"}
	}
	switch strings.ToLower(strings.TrimSpace(envvars.Value(env, envvars.HermesTUITruecolor))) {
	case "1", "true", "yes", "on":
		set := map[string]string{envvars.ForceColor: envvars.ForceColorTruecolor}
		if envvars.Value(env, envvars.ColorTerm) == "" {
			set[envvars.ColorTerm] = envvars.Truecolor
		}
		return Result{
			Force: true,
			Set:   set,
		}
	case "0", "false", "no", "off":
		return Result{Evidence: "tui_terminal_truecolor_disabled"}
	default:
		if shouldDowngradeAppleTerminal(env) {
			unset := []string{envvars.ColorTerm}
			if envvars.Value(env, envvars.ForceColor) == envvars.ForceColorTruecolor {
				unset = append(unset, envvars.ForceColor)
			}
			return Result{Unset: unset, Evidence: "tui_terminal_truecolor_downgraded"}
		}
		return Result{}
	}
}

func shouldDowngradeAppleTerminal(env map[string]string) bool {
	return envvars.Value(env, envvars.TermProgram) == envvars.AppleTerminalProgram && terminalAdvertisesTruecolor(env)
}

func terminalAdvertisesTruecolor(env map[string]string) bool {
	switch strings.ToLower(envvars.Value(env, envvars.ColorTerm)) {
	case envvars.Truecolor, envvars.Truecolor24Bit:
		return true
	}
	return envvars.Value(env, envvars.ForceColor) == envvars.ForceColorTruecolor
}
