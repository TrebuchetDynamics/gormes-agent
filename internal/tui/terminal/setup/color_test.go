package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func TestTUITruecolorAndTerminalParityHints(t *testing.T) {
	if got := TruecolorDecision(nil); got.Force {
		t.Fatalf("TruecolorDecision(nil) = %+v, want no force", got)
	}
	if got := TruecolorDecision(map[string]string{envvars.TermProgram: envvars.AppleTerminalProgram, "TERM": "xterm-256color"}); got.Force {
		t.Fatalf("Apple Terminal truecolor decision = %+v, want no implicit force", got)
	}
	appleDowngrade := TruecolorDecision(map[string]string{
		envvars.TermProgram: envvars.AppleTerminalProgram,
		envvars.ColorTerm:   envvars.Truecolor,
		envvars.ForceColor:  envvars.ForceColorTruecolor,
	})
	if appleDowngrade.Force || !sameStringSet(appleDowngrade.Unset, []string{envvars.ColorTerm, envvars.ForceColor}) {
		t.Fatalf("Apple Terminal advertised truecolor decision = %+v, want COLORTERM and FORCE_COLOR unset", appleDowngrade)
	}
	appleKeepForceColor := TruecolorDecision(map[string]string{
		envvars.TermProgram: envvars.AppleTerminalProgram,
		envvars.ColorTerm:   envvars.Truecolor24Bit,
		envvars.ForceColor:  "2",
	})
	if appleKeepForceColor.Force || !sameStringSet(appleKeepForceColor.Unset, []string{envvars.ColorTerm}) {
		t.Fatalf("Apple Terminal non-3 FORCE_COLOR decision = %+v, want only COLORTERM unset", appleKeepForceColor)
	}
	nonApple := TruecolorDecision(map[string]string{envvars.ColorTerm: envvars.Truecolor, envvars.ForceColor: envvars.ForceColorTruecolor})
	if nonApple.Force || len(nonApple.Unset) != 0 || len(nonApple.Set) != 0 {
		t.Fatalf("non-Apple advertised truecolor decision = %+v, want untouched", nonApple)
	}
	enabled := TruecolorDecision(map[string]string{envvars.HermesTUITruecolor: "1", envvars.ForceColor: "0"})
	if !enabled.Force || enabled.Set[envvars.ColorTerm] != envvars.Truecolor || enabled.Set[envvars.ForceColor] != envvars.ForceColorTruecolor {
		t.Fatalf("enabled truecolor = %+v, want COLORTERM=truecolor FORCE_COLOR=3", enabled)
	}
	appleEnabled := TruecolorDecision(map[string]string{
		envvars.TermProgram:        envvars.AppleTerminalProgram,
		envvars.ColorTerm:          envvars.Truecolor,
		envvars.ForceColor:         envvars.ForceColorTruecolor,
		envvars.HermesTUITruecolor: "yes",
	})
	if !appleEnabled.Force || len(appleEnabled.Unset) != 0 {
		t.Fatalf("explicit Apple truecolor = %+v, want force without downgrade", appleEnabled)
	}
	disabled := TruecolorDecision(map[string]string{envvars.HermesTUITruecolor: "off", envvars.ColorTerm: envvars.Truecolor})
	if disabled.Force || len(disabled.Unset) != 0 {
		t.Fatalf("disabled truecolor = %+v, want explicit opt-out without env mutation", disabled)
	}
	if got := TruecolorDecision(map[string]string{envvars.NoColor: "1", envvars.HermesTUITruecolor: "1"}); got.Force {
		t.Fatalf("NO_COLOR truecolor decision = %+v, want no force", got)
	}
	if got := TruecolorDecision(map[string]string{envvars.NoColor: "", envvars.HermesTUITruecolor: "1"}); got.Force {
		t.Fatalf("empty NO_COLOR truecolor decision = %+v, want no force", got)
	}

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
