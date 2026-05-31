package osc52

import (
	"encoding/base64"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/cliptext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

const ClipboardQuery = "\x1b]52;c;?\x07"

type Response struct {
	Code int
	Type string
	Data string
}

type SendFunc func(query string) (Response, error)

func BuildClipboardQuery(env map[string]string) string {
	if envvars.Value(env, envvars.TMUX) == "" {
		return ClipboardQuery
	}
	escaped := strings.ReplaceAll(ClipboardQuery, "\x1b", "\x1b\x1b")
	return "\x1bPtmux;" + escaped + "\x1b\\"
}

func ParseClipboardData(data string) (string, bool) {
	_, encoded, ok := strings.Cut(data, ";")
	if !ok || encoded == "" || encoded == "?" {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", false
	}
	text := string(decoded)
	if !cliptext.IsUsable(text) {
		return "", false
	}
	return text, true
}

func ReadClipboard(env map[string]string, send SendFunc, flush func() error) cliptext.Result {
	if send == nil {
		return cliptext.Result{Evidence: "tui_terminal_osc52_unavailable"}
	}
	if flush != nil {
		if err := flush(); err != nil {
			return cliptext.Result{Evidence: "tui_terminal_osc52_unavailable"}
		}
	}
	response, err := send(BuildClipboardQuery(env))
	if err != nil || response.Code != 52 || response.Type != "osc" {
		return cliptext.Result{Evidence: "tui_terminal_osc52_unavailable"}
	}
	text, ok := ParseClipboardData(response.Data)
	if !ok {
		return cliptext.Result{Evidence: "tui_terminal_osc52_unavailable"}
	}
	return cliptext.Result{Text: text, OK: true}
}
