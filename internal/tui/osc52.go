package tui

import (
	"encoding/base64"
	"strings"
)

const OSC52ClipboardQuery = "\x1b]52;c;?\x07"

type OSC52Response struct {
	Code int
	Type string
	Data string
}

type OSC52SendFunc func(query string) (OSC52Response, error)

func BuildOSC52ClipboardQuery(env map[string]string) string {
	if envValue(env, "TMUX") == "" {
		return OSC52ClipboardQuery
	}
	escaped := strings.ReplaceAll(OSC52ClipboardQuery, "\x1b", "\x1b\x1b")
	return "\x1bPtmux;" + escaped + "\x1b\\"
}

func ParseOSC52ClipboardData(data string) (string, bool) {
	_, encoded, ok := strings.Cut(data, ";")
	if !ok || encoded == "" || encoded == "?" {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", false
	}
	text := string(decoded)
	if !IsUsableClipboardText(text) {
		return "", false
	}
	return text, true
}

func ReadOSC52Clipboard(env map[string]string, send OSC52SendFunc, flush func() error) ClipboardTextResult {
	if send == nil {
		return ClipboardTextResult{Evidence: "tui_terminal_osc52_unavailable"}
	}
	if flush != nil {
		if err := flush(); err != nil {
			return ClipboardTextResult{Evidence: "tui_terminal_osc52_unavailable"}
		}
	}
	response, err := send(BuildOSC52ClipboardQuery(env))
	if err != nil || response.Code != 52 || response.Type != "osc" {
		return ClipboardTextResult{Evidence: "tui_terminal_osc52_unavailable"}
	}
	text, ok := ParseOSC52ClipboardData(response.Data)
	if !ok {
		return ClipboardTextResult{Evidence: "tui_terminal_osc52_unavailable"}
	}
	return ClipboardTextResult{Text: text, OK: true}
}
