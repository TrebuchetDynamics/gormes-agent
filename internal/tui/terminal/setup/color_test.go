package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func TestTerminalParityHints(t *testing.T) {
	fake := &fakeTerminalFileOps{readErr: os.ErrNotExist}
	hints := TerminalParityHints(map[string]string{
		envvars.TermProgram:   envvars.AppleTerminalProgram,
		"TERM_SESSION_ID":     "w0t0p0:123",
		envvars.SSHConnection: "1",
		envvars.TMUX:          "/tmp/tmux-1/default,1,0",
	}, TerminalSetupOptions{FileOps: fake.ops(), HomeDir: t.TempDir()})
	for _, key := range []string{"apple-terminal", "remote", "tmux"} {
		if !hasTerminalHint(hints, key) {
			t.Fatalf("TerminalParityHints() = %+v, missing %q", hints, key)
		}
	}

	ideHints := TerminalParityHints(map[string]string{envvars.TermProgram: envvars.VSCodeTermProgram}, TerminalSetupOptions{
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
