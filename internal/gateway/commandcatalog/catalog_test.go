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

func TestRenderUsageAndOutOfRange(t *testing.T) {
	if got := Render(Request{RawArgs: "wat", BuiltinLines: []string{"`/help` -- Show available commands"}}); !strings.Contains(got, "Usage: /commands [page]") {
		t.Fatalf("invalid args reply = %q, want usage", got)
	}

	got := Render(Request{RawArgs: "99", BuiltinLines: []string{"`/help` -- Show available commands"}})
	if !strings.Contains(got, "page 1/1") || !strings.Contains(got, "requested page 99 out of range") {
		t.Fatalf("out-of-range reply = %q, want clamped page notice", got)
	}
}
