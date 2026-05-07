package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type recordedClipboardRun struct {
	name string
	args []string
	opts ClipboardCommandOptions
}

func TestTUIClipboardReadFallbacks(t *testing.T) {
	t.Run("macos pbpaste", func(t *testing.T) {
		var calls []recordedClipboardRun
		result := ReadClipboardText(ClipboardReadRequest{
			Platform: "darwin",
			Run: func(name string, args []string, opts ClipboardCommandOptions) (ClipboardRunResult, error) {
				calls = append(calls, recordedClipboardRun{name: name, args: args, opts: opts})
				return ClipboardRunResult{Stdout: "hello world\n"}, nil
			},
		})

		if !result.OK || result.Text != "hello world\n" {
			t.Fatalf("ReadClipboardText() = %+v, want pbpaste text", result)
		}
		if len(calls) != 1 || calls[0].name != "pbpaste" || len(calls[0].args) != 0 {
			t.Fatalf("calls = %+v, want pbpaste with no args", calls)
		}
		if calls[0].opts.Encoding != "utf8" || calls[0].opts.MaxBuffer != 4*1024*1024 || !calls[0].opts.WindowsHide {
			t.Fatalf("opts = %+v, want Hermes clipboard exec options", calls[0].opts)
		}
	})

	t.Run("windows powershell", func(t *testing.T) {
		var calls []recordedClipboardRun
		result := ReadClipboardText(ClipboardReadRequest{
			Platform: "win32",
			Run: func(name string, args []string, opts ClipboardCommandOptions) (ClipboardRunResult, error) {
				calls = append(calls, recordedClipboardRun{name: name, args: args, opts: opts})
				return ClipboardRunResult{Stdout: "from windows\r\n"}, nil
			},
		})

		if !result.OK || result.Text != "from windows\r\n" {
			t.Fatalf("ReadClipboardText() = %+v, want PowerShell text", result)
		}
		wantArgs := []string{"-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw"}
		if len(calls) != 1 || calls[0].name != "powershell" || !reflect.DeepEqual(calls[0].args, wantArgs) {
			t.Fatalf("calls = %+v, want powershell %q", calls, wantArgs)
		}
	})

	t.Run("wsl powershell exe first", func(t *testing.T) {
		var calls []recordedClipboardRun
		result := ReadClipboardText(ClipboardReadRequest{
			Platform: "linux",
			Env:      map[string]string{"WSL_INTEROP": "/tmp/socket"},
			Run: func(name string, args []string, opts ClipboardCommandOptions) (ClipboardRunResult, error) {
				calls = append(calls, recordedClipboardRun{name: name, args: args, opts: opts})
				return ClipboardRunResult{Stdout: "from wsl\n"}, nil
			},
		})

		if !result.OK || result.Text != "from wsl\n" {
			t.Fatalf("ReadClipboardText() = %+v, want WSL text", result)
		}
		if len(calls) != 1 || calls[0].name != "powershell.exe" {
			t.Fatalf("calls = %+v, want powershell.exe first", calls)
		}
	})

	t.Run("wayland falls back to xclip", func(t *testing.T) {
		var calls []recordedClipboardRun
		result := ReadClipboardText(ClipboardReadRequest{
			Platform: "linux",
			Env:      map[string]string{"WAYLAND_DISPLAY": "wayland-1"},
			Run: func(name string, args []string, opts ClipboardCommandOptions) (ClipboardRunResult, error) {
				calls = append(calls, recordedClipboardRun{name: name, args: args, opts: opts})
				if name == "wl-paste" {
					return ClipboardRunResult{}, errors.New("missing")
				}
				return ClipboardRunResult{Stdout: "from xclip\n"}, nil
			},
		})

		if !result.OK || result.Text != "from xclip\n" {
			t.Fatalf("ReadClipboardText() = %+v, want xclip fallback text", result)
		}
		if got, want := []string{calls[0].name, calls[1].name}, []string{"wl-paste", "xclip"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("clipboard backend order = %v, want %v", got, want)
		}
	})

	t.Run("rejects unusable payload", func(t *testing.T) {
		result := ReadClipboardText(ClipboardReadRequest{
			Platform: "darwin",
			Run: func(string, []string, ClipboardCommandOptions) (ClipboardRunResult, error) {
				return ClipboardRunResult{Stdout: "PNG\x00\x01IHDR"}, nil
			},
		})

		if result.OK || result.Evidence != "tui_terminal_clipboard_unusable" {
			t.Fatalf("ReadClipboardText() = %+v, want unusable evidence", result)
		}
	})

	t.Run("returns unavailable when every backend fails", func(t *testing.T) {
		result := ReadClipboardText(ClipboardReadRequest{
			Platform: "linux",
			Env:      map[string]string{"WAYLAND_DISPLAY": "wayland-1"},
			Run: func(string, []string, ClipboardCommandOptions) (ClipboardRunResult, error) {
				return ClipboardRunResult{}, errors.New("no clipboard")
			},
		})

		if result.OK || result.Evidence != "tui_terminal_clipboard_unavailable" {
			t.Fatalf("ReadClipboardText() = %+v, want unavailable evidence", result)
		}
	})
}

func TestTUIClipboardWriteMacOnly(t *testing.T) {
	t.Run("unsupported platform does not start a writer", func(t *testing.T) {
		started := false
		result := WriteClipboardText(ClipboardWriteRequest{
			Platform: "linux",
			Text:     "hello",
			Start: func(string, []string, ClipboardCommandOptions, string) error {
				started = true
				return nil
			},
		})

		if result.OK || result.Evidence != "tui_terminal_clipboard_write_unsupported" || started {
			t.Fatalf("WriteClipboardText() = %+v started=%v, want unsupported without start", result, started)
		}
	})

	t.Run("macos writes through pbcopy", func(t *testing.T) {
		var gotName string
		var gotInput string
		result := WriteClipboardText(ClipboardWriteRequest{
			Platform: "darwin",
			Text:     "hello world",
			Start: func(name string, args []string, opts ClipboardCommandOptions, input string) error {
				gotName = name
				gotInput = input
				if len(args) != 0 || !reflect.DeepEqual(opts.Stdio, []string{"pipe", "ignore", "ignore"}) || !opts.WindowsHide {
					t.Fatalf("pbcopy args=%q opts=%+v, want Hermes write options", args, opts)
				}
				return nil
			},
		})

		if !result.OK || gotName != "pbcopy" || gotInput != "hello world" {
			t.Fatalf("WriteClipboardText() = %+v name=%q input=%q, want pbcopy success", result, gotName, gotInput)
		}
	})

	t.Run("macos pbcopy failure returns evidence", func(t *testing.T) {
		result := WriteClipboardText(ClipboardWriteRequest{
			Platform: "darwin",
			Text:     "hello",
			Start: func(string, []string, ClipboardCommandOptions, string) error {
				return errors.New("closed")
			},
		})

		if result.OK || result.Evidence != "tui_terminal_clipboard_write_failed" {
			t.Fatalf("WriteClipboardText() = %+v, want failure evidence", result)
		}
	})
}

func TestTUIOSC52QueryDecodeAndTmuxWrap(t *testing.T) {
	if got := BuildOSC52ClipboardQuery(nil); got != OSC52ClipboardQuery {
		t.Fatalf("BuildOSC52ClipboardQuery(nil) = %q, want raw query", got)
	}

	tmux := BuildOSC52ClipboardQuery(map[string]string{"TMUX": "/tmp/tmux-1/default,1,0"})
	if !strings.Contains(tmux, "\x1bPtmux;") || !strings.Contains(tmux, "]52;c;?") {
		t.Fatalf("tmux query = %q, want tmux passthrough OSC52 query", tmux)
	}

	encoded := "c;aGVsbG8gZnJvbSBvc2M1Mg=="
	if got, ok := ParseOSC52ClipboardData(encoded); !ok || got != "hello from osc52" {
		t.Fatalf("ParseOSC52ClipboardData(%q) = %q,%v", encoded, got, ok)
	}
	if got, ok := ParseOSC52ClipboardData("c;?"); ok || got != "" {
		t.Fatalf("ParseOSC52ClipboardData query = %q,%v, want empty false", got, ok)
	}

	flushed := false
	result := ReadOSC52Clipboard(map[string]string{"TMUX": "1"}, func(query string) (OSC52Response, error) {
		if !strings.Contains(query, "\x1bPtmux;") {
			t.Fatalf("query = %q, want tmux passthrough", query)
		}
		return OSC52Response{Code: 52, Type: "osc", Data: "c;cXVlcmllZCB0ZXh0"}, nil
	}, func() error {
		flushed = true
		return nil
	})
	if !result.OK || result.Text != "queried text" || !flushed {
		t.Fatalf("ReadOSC52Clipboard() = %+v flushed=%v, want decoded text", result, flushed)
	}

	unsupported := ReadOSC52Clipboard(nil, nil, nil)
	if unsupported.OK || unsupported.Evidence != "tui_terminal_osc52_unavailable" {
		t.Fatalf("ReadOSC52Clipboard(nil) = %+v, want unavailable evidence", unsupported)
	}
}
