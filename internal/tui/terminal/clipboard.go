package terminal

import (
	"encoding/base64"
	"strings"
	"unicode/utf8"
)

const clipboardMaxBuffer = 4 * 1024 * 1024

const OSC52ClipboardQuery = "\x1b]52;c;?\x07"

// ClipboardCommandOptions records the process options Hermes uses for
// clipboard helper commands. Tests inspect it through injected runners; the
// TUI package never starts host clipboard processes directly.
type ClipboardCommandOptions struct {
	Encoding    string
	MaxBuffer   int
	WindowsHide bool
	Stdio       []string
}

type ClipboardRunResult struct {
	Stdout string
}

type ClipboardRunFunc func(name string, args []string, opts ClipboardCommandOptions) (ClipboardRunResult, error)

type ClipboardReadRequest struct {
	Platform string
	Env      map[string]string
	Run      ClipboardRunFunc
}

type ClipboardTextResult struct {
	Text     string
	OK       bool
	Evidence string
	Attempts []string
}

type ClipboardStartFunc func(name string, args []string, opts ClipboardCommandOptions, input string) error

type ClipboardWriteRequest struct {
	Platform string
	Text     string
	Start    ClipboardStartFunc
}

type ClipboardWriteResult struct {
	OK       bool
	Evidence string
}

type OSC52Response struct {
	Code int
	Type string
	Data string
}

type OSC52SendFunc func(query string) (OSC52Response, error)

type clipboardBackend struct {
	name string
	args []string
}

// ReadClipboardText mirrors Hermes ui-tui clipboard fallback order while
// keeping command execution injectable.
func ReadClipboardText(req ClipboardReadRequest) ClipboardTextResult {
	if req.Run == nil {
		return ClipboardTextResult{Evidence: "tui_terminal_clipboard_unavailable"}
	}
	for _, backend := range clipboardBackends(req.Platform, req.Env) {
		result, err := req.Run(backend.name, backend.args, clipboardReadOptions())
		if err != nil {
			continue
		}
		if !IsUsableClipboardText(result.Stdout) {
			return ClipboardTextResult{
				Evidence: "tui_terminal_clipboard_unusable",
				Attempts: []string{backend.name},
			}
		}
		return ClipboardTextResult{
			Text:     result.Stdout,
			OK:       true,
			Attempts: []string{backend.name},
		}
	}
	return ClipboardTextResult{Evidence: "tui_terminal_clipboard_unavailable"}
}

func clipboardBackends(platform string, env map[string]string) []clipboardBackend {
	powershellArgs := []string{"-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw"}
	switch platform {
	case "darwin":
		return []clipboardBackend{{name: "pbpaste"}}
	case "win32":
		return []clipboardBackend{{name: "powershell", args: powershellArgs}}
	case "linux":
		var backends []clipboardBackend
		if envValue(env, "WSL_INTEROP") != "" {
			backends = append(backends, clipboardBackend{name: "powershell.exe", args: powershellArgs})
		}
		if envValue(env, "WAYLAND_DISPLAY") != "" {
			backends = append(backends, clipboardBackend{name: "wl-paste", args: []string{"--type", "text"}})
		}
		backends = append(backends, clipboardBackend{name: "xclip", args: []string{"-selection", "clipboard", "-out"}})
		return backends
	default:
		return nil
	}
}

func clipboardReadOptions() ClipboardCommandOptions {
	return ClipboardCommandOptions{
		Encoding:    "utf8",
		MaxBuffer:   clipboardMaxBuffer,
		WindowsHide: true,
	}
}

// IsUsableClipboardText rejects empty or binary-looking clipboard payloads.
func IsUsableClipboardText(text string) bool {
	if strings.TrimSpace(text) == "" || !utf8.ValidString(text) {
		return false
	}
	for _, r := range text {
		switch r {
		case '\t', '\n', '\r':
			continue
		case '\uFFFD':
			return false
		}
		if r < 0x20 {
			return false
		}
	}
	return true
}

func WriteClipboardText(req ClipboardWriteRequest) ClipboardWriteResult {
	if req.Platform != "darwin" {
		return ClipboardWriteResult{Evidence: "tui_terminal_clipboard_write_unsupported"}
	}
	if req.Start == nil {
		return ClipboardWriteResult{Evidence: "tui_terminal_clipboard_write_failed"}
	}
	opts := ClipboardCommandOptions{
		Stdio:       []string{"pipe", "ignore", "ignore"},
		WindowsHide: true,
	}
	if err := req.Start("pbcopy", nil, opts, req.Text); err != nil {
		return ClipboardWriteResult{Evidence: "tui_terminal_clipboard_write_failed"}
	}
	return ClipboardWriteResult{OK: true}
}

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
