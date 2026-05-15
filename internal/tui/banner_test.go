package tui

import (
	"strings"
	"testing"
)

func TestBannerLogo_NonEmpty(t *testing.T) {
	skin := DefaultHermesSkin()
	logo := bannerLogo(skin)
	if strings.TrimSpace(logo) == "" {
		t.Fatal("banner logo is empty")
	}
	if !strings.Contains(logo, "██████╗  ██████╗ ██████╗") {
		t.Fatal("banner logo missing GORMES-AGENT block art")
	}
	if strings.Contains(logo, "██╗  ██╗███████╗██████╗") {
		t.Fatal("banner logo still contains upstream HERMES-AGENT block art")
	}
}

func TestBannerCaduceus_NonEmpty(t *testing.T) {
	cad := bannerCaduceus()
	if strings.TrimSpace(cad) == "" {
		t.Fatal("banner caduceus is empty")
	}
}

func TestBannerWelcome(t *testing.T) {
	w := bannerWelcome()
	if !strings.Contains(w, "Welcome to Gormes. Type your message or /help for commands.") {
		t.Fatalf("welcome = %q, want Gormes product branding", w)
	}
	if strings.Contains(w, "Gormes Agent") {
		t.Fatalf("welcome must not emit deprecated Gormes Agent wording: %q", w)
	}
}

func TestWelcomePanel_SessionContextAndIdentity(t *testing.T) {
	skin := DefaultHermesSkin()
	ctx := welcomeContext{
		Model:     "anthropic/claude-sonnet-4-20250514",
		Provider:  "anthropic",
		Runtime:   "anthropic_messages",
		CWD:       "~/git/gormes-agent",
		SessionID: "abcdef1234567890",
		Version:   "0.2.11",
	}
	got := welcomePanel(skin, ctx, 100)

	// Identity + the out-of-scope intro invariants must survive, in order.
	for _, want := range []string{
		"⚕ Gormes",
		"Go-native Hermes-compatible agent",
		"Type your message or /help for commands.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("welcome panel missing identity/intro invariant %q in:\n%s", want, got)
		}
	}
	idTitle := strings.Index(got, "⚕ Gormes")
	idSub := strings.Index(got, "Go-native Hermes-compatible agent")
	idTip := strings.Index(got, "Type your message or /help for commands.")
	if !(idTitle < idSub && idSub < idTip) {
		t.Fatalf("intro invariants out of order (title=%d sub=%d tip=%d):\n%s", idTitle, idSub, idTip, got)
	}

	// Session context lines.
	for _, want := range []string{
		ctx.Model,
		"anthropic",
		ctx.CWD,
		"abcdef12", // shortSessionID
		"0.2.11",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("welcome panel missing session context %q in:\n%s", want, got)
		}
	}

	// Framed with a horizontal rule at full width.
	if !strings.Contains(got, strings.Repeat("─", 8)) {
		t.Fatalf("welcome panel not rule-framed at width 100:\n%s", got)
	}
	// No-sidebar contract: no line carrying "Gormes" may pair vertical pipes,
	// and the panel must not draw a box border at all.
	if strings.Contains(got, "│") {
		t.Fatalf("welcome panel drew sidebar/box pipes (violates no-sidebar contract):\n%s", got)
	}
	if strings.Contains(got, "start typing below to begin") {
		t.Fatalf("welcome panel leaked legacy placeholder:\n%s", got)
	}
}

func TestWelcomePanel_SkinDerivedPalette(t *testing.T) {
	def := BuiltinSkins()["default"]
	pos := BuiltinSkins()["poseidon"]

	dp := welcomePaletteFor(def)
	pp := welcomePaletteFor(pos)

	// Palette must be sourced from the active skin's banner tokens, not hardcoded.
	if dp.border != def.Colors.BannerBorder || dp.title != def.Colors.BannerTitle {
		t.Fatalf("default palette not sourced from skin: %+v vs skin %+v", dp, def.Colors)
	}
	if pp.border != pos.Colors.BannerBorder {
		t.Fatalf("poseidon palette not sourced from skin: %+v", pp)
	}
	if dp.border == pp.border {
		t.Fatalf("switching skin did not re-theme border (%q == %q)", dp.border, pp.border)
	}

	// Both skins still render the stable identity structure.
	ctx := welcomeContext{Model: "m"}
	for name, sk := range map[string]HermesSkin{"default": def, "poseidon": pos} {
		out := welcomePanel(sk, ctx, 100)
		if !strings.Contains(out, "⚕ Gormes") || !strings.Contains(out, "Type your message or /help for commands.") {
			t.Fatalf("skin %s lost identity/intro structure:\n%s", name, out)
		}
	}
}

func TestWelcomePanel_MinimalChromeDegrades(t *testing.T) {
	skin := DefaultHermesSkin()
	ctx := welcomeContext{Model: "anthropic/x"}
	got := welcomePanel(skin, ctx, 50) // < hermesMinimalChromeWidth (64)

	if strings.Contains(got, strings.Repeat("─", 8)) {
		t.Fatalf("minimal-chrome welcome panel must drop the rule frame at width 50:\n%s", got)
	}
	if !strings.Contains(got, "Type your message or /help for commands.") {
		t.Fatalf("minimal-chrome welcome panel dropped the slash tip:\n%s", got)
	}
	if !strings.Contains(got, "⚕ Gormes") {
		t.Fatalf("minimal-chrome welcome panel dropped identity:\n%s", got)
	}
}
