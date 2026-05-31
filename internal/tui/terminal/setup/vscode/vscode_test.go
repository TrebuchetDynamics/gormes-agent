package vscode

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func TestDetectLikeTerminal(t *testing.T) {
	if got := DetectLikeTerminal(map[string]string{envvars.CursorTraceID: "x"}); got != KindCursor {
		t.Fatalf("cursor detect = %q", got)
	}
	if got := DetectLikeTerminal(map[string]string{envvars.VSCodeGitAskpassMain: "/tmp/windsurf"}); got != KindWindsurf {
		t.Fatalf("windsurf detect = %q", got)
	}
	if got := DetectLikeTerminal(map[string]string{envvars.TermProgram: envvars.VSCodeTermProgram}); got != KindVSCode {
		t.Fatalf("vscode detect = %q", got)
	}
	if got := DetectLikeTerminal(nil); got != "" {
		t.Fatalf("empty detect = %q, want empty", got)
	}
}

func TestVSCodeStyleConfigDir(t *testing.T) {
	if got := VSCodeStyleConfigDir("Code", "darwin", nil, "/home/me"); got != "/home/me/Library/Application Support/Code/User" {
		t.Fatalf("darwin config dir = %q", got)
	}
	if got := VSCodeStyleConfigDir("Code", "linux", nil, "/home/me"); got != "/home/me/.config/Code/User" {
		t.Fatalf("linux config dir = %q", got)
	}
	got := VSCodeStyleConfigDir("Code", "win32", map[string]string{envvars.AppData: "C:/Users/me/AppData/Roaming"}, "/home/me")
	if got != "C:/Users/me/AppData/Roaming/Code/User" {
		t.Fatalf("win32 config dir = %q", got)
	}
}

func TestKeybindingsPathUsesDetectedAppName(t *testing.T) {
	got := KeybindingsPath(KindCursor, "linux", nil, "/home/me")
	if got != "/home/me/.config/Cursor/User/keybindings.json" {
		t.Fatalf("cursor keybindings path = %q", got)
	}
}
