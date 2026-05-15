package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

	// Framed with a boxed product banner at full width, matching the
	// operator-visible Hermes/Gormes startup surface rather than a loose
	// rule-delimited text block.
	for _, want := range []string{"╔", "║", "╚"} {
		if !strings.Contains(got, want) {
			t.Fatalf("welcome panel missing boxed banner marker %q:\n%s", want, got)
		}
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

func TestWelcomePanel_VersionToolCountSeam(t *testing.T) {
	// Zero-value safe: with no seam set and an empty ctx, R1 best-effort/omit
	// behavior holds — no version/tools lines, no panic.
	SetWelcomeContext("", 0)
	bare := welcomePanel(DefaultHermesSkin(), welcomeContext{Model: "anthropic/x"}, 100)
	if strings.Contains(bare, " tools") {
		t.Fatalf("unset seam must not render a tool-count line:\n%s", bare)
	}

	// Seeded by cmd/gormes at startup: real version + agent tool count show.
	SetWelcomeContext("0.2.11", 42, "terminal", "skills")
	defer SetWelcomeContext("", 0)
	got := welcomePanel(DefaultHermesSkin(), welcomeContext{Model: "anthropic/x"}, 100)
	if !strings.Contains(got, "0.2.11") {
		t.Fatalf("seeded version not rendered:\n%s", got)
	}
	if !strings.Contains(got, "42 tools") {
		t.Fatalf("seeded tool count not rendered:\n%s", got)
	}
	if !strings.Contains(got, "toolsets: terminal, skills") {
		t.Fatalf("seeded toolsets not rendered:\n%s", got)
	}

	// An explicit ctx value still wins over the seam (frame-derived data is
	// authoritative when present).
	ctxWin := welcomePanel(DefaultHermesSkin(), welcomeContext{Model: "m", Version: "9.9.9"}, 100)
	if !strings.Contains(ctxWin, "9.9.9") || strings.Contains(ctxWin, "0.2.11") {
		t.Fatalf("explicit ctx.Version must win over seam:\n%s", ctxWin)
	}
}

func TestWelcomePanel_FullWidthDoesNotDuplicateBrandHeader(t *testing.T) {
	got := welcomePanel(DefaultHermesSkin(), welcomeContext{
		Model:   "gpt-5.5",
		Version: "0.2.11",
	}, 72)

	if count := strings.Count(got, "⚕ Gormes"); count != 1 {
		t.Fatalf("full-width welcome panel rendered duplicate product headers (%d):\n%s", count, got)
	}
	if strings.Contains(got, "\nversion 0.2.11") {
		t.Fatalf("full-width welcome panel repeated the boxed version as context:\n%s", got)
	}
}

func TestWelcomePanel_WrapsStartupSummaryToWidth(t *testing.T) {
	SetWelcomeContext("0.2.11", 35,
		"browser",
		"clarify",
		"code_execution",
		"file",
		"homeassistant",
		"image_gen",
		"kanban",
		"memory",
		"messaging",
		"skills",
		"terminal",
	)
	defer SetWelcomeContext("", 0)

	const width = 72
	got := welcomePanel(DefaultHermesSkin(), welcomeContext{
		Model:    "gpt-5.5",
		Provider: "openai-codex",
		Runtime:  "codex_responses",
		CWD:      "…-openclaw/workspace-mineru/gormes-agent",
	}, width)

	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Fatalf("welcome panel line width = %d, want <= %d:\n%q\n\nfull panel:\n%s", w, width, line, got)
		}
	}
	if !strings.Contains(got, "35 tools") {
		t.Fatalf("wrapped summary dropped tool count:\n%s", got)
	}
	if !strings.Contains(got, "/skills list") {
		t.Fatalf("short startup tip dropped skills command:\n%s", got)
	}
}

func TestWelcomePanel_MinimalChromeDegrades(t *testing.T) {
	skin := DefaultHermesSkin()
	ctx := welcomeContext{Model: "anthropic/x"}
	got := welcomePanel(skin, ctx, 50) // < hermesMinimalChromeWidth (64)

	for _, banned := range []string{"╔", "║", "╚"} {
		if strings.Contains(got, banned) {
			t.Fatalf("minimal-chrome welcome panel must drop boxed banner marker %q at width 50:\n%s", banned, got)
		}
	}
	if !strings.Contains(got, "Type your message or /help for commands.") {
		t.Fatalf("minimal-chrome welcome panel dropped the slash tip:\n%s", got)
	}
	if !strings.Contains(got, "⚕ Gormes") {
		t.Fatalf("minimal-chrome welcome panel dropped identity:\n%s", got)
	}
}
