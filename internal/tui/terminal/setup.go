package terminal

import setuppkg "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup"

type TerminalSetupFileOps = setuppkg.TerminalSetupFileOps
type TerminalSetupOptions = setuppkg.TerminalSetupOptions
type TerminalSetupResult = setuppkg.TerminalSetupResult
type TruecolorResult = setuppkg.TruecolorResult
type TerminalParityHint = setuppkg.TerminalParityHint

func DetectVSCodeLikeTerminal(env map[string]string) string {
	return setuppkg.DetectVSCodeLikeTerminal(env)
}

func VSCodeStyleConfigDir(app, platform string, env map[string]string, home string) string {
	return setuppkg.VSCodeStyleConfigDir(app, platform, env, home)
}

func StripJSONComments(input string) string {
	return setuppkg.StripJSONComments(input)
}

func ConfigureDetectedTerminalKeybindings(opts TerminalSetupOptions) TerminalSetupResult {
	return setuppkg.ConfigureDetectedTerminalKeybindings(opts)
}

func ConfigureTerminalKeybindings(kind string, opts TerminalSetupOptions) TerminalSetupResult {
	return setuppkg.ConfigureTerminalKeybindings(kind, opts)
}

func ShouldPromptForTerminalSetup(opts TerminalSetupOptions) bool {
	return setuppkg.ShouldPromptForTerminalSetup(opts)
}

func TruecolorDecision(env map[string]string) TruecolorResult {
	return setuppkg.TruecolorDecision(env)
}

func TerminalParityHints(env map[string]string, opts TerminalSetupOptions) []TerminalParityHint {
	return setuppkg.TerminalParityHints(env, opts)
}
