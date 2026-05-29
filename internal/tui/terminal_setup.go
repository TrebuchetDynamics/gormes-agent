package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal"

type TerminalSetupFileOps = terminal.TerminalSetupFileOps
type TerminalSetupOptions = terminal.TerminalSetupOptions
type TerminalSetupResult = terminal.TerminalSetupResult
type TruecolorResult = terminal.TruecolorResult
type TerminalParityHint = terminal.TerminalParityHint

func DetectVSCodeLikeTerminal(env map[string]string) string {
	return terminal.DetectVSCodeLikeTerminal(env)
}

func VSCodeStyleConfigDir(app, platform string, env map[string]string, home string) string {
	return terminal.VSCodeStyleConfigDir(app, platform, env, home)
}

func StripJSONComments(input string) string {
	return terminal.StripJSONComments(input)
}

func ConfigureDetectedTerminalKeybindings(opts TerminalSetupOptions) TerminalSetupResult {
	return terminal.ConfigureDetectedTerminalKeybindings(opts)
}

func ConfigureTerminalKeybindings(kind string, opts TerminalSetupOptions) TerminalSetupResult {
	return terminal.ConfigureTerminalKeybindings(kind, opts)
}

func ShouldPromptForTerminalSetup(opts TerminalSetupOptions) bool {
	return terminal.ShouldPromptForTerminalSetup(opts)
}

func TruecolorDecision(env map[string]string) TruecolorResult {
	return terminal.TruecolorDecision(env)
}

func TerminalParityHints(env map[string]string, opts TerminalSetupOptions) []TerminalParityHint {
	return terminal.TerminalParityHints(env, opts)
}
