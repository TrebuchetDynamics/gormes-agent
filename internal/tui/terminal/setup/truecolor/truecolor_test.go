package truecolor

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func TestDecision(t *testing.T) {
	if got := Decision(nil); got.Force {
		t.Fatalf("Decision(nil) = %+v, want no force", got)
	}
	if got := Decision(map[string]string{envvars.TermProgram: envvars.AppleTerminalProgram, "TERM": "xterm-256color"}); got.Force {
		t.Fatalf("Apple Terminal truecolor decision = %+v, want no implicit force", got)
	}
	appleDowngrade := Decision(map[string]string{
		envvars.TermProgram: envvars.AppleTerminalProgram,
		envvars.ColorTerm:   envvars.Truecolor,
		envvars.ForceColor:  envvars.ForceColorTruecolor,
	})
	if appleDowngrade.Force || !sameStringSet(appleDowngrade.Unset, []string{envvars.ColorTerm, envvars.ForceColor}) {
		t.Fatalf("Apple Terminal advertised truecolor decision = %+v, want COLORTERM and FORCE_COLOR unset", appleDowngrade)
	}
	appleKeepForceColor := Decision(map[string]string{
		envvars.TermProgram: envvars.AppleTerminalProgram,
		envvars.ColorTerm:   envvars.Truecolor24Bit,
		envvars.ForceColor:  "2",
	})
	if appleKeepForceColor.Force || !sameStringSet(appleKeepForceColor.Unset, []string{envvars.ColorTerm}) {
		t.Fatalf("Apple Terminal non-3 FORCE_COLOR decision = %+v, want only COLORTERM unset", appleKeepForceColor)
	}
	nonApple := Decision(map[string]string{envvars.ColorTerm: envvars.Truecolor, envvars.ForceColor: envvars.ForceColorTruecolor})
	if nonApple.Force || len(nonApple.Unset) != 0 || len(nonApple.Set) != 0 {
		t.Fatalf("non-Apple advertised truecolor decision = %+v, want untouched", nonApple)
	}
	enabled := Decision(map[string]string{envvars.HermesTUITruecolor: "1", envvars.ForceColor: "0"})
	if !enabled.Force || enabled.Set[envvars.ColorTerm] != envvars.Truecolor || enabled.Set[envvars.ForceColor] != envvars.ForceColorTruecolor {
		t.Fatalf("enabled truecolor = %+v, want COLORTERM=truecolor FORCE_COLOR=3", enabled)
	}
	appleEnabled := Decision(map[string]string{
		envvars.TermProgram:        envvars.AppleTerminalProgram,
		envvars.ColorTerm:          envvars.Truecolor,
		envvars.ForceColor:         envvars.ForceColorTruecolor,
		envvars.HermesTUITruecolor: "yes",
	})
	if !appleEnabled.Force || len(appleEnabled.Unset) != 0 {
		t.Fatalf("explicit Apple truecolor = %+v, want force without downgrade", appleEnabled)
	}
	disabled := Decision(map[string]string{envvars.HermesTUITruecolor: "off", envvars.ColorTerm: envvars.Truecolor})
	if disabled.Force || len(disabled.Unset) != 0 {
		t.Fatalf("disabled truecolor = %+v, want explicit opt-out without env mutation", disabled)
	}
	if got := Decision(map[string]string{envvars.NoColor: "1", envvars.HermesTUITruecolor: "1"}); got.Force {
		t.Fatalf("NO_COLOR truecolor decision = %+v, want no force", got)
	}
	if got := Decision(map[string]string{envvars.NoColor: "", envvars.HermesTUITruecolor: "1"}); got.Force {
		t.Fatalf("empty NO_COLOR truecolor decision = %+v, want no force", got)
	}
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
