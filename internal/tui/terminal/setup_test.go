package terminal

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeTerminalFileOps struct {
	readErr error
	body    []byte
	writes  []string
	copies  [][2]string
	mkdirs  []string
}

func (f *fakeTerminalFileOps) ops() TerminalSetupFileOps {
	return TerminalSetupFileOps{
		MkdirAll: func(path string, _ os.FileMode) error {
			f.mkdirs = append(f.mkdirs, path)
			return nil
		},
		ReadFile: func(string) ([]byte, error) {
			if f.readErr != nil {
				return nil, f.readErr
			}
			return append([]byte(nil), f.body...), nil
		},
		WriteFile: func(_ string, body []byte, _ os.FileMode) error {
			f.writes = append(f.writes, string(body))
			return nil
		},
		CopyFile: func(src, dst string) error {
			f.copies = append(f.copies, [2]string{src, dst})
			return nil
		},
	}
}

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

func TestTUITruecolorAndTerminalParityHints(t *testing.T) {
	if got := TruecolorDecision(nil); got.Force {
		t.Fatalf("TruecolorDecision(nil) = %+v, want no force", got)
	}
	if got := TruecolorDecision(map[string]string{"TERM_PROGRAM": "Apple_Terminal", "TERM": "xterm-256color"}); got.Force {
		t.Fatalf("Apple Terminal truecolor decision = %+v, want no implicit force", got)
	}
	appleDowngrade := TruecolorDecision(map[string]string{
		"TERM_PROGRAM": "Apple_Terminal",
		"COLORTERM":    "truecolor",
		"FORCE_COLOR":  "3",
	})
	if appleDowngrade.Force || !sameStringSet(appleDowngrade.Unset, []string{"COLORTERM", "FORCE_COLOR"}) {
		t.Fatalf("Apple Terminal advertised truecolor decision = %+v, want COLORTERM and FORCE_COLOR unset", appleDowngrade)
	}
	appleKeepForceColor := TruecolorDecision(map[string]string{
		"TERM_PROGRAM": "Apple_Terminal",
		"COLORTERM":    "24bit",
		"FORCE_COLOR":  "2",
	})
	if appleKeepForceColor.Force || !sameStringSet(appleKeepForceColor.Unset, []string{"COLORTERM"}) {
		t.Fatalf("Apple Terminal non-3 FORCE_COLOR decision = %+v, want only COLORTERM unset", appleKeepForceColor)
	}
	nonApple := TruecolorDecision(map[string]string{"COLORTERM": "truecolor", "FORCE_COLOR": "3"})
	if nonApple.Force || len(nonApple.Unset) != 0 || len(nonApple.Set) != 0 {
		t.Fatalf("non-Apple advertised truecolor decision = %+v, want untouched", nonApple)
	}
	enabled := TruecolorDecision(map[string]string{"HERMES_TUI_TRUECOLOR": "1", "FORCE_COLOR": "0"})
	if !enabled.Force || enabled.Set["COLORTERM"] != "truecolor" || enabled.Set["FORCE_COLOR"] != "3" {
		t.Fatalf("enabled truecolor = %+v, want COLORTERM=truecolor FORCE_COLOR=3", enabled)
	}
	appleEnabled := TruecolorDecision(map[string]string{
		"TERM_PROGRAM":         "Apple_Terminal",
		"COLORTERM":            "truecolor",
		"FORCE_COLOR":          "3",
		"HERMES_TUI_TRUECOLOR": "yes",
	})
	if !appleEnabled.Force || len(appleEnabled.Unset) != 0 {
		t.Fatalf("explicit Apple truecolor = %+v, want force without downgrade", appleEnabled)
	}
	disabled := TruecolorDecision(map[string]string{"HERMES_TUI_TRUECOLOR": "off", "COLORTERM": "truecolor"})
	if disabled.Force || len(disabled.Unset) != 0 {
		t.Fatalf("disabled truecolor = %+v, want explicit opt-out without env mutation", disabled)
	}
	if got := TruecolorDecision(map[string]string{"NO_COLOR": "1", "HERMES_TUI_TRUECOLOR": "1"}); got.Force {
		t.Fatalf("NO_COLOR truecolor decision = %+v, want no force", got)
	}
	if got := TruecolorDecision(map[string]string{"NO_COLOR": "", "HERMES_TUI_TRUECOLOR": "1"}); got.Force {
		t.Fatalf("empty NO_COLOR truecolor decision = %+v, want no force", got)
	}

	fake := &fakeTerminalFileOps{readErr: os.ErrNotExist}
	hints := TerminalParityHints(map[string]string{
		"TERM_PROGRAM":    "Apple_Terminal",
		"TERM_SESSION_ID": "w0t0p0:123",
		"SSH_CONNECTION":  "1",
		"TMUX":            "/tmp/tmux-1/default,1,0",
	}, TerminalSetupOptions{FileOps: fake.ops(), HomeDir: t.TempDir()})
	for _, key := range []string{"apple-terminal", "remote", "tmux"} {
		if !hasTerminalHint(hints, key) {
			t.Fatalf("TerminalParityHints() = %+v, missing %q", hints, key)
		}
	}

	ideHints := TerminalParityHints(map[string]string{"TERM_PROGRAM": "vscode"}, TerminalSetupOptions{
		FileOps: fake.ops(),
		HomeDir: filepath.Join(t.TempDir(), "home"),
	})
	if !hasTerminalHint(ideHints, "ide-setup") {
		t.Fatalf("TerminalParityHints(vscode missing) = %+v, want ide-setup", ideHints)
	}
}

func hasTerminalHint(hints []TerminalParityHint, key string) bool {
	for _, hint := range hints {
		if hint.Key == key {
			return true
		}
	}
	return false
}

func sameStringSet(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]bool, len(got))
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			return false
		}
	}
	return true
}

func defaultCompleteKeybindingsJSON(t *testing.T) string {
	t.Helper()
	bindings := defaultTerminalKeybindings("darwin")
	body, err := json.Marshal(bindings)
	if err != nil {
		t.Fatalf("marshal complete bindings: %v", err)
	}
	return string(body)
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
