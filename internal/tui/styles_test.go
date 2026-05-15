package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestChatPalette_SkinDerivedAndRoleDistinct(t *testing.T) {
	def := BuiltinSkins()["default"]
	p := chatPaletteFor(def)

	// Every semantic color must be sourced from the active skin's tokens,
	// never hardcoded.
	cases := map[string][2]string{
		"user":       {p.user, def.Colors.UILabel},
		"toolName":   {p.toolName, def.Colors.UIAcent},
		"toolOutput": {p.toolOutput, def.Colors.BannerDim},
		"error":      {p.errorc, def.Colors.UIError},
		"prompt":     {p.prompt, def.Colors.Prompt},
		"separator":  {p.separator, def.Colors.SessionBorder},
	}
	for role, pair := range cases {
		if pair[0] == "" || pair[0] != pair[1] {
			t.Fatalf("role %s color %q not sourced from skin token %q", role, pair[0], pair[1])
		}
	}

	// Roles must be visually distinguishable from one another.
	if p.user == p.toolName || p.toolName == p.errorc || p.user == p.errorc {
		t.Fatalf("semantic roles not distinct: user=%q tool=%q error=%q", p.user, p.toolName, p.errorc)
	}
}

func TestChatPalette_ReThemesPerSkin(t *testing.T) {
	def := chatPaletteFor(BuiltinSkins()["default"])
	pos := chatPaletteFor(BuiltinSkins()["poseidon"])

	if def.user == pos.user && def.errorc == pos.errorc && def.toolName == pos.toolName {
		t.Fatalf("switching skin did not re-theme any chat role: default=%+v poseidon=%+v", def, pos)
	}

	// Styles resolve without panicking for every built-in skin.
	for name, sk := range BuiltinSkins() {
		cs := chatStylesFor(sk)
		if cs.User.Render("x") == "" || cs.Error.Render("y") == "" {
			t.Fatalf("skin %s produced empty role render", name)
		}
	}
}

func TestView_RoleDifferentiationSurvivesStyleRefactor(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "user", Content: "first ask"},
			{Role: "assistant", Content: "the answer"},
			{Role: "tool", Name: "search_files", Content: "hit one\nhit two"},
		},
		LastError: "boom failure",
	}
	got := renderConv(frame, 100, 24)

	// Each role keeps a distinct, scannable marker after routing through the
	// semantic style system.
	for role, marker := range map[string]string{
		"user":      "❯",
		"assistant": "┊",
		"tool":      "⚡ search_files",
		"error":     "err:",
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("role %s lost its differentiator %q after style refactor:\n%s", role, marker, got)
		}
	}
}
