package hints

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func TestParityHintsClassifiesEnvironmentAndSetupCallback(t *testing.T) {
	env := map[string]string{
		envvars.TermProgram:   envvars.AppleTerminalProgram,
		envvars.SSHConnection: "1",
		envvars.TMUX:          "/tmp/tmux-1/default,1,0",
	}
	hints := ParityHints(env, func(got map[string]string) bool {
		if got[envvars.TermProgram] != envvars.AppleTerminalProgram {
			t.Fatalf("missingSetup env = %#v, want original env", got)
		}
		return true
	})
	for _, key := range []string{"apple-terminal", "remote", "tmux", "ide-setup"} {
		if !hasHint(hints, key) {
			t.Fatalf("ParityHints() = %+v, missing %q", hints, key)
		}
	}
}

func TestParityHintsSkipsSetupWhenNoCallback(t *testing.T) {
	if got := ParityHints(map[string]string{envvars.TermProgram: envvars.VSCodeTermProgram}, nil); hasHint(got, "ide-setup") {
		t.Fatalf("ParityHints(nil setup callback) = %+v, want no ide-setup hint", got)
	}
}

func hasHint(hints []Hint, key string) bool {
	for _, hint := range hints {
		if hint.Key == key {
			return true
		}
	}
	return false
}
