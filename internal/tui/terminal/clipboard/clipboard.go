package clipboard

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/cliptext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/osc52"
)

const clipboardMaxBuffer = 4 * 1024 * 1024

const OSC52ClipboardQuery = osc52.ClipboardQuery

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

type ClipboardTextResult = cliptext.Result

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

type OSC52Response = osc52.Response

type OSC52SendFunc = osc52.SendFunc

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
		if envvars.Value(env, envvars.WSLInterop) != "" {
			backends = append(backends, clipboardBackend{name: "powershell.exe", args: powershellArgs})
		}
		if envvars.Value(env, envvars.WaylandDisplay) != "" {
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
	return cliptext.IsUsable(text)
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
	return osc52.BuildClipboardQuery(env)
}

func ParseOSC52ClipboardData(data string) (string, bool) {
	return osc52.ParseClipboardData(data)
}

func ReadOSC52Clipboard(env map[string]string, send OSC52SendFunc, flush func() error) ClipboardTextResult {
	return osc52.ReadClipboard(env, send, flush)
}
