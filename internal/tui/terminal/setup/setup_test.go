package setup

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestTUITerminalSetupKeybindings(t *testing.T) {
	t.Run("detects VS Code family terminals", func(t *testing.T) {
		if got := DetectVSCodeLikeTerminal(map[string]string{"CURSOR_TRACE_ID": "x"}); got != "cursor" {
			t.Fatalf("cursor detect = %q", got)
		}
		if got := DetectVSCodeLikeTerminal(map[string]string{"VSCODE_GIT_ASKPASS_MAIN": "/tmp/windsurf"}); got != "windsurf" {
			t.Fatalf("windsurf detect = %q", got)
		}
		if got := DetectVSCodeLikeTerminal(map[string]string{"TERM_PROGRAM": "vscode"}); got != "vscode" {
			t.Fatalf("vscode detect = %q", got)
		}
		if got := DetectVSCodeLikeTerminal(nil); got != "" {
			t.Fatalf("empty detect = %q, want empty", got)
		}
	})

	t.Run("computes config dirs", func(t *testing.T) {
		if got := VSCodeStyleConfigDir("Code", "darwin", nil, "/home/me"); got != "/home/me/Library/Application Support/Code/User" {
			t.Fatalf("darwin config dir = %q", got)
		}
		if got := VSCodeStyleConfigDir("Code", "linux", nil, "/home/me"); got != "/home/me/.config/Code/User" {
			t.Fatalf("linux config dir = %q", got)
		}
		got := VSCodeStyleConfigDir("Code", "win32", map[string]string{"APPDATA": "C:/Users/me/AppData/Roaming"}, "/home/me")
		if got != "C:/Users/me/AppData/Roaming/Code/User" {
			t.Fatalf("win32 config dir = %q", got)
		}
	})

	t.Run("strips comments and trailing commas", func(t *testing.T) {
		input := `[
		  // comment
		  {"key":"a","args":{"text":"// not a comment",},},
		  /* block */ {"key":"b"},
		]`
		var parsed []map[string]any
		if err := json.Unmarshal([]byte(StripJSONComments(input)), &parsed); err != nil {
			t.Fatalf("StripJSONComments output did not parse: %v", err)
		}
		if parsed[0]["key"] != "a" || parsed[0]["args"].(map[string]any)["text"] != "// not a comment" {
			t.Fatalf("parsed = %#v, want string contents preserved", parsed)
		}
	})

	t.Run("writes missing bindings and backs up existing files", func(t *testing.T) {
		fake := &fakeTerminalFileOps{body: []byte(`[]`)}
		result := ConfigureTerminalKeybindings("vscode", TerminalSetupOptions{
			HomeDir:  "/Users/me",
			Platform: "darwin",
			FileOps:  fake.ops(),
		})
		if !result.Success || !result.RequiresRestart {
			t.Fatalf("ConfigureTerminalKeybindings() = %+v, want success requiring restart", result)
		}
		if len(fake.writes) != 1 || len(fake.copies) != 1 {
			t.Fatalf("writes=%d copies=%d, want one write and one backup", len(fake.writes), len(fake.copies))
		}
		written := fake.writes[0]
		for _, want := range []string{"cmd+c", "terminalTextSelected", "\\u001b[99;13u", "shift+enter", "cmd+enter", "cmd+z"} {
			if !strings.Contains(written, want) {
				t.Fatalf("written keybindings missing %q:\n%s", want, written)
			}
		}
	})

	t.Run("does not add mac copy binding on linux", func(t *testing.T) {
		fake := &fakeTerminalFileOps{readErr: os.ErrNotExist}
		result := ConfigureTerminalKeybindings("vscode", TerminalSetupOptions{
			HomeDir:  "/home/me",
			Platform: "linux",
			FileOps:  fake.ops(),
		})
		if !result.Success {
			t.Fatalf("ConfigureTerminalKeybindings() = %+v, want success", result)
		}
		if strings.Contains(fake.writes[0], "cmd+c") || strings.Contains(fake.writes[0], "terminalTextSelected") {
			t.Fatalf("linux keybindings unexpectedly contain mac copy binding:\n%s", fake.writes[0])
		}
	})

	t.Run("reports overlapping conflicts without writing", func(t *testing.T) {
		fake := &fakeTerminalFileOps{body: []byte(`[{"key":"cmd+c","command":"workbench.action.terminal.copySelection","when":"terminalFocus"}]`)}
		result := ConfigureTerminalKeybindings("cursor", TerminalSetupOptions{
			HomeDir:  "/Users/me",
			Platform: "darwin",
			FileOps:  fake.ops(),
		})
		if result.Success || result.Evidence != "tui_terminal_keybinding_conflict" || !strings.Contains(result.Message, "cmd+c") {
			t.Fatalf("ConfigureTerminalKeybindings() = %+v, want cmd+c conflict", result)
		}
		if len(fake.writes) != 0 || len(fake.copies) != 0 {
			t.Fatalf("writes=%d copies=%d, want no mutation on conflict", len(fake.writes), len(fake.copies))
		}
	})

	t.Run("ignores disjoint terminal selection conflicts", func(t *testing.T) {
		fake := &fakeTerminalFileOps{body: []byte(`[{"key":"cmd+c","command":"workbench.action.terminal.sendSequence","when":"terminalFocus && !terminalTextSelected","args":{"text":"\u0003"}}]`)}
		result := ConfigureTerminalKeybindings("vscode", TerminalSetupOptions{
			HomeDir:  "/Users/me",
			Platform: "darwin",
			FileOps:  fake.ops(),
		})
		if !result.Success || len(fake.writes) != 1 {
			t.Fatalf("ConfigureTerminalKeybindings() = %+v writes=%d, want disjoint success", result, len(fake.writes))
		}
	})

	t.Run("refuses detected setup from ssh", func(t *testing.T) {
		result := ConfigureDetectedTerminalKeybindings(TerminalSetupOptions{
			Env:      map[string]string{"SSH_CONNECTION": "1 2 3 4", "TERM_PROGRAM": "vscode"},
			HomeDir:  "/Users/me",
			Platform: "darwin",
		})
		if result.Success || result.Evidence != "tui_terminal_setup_remote_refused" {
			t.Fatalf("ConfigureDetectedTerminalKeybindings() = %+v, want SSH refusal", result)
		}
	})

	t.Run("prompt decision sees missing and complete bindings", func(t *testing.T) {
		missing := &fakeTerminalFileOps{readErr: os.ErrNotExist}
		if !ShouldPromptForTerminalSetup(TerminalSetupOptions{
			Env:     map[string]string{"TERM_PROGRAM": "vscode"},
			HomeDir: "/tmp/fake-home",
			FileOps: missing.ops(),
		}) {
			t.Fatal("ShouldPromptForTerminalSetup missing = false, want true")
		}

		completeFile := &fakeTerminalFileOps{body: []byte(defaultCompleteKeybindingsJSON(t))}
		if ShouldPromptForTerminalSetup(TerminalSetupOptions{
			Env:     map[string]string{"TERM_PROGRAM": "vscode"},
			HomeDir: "/tmp/fake-home",
			FileOps: completeFile.ops(),
		}) {
			t.Fatal("ShouldPromptForTerminalSetup complete = true, want false")
		}
	})
}

func TestTUITerminalSetupReadFailure(t *testing.T) {
	fake := &fakeTerminalFileOps{readErr: errors.New("permission denied")}
	result := ConfigureTerminalKeybindings("vscode", TerminalSetupOptions{
		HomeDir:  "/Users/me",
		Platform: "darwin",
		FileOps:  fake.ops(),
	})
	if result.Success || result.Evidence != "tui_terminal_keybindings_read_failed" {
		t.Fatalf("ConfigureTerminalKeybindings(read failure) = %+v, want read failure", result)
	}
}
