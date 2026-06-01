package terminal

import (
	tea "github.com/charmbracelet/bubbletea"

	clip "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/clipboard"
	mousepkg "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/mouse"
	selectionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/selection"
	setuppkg "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup"
)

const (
	OSC52ClipboardQuery = clip.OSC52ClipboardQuery
	MouseSlashUsage     = mousepkg.MouseSlashUsage
	NativeSelectionHelp = selectionpkg.NativeSelectionHelp
)

type ClipboardCommandOptions = clip.ClipboardCommandOptions
type ClipboardRunResult = clip.ClipboardRunResult
type ClipboardRunFunc = clip.ClipboardRunFunc
type ClipboardReadRequest = clip.ClipboardReadRequest
type ClipboardTextResult = clip.ClipboardTextResult
type ClipboardStartFunc = clip.ClipboardStartFunc
type ClipboardWriteRequest = clip.ClipboardWriteRequest
type ClipboardWriteResult = clip.ClipboardWriteResult
type OSC52Response = clip.OSC52Response
type OSC52SendFunc = clip.OSC52SendFunc

type MouseSlashResult = mousepkg.MouseSlashResult
type MouseSlashDecision = mousepkg.MouseSlashDecision

type TerminalSetupFileOps = setuppkg.TerminalSetupFileOps
type TerminalSetupOptions = setuppkg.TerminalSetupOptions
type TerminalSetupResult = setuppkg.TerminalSetupResult
type TruecolorResult = setuppkg.TruecolorResult
type TerminalParityHint = setuppkg.TerminalParityHint

// ReadClipboardText mirrors Hermes ui-tui clipboard fallback order while
// keeping command execution injectable.
func ReadClipboardText(req ClipboardReadRequest) ClipboardTextResult {
	return clip.ReadClipboardText(req)
}

// IsUsableClipboardText rejects empty or binary-looking clipboard payloads.
func IsUsableClipboardText(text string) bool {
	return clip.IsUsableClipboardText(text)
}

func WriteClipboardText(req ClipboardWriteRequest) ClipboardWriteResult {
	return clip.WriteClipboardText(req)
}

func BuildOSC52ClipboardQuery(env map[string]string) string {
	return clip.BuildOSC52ClipboardQuery(env)
}

func ParseOSC52ClipboardData(data string) (string, bool) {
	return clip.ParseOSC52ClipboardData(data)
}

func ReadOSC52Clipboard(env map[string]string, send OSC52SendFunc, flush func() error) ClipboardTextResult {
	return clip.ReadOSC52Clipboard(env, send, flush)
}

func HandleMouseSlash(input string, current bool) MouseSlashDecision {
	return mousepkg.HandleMouseSlash(input, current)
}

func ParseMouseTrackingSlash(input string, current bool) MouseSlashResult {
	return mousepkg.ParseMouseTrackingSlash(input, current)
}

func DefaultMouseModeCmd(enabled bool) tea.Cmd {
	return mousepkg.DefaultMouseModeCmd(enabled)
}

func SelectionHelpLine() string {
	return selectionpkg.SelectionHelpLine()
}

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
