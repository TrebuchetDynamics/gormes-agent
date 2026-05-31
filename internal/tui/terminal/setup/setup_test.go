package setup

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func TestTUITerminalSetupKeybindings(t *testing.T) {
	t.Run("writes missing bindings and backs up existing files", func(t *testing.T) {
		fake := &fakeTerminalFileOps{body: []byte(`[]`)}
		result := ConfigureTerminalKeybindings(vscodeKindVSCode, TerminalSetupOptions{
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
		result := ConfigureTerminalKeybindings(vscodeKindVSCode, TerminalSetupOptions{
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
		result := ConfigureTerminalKeybindings(vscodeKindCursor, TerminalSetupOptions{
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
		result := ConfigureTerminalKeybindings(vscodeKindVSCode, TerminalSetupOptions{
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
			Env:      map[string]string{envvars.SSHConnection: "1 2 3 4", envvars.TermProgram: envvars.VSCodeTermProgram},
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
			Env:     map[string]string{envvars.TermProgram: envvars.VSCodeTermProgram},
			HomeDir: "/tmp/fake-home",
			FileOps: missing.ops(),
		}) {
			t.Fatal("ShouldPromptForTerminalSetup missing = false, want true")
		}

		completeFile := &fakeTerminalFileOps{body: []byte(defaultCompleteKeybindingsJSON(t))}
		if ShouldPromptForTerminalSetup(TerminalSetupOptions{
			Env:     map[string]string{envvars.TermProgram: envvars.VSCodeTermProgram},
			HomeDir: "/tmp/fake-home",
			FileOps: completeFile.ops(),
		}) {
			t.Fatal("ShouldPromptForTerminalSetup complete = true, want false")
		}
	})
}

func TestTUITerminalSetupReadFailure(t *testing.T) {
	fake := &fakeTerminalFileOps{readErr: errors.New("permission denied")}
	result := ConfigureTerminalKeybindings(vscodeKindVSCode, TerminalSetupOptions{
		HomeDir:  "/Users/me",
		Platform: "darwin",
		FileOps:  fake.ops(),
	})
	if result.Success || result.Evidence != "tui_terminal_keybindings_read_failed" {
		t.Fatalf("ConfigureTerminalKeybindings(read failure) = %+v, want read failure", result)
	}
}
