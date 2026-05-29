package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal"

type ClipboardCommandOptions = terminal.ClipboardCommandOptions
type ClipboardRunResult = terminal.ClipboardRunResult
type ClipboardRunFunc = terminal.ClipboardRunFunc
type ClipboardReadRequest = terminal.ClipboardReadRequest
type ClipboardTextResult = terminal.ClipboardTextResult
type ClipboardStartFunc = terminal.ClipboardStartFunc
type ClipboardWriteRequest = terminal.ClipboardWriteRequest
type ClipboardWriteResult = terminal.ClipboardWriteResult

// ReadClipboardText mirrors Hermes ui-tui clipboard fallback order while
// keeping command execution injectable.
func ReadClipboardText(req ClipboardReadRequest) ClipboardTextResult {
	return terminal.ReadClipboardText(req)
}

// IsUsableClipboardText rejects empty or binary-looking clipboard payloads.
func IsUsableClipboardText(text string) bool {
	return terminal.IsUsableClipboardText(text)
}

func WriteClipboardText(req ClipboardWriteRequest) ClipboardWriteResult {
	return terminal.WriteClipboardText(req)
}
