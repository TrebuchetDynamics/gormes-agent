package terminal

import clip "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/clipboard"

const OSC52ClipboardQuery = clip.OSC52ClipboardQuery

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
