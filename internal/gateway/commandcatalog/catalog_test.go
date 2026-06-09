package commandcatalog

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/gatewaytest"
)

func TestRenderPaginatesBuiltinsAndSkills(t *testing.T) {
	builtins := make([]string, 0, 18)
	for i := 1; i <= 18; i++ {
		builtins = append(builtins, "`/cmd` -- command")
	}
	reply := Render(Request{
		Platform:     "telegram",
		RawArgs:      "2",
		BuiltinLines: builtins,
		SkillCommands: []Command{
			{Name: "review-skill", Description: "Review code"},
			{Name: "ops-skill", Description: "Operate safely"},
		},
	})

	gatewaytest.AssertContainsAll(t, reply, "Available commands", "page 2/2", "Skill commands", "`/ops-skill`", "`/review-skill`")
}

func TestRenderCountsOnlyCommandsNotSectionChrome(t *testing.T) {
	got := Render(Request{
		SkillCommands: []Command{{Name: "review-skill", Description: "Review code"}},
	})
	if strings.Contains(got, "Available commands (3 total)") {
		t.Fatalf("Render counted blank/header chrome as commands:\n%s", got)
	}
	if !strings.Contains(got, "Available commands (1 total)") {
		t.Fatalf("Render missing command-only total:\n%s", got)
	}
}

func TestRenderOmitsSkillCommandsHeaderWhenAllDynamicNamesInvalid(t *testing.T) {
	got := Render(Request{
		BuiltinLines:  []string{"`/help` -- Show available commands"},
		SkillCommands: []Command{{Name: " \n\t ", Description: "hidden"}},
	})
	if strings.Contains(got, "Skill commands:") || strings.Contains(got, "hidden") {
		t.Fatalf("Render leaked empty skill section for invalid dynamic commands:\n%s", got)
	}
	if !strings.Contains(got, "`/help` -- Show available commands") {
		t.Fatalf("Render omitted builtin help line:\n%s", got)
	}
}

func TestRenderRedactsSecretLikeDynamicSkillCommands(t *testing.T) {
	reply := Render(Request{
		SkillCommands: []Command{{
			Name:        "api_key=plain-secret-token",
			Description: "token=plain-secret-token",
		}},
	})
	for _, forbidden := range []string{"plain-secret-token", "api_key", "token="} {
		if strings.Contains(reply, forbidden) {
			t.Fatalf("Render leaked secret-like dynamic command value %q in:\n%s", forbidden, reply)
		}
	}
	if !strings.Contains(reply, "[redacted]") {
		t.Fatalf("Render missing redaction marker:\n%s", reply)
	}
}

func TestRenderSanitizesDynamicSkillCommands(t *testing.T) {
	reply := Render(Request{
		BuiltinLines: []string{"`/help` -- Show available commands"},
		SkillCommands: []Command{{
			Name:        "evil`\n**Injected:**",
			Description: "does work\n**Injected desc:** `token`",
		}},
	})
	for _, forbidden := range []string{"**Injected:**", "**Injected desc:**", "`token`", "evil`\n"} {
		if strings.Contains(reply, forbidden) {
			t.Fatalf("Render leaked dynamic command injection %q in:\n%s", forbidden, reply)
		}
	}
	gatewaytest.AssertContainsAll(t, reply, "`/evil' ''Injected:''` -- does work ''Injected desc:'' 'token'")
}

func TestRenderHugeNumericPageClampsOutOfRange(t *testing.T) {
	got := Render(Request{RawArgs: strings.Repeat("9", 80), BuiltinLines: []string{"`/help` -- Show available commands"}})
	if strings.Contains(got, "Usage: /commands [page]") {
		t.Fatalf("huge numeric page rendered usage instead of clamped page notice: %q", got)
	}
	if !strings.Contains(got, "page 1/1") || !strings.Contains(got, "requested page ") || !strings.Contains(got, "out of range") {
		t.Fatalf("huge numeric page = %q, want clamped out-of-range page notice", got)
	}
}

func TestRenderUsageAndOutOfRange(t *testing.T) {
	if got := Render(Request{RawArgs: "wat", BuiltinLines: []string{"`/help` -- Show available commands"}}); !strings.Contains(got, "Usage: /commands [page]") {
		t.Fatalf("invalid args reply = %q, want usage", got)
	}

	got := Render(Request{RawArgs: "99", BuiltinLines: []string{"`/help` -- Show available commands"}})
	if !strings.Contains(got, "page 1/1") || !strings.Contains(got, "requested page 99 out of range") {
		t.Fatalf("out-of-range reply = %q, want clamped page notice", got)
	}
}
