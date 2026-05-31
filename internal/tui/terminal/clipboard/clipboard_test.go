package clipboard

import (
	"encoding/base64"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func TestReadClipboardTextLinuxBackendOrderUsesTerminalEnvContract(t *testing.T) {
	var attempts []string
	result := ReadClipboardText(ClipboardReadRequest{
		Platform: "linux",
		Env: map[string]string{
			envvars.WSLInterop:     "/run/WSL/1_interop",
			envvars.WaylandDisplay: "wayland-0",
		},
		Run: func(name string, args []string, opts ClipboardCommandOptions) (ClipboardRunResult, error) {
			attempts = append(attempts, name)
			if name != "powershell.exe" {
				t.Fatalf("first backend = %q; want powershell.exe", name)
			}
			if opts.Encoding != "utf8" || opts.MaxBuffer == 0 || !opts.WindowsHide {
				t.Fatalf("read options = %+v; want Hermes-compatible clipboard command options", opts)
			}
			return ClipboardRunResult{Stdout: "from windows clipboard"}, nil
		},
	})

	if !result.OK || result.Text != "from windows clipboard" {
		t.Fatalf("ReadClipboardText() = %+v, want successful WSL clipboard read", result)
	}
	if len(attempts) != 1 || attempts[0] != "powershell.exe" {
		t.Fatalf("attempts = %#v, want only powershell.exe after success", attempts)
	}
}

func TestBuildOSC52ClipboardQueryWrapsForTmux(t *testing.T) {
	plain := BuildOSC52ClipboardQuery(nil)
	if plain != OSC52ClipboardQuery {
		t.Fatalf("plain OSC52 query = %q; want base query", plain)
	}

	wrapped := BuildOSC52ClipboardQuery(map[string]string{envvars.TMUX: "/tmp/tmux-1/default,1,0"})
	want := "\x1bPtmux;\x1b\x1b]52;c;?\x07\x1b\\"
	if wrapped != want {
		t.Fatalf("tmux OSC52 query = %q; want %q", wrapped, want)
	}
}

func TestParseOSC52ClipboardDataRejectsUnusableText(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello from osc52"))
	if got, ok := ParseOSC52ClipboardData("c;" + encoded); !ok || got != "hello from osc52" {
		t.Fatalf("ParseOSC52ClipboardData(valid) = %q, %v; want decoded text", got, ok)
	}
	if got, ok := ParseOSC52ClipboardData("c;" + base64.StdEncoding.EncodeToString([]byte{0, 1, 2})); ok || got != "" {
		t.Fatalf("ParseOSC52ClipboardData(binary) = %q, %v; want rejection", got, ok)
	}
}
