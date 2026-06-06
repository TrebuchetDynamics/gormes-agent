package banner

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"
)

func forceTrueColor(t *testing.T) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}

func TestBannerLogo_NonEmpty(t *testing.T) {
	s := skin.DefaultHermesSkin()
	logo := bannerLogo(s)
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

func TestBannerLegacyHelpersUseSkinDerivedStyles(t *testing.T) {
	forceTrueColor(t)
	def := skin.BuiltinSkins()["default"]
	pos := skin.BuiltinSkins()["poseidon"]

	defColors := bannerLogoColors(def)
	posColors := bannerLogoColors(pos)
	if defColors[0] != def.Colors.BannerBorder || defColors[2] != def.Colors.BannerTitle {
		t.Fatalf("default logo colors not sourced from skin: %v vs %+v", defColors, def.Colors)
	}
	if posColors[0] != pos.Colors.BannerBorder || posColors[2] != pos.Colors.BannerTitle {
		t.Fatalf("poseidon logo colors not sourced from skin: %v vs %+v", posColors, pos.Colors)
	}
	if strings.Join(defColors, ",") == strings.Join(posColors, ",") {
		t.Fatalf("switching skins did not re-theme legacy logo colors: %v", defColors)
	}

	for _, rendered := range []string{bannerLogo(pos), bannerCaduceusWithSkin(pos), bannerWelcomeWithSkin(pos)} {
		if !strings.Contains(rendered, "\x1b[") {
			t.Fatalf("legacy banner helper should render skin ANSI styles:\n%s", rendered)
		}
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
	s := skin.DefaultHermesSkin()
	ctx := WelcomeContext{
		Model:     "anthropic/claude-sonnet-4-20250514",
		Provider:  "anthropic",
		Runtime:   "anthropic_messages",
		CWD:       "~/git/gormes-agent",
		SessionID: "abcdef1234567890",
		Version:   "0.2.11",
	}
	got := WelcomePanel(s, ctx, 100)

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
	def := skin.BuiltinSkins()["default"]
	pos := skin.BuiltinSkins()["poseidon"]
	defShared := skin.SkinStylesFor(def)
	posShared := skin.SkinStylesFor(pos)

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
	if got := defShared.BannerBorder.GetForeground(); got != lipgloss.Color(dp.border) {
		t.Fatalf("default welcome border does not use shared banner border style: %v vs %v", got, dp.border)
	}
	if got := posShared.BannerAccent.GetForeground(); got != lipgloss.Color(pp.accent) {
		t.Fatalf("poseidon welcome accent does not use shared banner accent style: %v vs %v", got, pp.accent)
	}

	// Both skins still render the stable identity structure.
	ctx := WelcomeContext{Model: "m"}
	for name, sk := range map[string]skin.HermesSkin{"default": def, "poseidon": pos} {
		out := WelcomePanel(sk, ctx, 100)
		if !strings.Contains(out, "⚕ Gormes") || !strings.Contains(out, "Type your message or /help for commands.") {
			t.Fatalf("skin %s lost identity/intro structure:\n%s", name, out)
		}
	}
}

func TestWelcomePanel_VersionToolCountSeam(t *testing.T) {
	// Zero-value safe: with no seam set and an empty ctx, R1 best-effort/omit
	// behavior holds — no version/tools lines, no panic.
	SetWelcomeContext("", 0)
	bare := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{Model: "anthropic/x"}, 100)
	if strings.Contains(bare, " tools") {
		t.Fatalf("unset seam must not render a tool-count line:\n%s", bare)
	}

	// Seeded by cmd/gormes at startup: real version + agent tool count show.
	SetWelcomeContext("0.2.11", 42, "terminal", "skills")
	defer SetWelcomeContext("", 0)
	got := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{Model: "anthropic/x"}, 100)
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
	ctxWin := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{Model: "m", Version: "9.9.9"}, 100)
	if !strings.Contains(ctxWin, "9.9.9") || strings.Contains(ctxWin, "0.2.11") {
		t.Fatalf("explicit ctx.Version must win over seam:\n%s", ctxWin)
	}
}

func TestWelcomePanel_FullWidthDoesNotDuplicateBrandHeader(t *testing.T) {
	got := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{
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
	got := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{
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
	s := skin.DefaultHermesSkin()
	ctx := WelcomeContext{Model: "anthropic/x"}
	got := WelcomePanel(s, ctx, 50) // < hermesMinimalChromeWidth (64)

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