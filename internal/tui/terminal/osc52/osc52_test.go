package osc52

import (
	"encoding/base64"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func TestBuildClipboardQueryWrapsForTmux(t *testing.T) {
	plain := BuildClipboardQuery(nil)
	if plain != ClipboardQuery {
		t.Fatalf("plain OSC52 query = %q; want base query", plain)
	}

	wrapped := BuildClipboardQuery(map[string]string{envvars.TMUX: "/tmp/tmux-1/default,1,0"})
	want := "\x1bPtmux;\x1b\x1b]52;c;?\x07\x1b\\"
	if wrapped != want {
		t.Fatalf("tmux OSC52 query = %q; want %q", wrapped, want)
	}
}

func TestParseClipboardDataRejectsUnusableText(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello from osc52"))
	if got, ok := ParseClipboardData("c;" + encoded); !ok || got != "hello from osc52" {
		t.Fatalf("ParseClipboardData(valid) = %q, %v; want decoded text", got, ok)
	}
	if got, ok := ParseClipboardData("c;" + base64.StdEncoding.EncodeToString([]byte{0, 1, 2})); ok || got != "" {
		t.Fatalf("ParseClipboardData(binary) = %q, %v; want rejection", got, ok)
	}
}
