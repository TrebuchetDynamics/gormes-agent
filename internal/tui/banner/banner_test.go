package banner

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/testenv"
)

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
	for _, want := range []string{
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⡀⠀⣀⣀⠀⢀⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⢀⣠⣴⣾⣿⣿⣇⠸⣿⣿⠇⣸⣿⣿⣷⣦⣄⡀⠀⠀⠀⠀⠀⠀",
	} {
		if !strings.Contains(cad, want) {
			t.Fatalf("banner caduceus missing Hermes-style full-width art line %q:\n%s", want, cad)
		}
	}
	if strings.Contains(cad, "⣿⣿⠁⠈⠳⠈⠠⠋⠁") {
		t.Fatalf("banner caduceus kept malformed truncated tail line:\n%s", cad)
	}
}

func TestBannerLegacyHelpersUseSkinDerivedStyles(t *testing.T) {
	testenv.TrueColor(t)
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
	if !strings.Contains(w, "Welcome to Gormes! Type your message or /help for commands.") {
		t.Fatalf("welcome = %q, want Gormes product branding", w)
	}
	if !strings.Contains(w, "✦ Tip: /voice tts toggles TTS-only mode") {
		t.Fatalf("welcome = %q, want Hermes-style starred voice tip", w)
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

	if strings.Contains(got, "CWD:") {
		t.Fatalf("welcome panel kept Gormes-specific CWD label instead of Hermes-style bare path:\n%s", got)
	}

	if strings.Contains(got, "abcdef12…") {
		t.Fatalf("welcome panel truncated session id; Hermes shows full session ids:\n%s", got)
	}

	if !strings.Contains(got, "claude-sonnet-4-20250514 · Anthropic") {
		t.Fatalf("welcome panel missing Hermes branding-style model/provider identity line:\n%s", got)
	}
	orgGot := WelcomePanel(s, WelcomeContext{Model: "claude-opus-4-6", Provider: "nous-research"}, 100)
	if !strings.Contains(orgGot, "claude-opus-4.6 · Nous Research") {
		t.Fatalf("welcome panel should preserve Hermes branding model names and title-case provider org labels:\n%s", orgGot)
	}
	if strings.Contains(got, "anthropic/claude-sonnet-4-20250514") || strings.Contains(got, " · anthropic") {
		t.Fatalf("welcome panel leaked provider-prefixed model or raw provider label instead of Hermes branding label:\n%s", got)
	}

	// Session context lines.
	for _, want := range []string{
		"claude-sonnet-4-20250514",
		"Anthropic",
		ctx.CWD,
		"abcdef1234567890", // Hermes-style full session id
		"0.2.11",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("welcome panel missing session context %q in:\n%s", want, got)
		}
	}

	// Framed with Hermes Ink's rounded product banner at full width,
	// matching the operator-visible startup surface rather than a loose
	// rule-delimited text block.
	for _, want := range []string{"╭", "│", "╰"} {
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

	// Seeded by cmd/gormes at startup: real version + agent tool/skill counts show.
	SetWelcomeContextWithDetails(
		"0.2.11",
		42,
		70,
		[]string{"terminal", "skills"},
		[]string{"creative: ascii-art, ascii-video", "devops: kanban-orchestrator"},
	)
	defer SetWelcomeContext("", 0)
	got := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{Model: "anthropic/x"}, 100)
	if !strings.Contains(got, "0.2.11") {
		t.Fatalf("seeded version not rendered:\n%s", got)
	}
	if !strings.Contains(got, "42 tools") {
		t.Fatalf("seeded tool count not rendered:\n%s", got)
	}
	if !strings.Contains(got, "terminal: terminal") || !strings.Contains(got, "skills: /skills list") {
		t.Fatalf("seeded toolsets not rendered as Hermes-style rows:\n%s", got)
	}
	if !strings.Contains(got, "42 tools · 70 skills · /help for commands") {
		t.Fatalf("seeded skill count not rendered in Hermes-style footer:\n%s", got)
	}
	for _, want := range []string{"Available Tools", "Available Skills", "creative: ascii-art, ascii-video", "devops: kanban-orchestrator"} {
		if !strings.Contains(got, want) {
			t.Fatalf("welcome panel missing Hermes section content %q:\n%s", want, got)
		}
	}
	SetWelcomeContextWithSkillCount("0.2.11", 42, 70, "terminal")
	fallback := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{Model: "anthropic/x"}, 100)
	for _, want := range []string{
		"autonomous-ai-agents: claude-code, codex, hermes-agent, opencode",
		"creative: architecture-diagram, ascii-art, ascii-video, b...",
		"data-science: jupyter-live-kernel",
		"devops: kanban-orchestrator, kanban-worker",
		"email: himalaya",
		"general: dogfood, yuanbao",
		"github: codebase-inspection, github-auth, github-code-r...",
		"media: gif-search, heartmula, songsee, youtube-content",
		"mlops: audiocraft-audio-generation, evaluating-llms-ha...",
		"note-taking: obsidian",
		"productivity: airtable, google-workspace, maps, nano-pdf, not...",
		"red-teaming: godmode",
		"research: arxiv, blogwatcher, llm-wiki, polymarket, resea...",
		"smart-home: openhue",
		"social-media: xurl",
		"software-development: hermes-agent-skill-authoring, node-inspect-debu...",
	} {
		if !strings.Contains(fallback, want) {
			t.Fatalf("fallback welcome panel missing Hermes-style skills row %q:\n%s", want, fallback)
		}
	}
	for _, stale := range []string{"autonomous-ai-agents: claude-code, codex, gormes, opencode", "general: /help, /new, /model", "skills: /skills list"} {
		if strings.Contains(fallback, stale) {
			t.Fatalf("fallback welcome panel kept command/help row %q where Hermes shows skill category contents:\n%s", stale, fallback)
		}
	}
	for _, stale := range []string{"▾ Available Tools", "▸ Available Skills"} {
		if strings.Contains(got, stale) {
			t.Fatalf("welcome panel kept disclosure marker on Hermes section heading %q:\n%s", stale, got)
		}
	}

	// An explicit ctx value still wins over the seam (frame-derived data is
	// authoritative when present).
	ctxWin := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{Model: "m", Version: "9.9.9"}, 100)
	if !strings.Contains(ctxWin, "9.9.9") || strings.Contains(ctxWin, "0.2.11") {
		t.Fatalf("explicit ctx.Version must win over seam:\n%s", ctxWin)
	}
}

func TestWelcomePanel_OverflowRowsUseHermesWording(t *testing.T) {
	toolsets := []string{"browser", "browser-cdp", "clarify", "code_execution", "computer_use", "cronjob", "delegation", "discord", "email"}
	skills := []string{"a: one", "b: two", "c: three", "d: four", "e: five", "f: six", "g: seven", "h: eight", "i: nine", "j: ten", "k: eleven", "l: twelve", "m: thirteen", "n: fourteen", "o: fifteen"}
	got := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{Toolsets: toolsets, SkillRows: skills}, 100)
	for _, want := range []string{"(and 1 more toolsets…)", "(and 1 more categories…)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("welcome overflow row missing Hermes wording %q:\n%s", want, got)
		}
	}
	for _, stale := range []string{"more toolsets...", "more skill categories..."} {
		if strings.Contains(got, stale) {
			t.Fatalf("welcome overflow row kept non-Hermes wording %q:\n%s", stale, got)
		}
	}
}

func TestWelcomePanel_WideTerminalIncludesHeroLogoAbovePanel(t *testing.T) {
	got := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{
		Model:     "gpt-5.5",
		Version:   "0.2.11",
		ToolCount: 28,
		Toolsets:  []string{"browser", "browser-cdp", "clarify", "code_execution"},
		SessionID: "session-wide",
	}, 140)

	logoIdx := strings.Index(got, "██████")
	panelIdx := strings.Index(got, "╭")
	caduceusIdx := strings.Index(got, "⣴⣾⣿")
	toolsIdx := strings.Index(got, "Available Tools")
	wideModelIdx := strings.Index(got, "gpt-5.5")
	wideSessionIdx := strings.Index(got, "Session: session-wide")
	if logoIdx < 0 {
		t.Fatalf("wide welcome panel missing Hermes-style hero ASCII logo:\n%s", got)
	}
	if panelIdx < 0 || logoIdx > panelIdx {
		t.Fatalf("wide welcome panel should render hero logo above boxed session panel:\n%s", got)
	}
	if caduceusIdx < 0 || toolsIdx < 0 {
		t.Fatalf("wide welcome panel should render caduceus art beside the tools/skills body:\n%s", got)
	}
	lines := strings.Split(got, "\n")
	toolsLineIdx, firstCaduceusLineIdx := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "Available Tools") {
			toolsLineIdx = i
		}
		if strings.Contains(line, "⢀⣀⡀") {
			firstCaduceusLineIdx = i
		}
	}
	if toolsLineIdx < 0 || firstCaduceusLineIdx <= toolsLineIdx {
		t.Fatalf("wide welcome panel should put Available Tools on its own row before caduceus art starts like Hermes:\n%s", got)
	}
	lastToolIdx, skillsIdx := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "code_execution: execute_code") {
			lastToolIdx = i
		}
		if strings.Contains(line, "Available Skills") {
			skillsIdx = i
		}
	}
	if lastToolIdx < 0 || skillsIdx <= lastToolIdx+1 {
		t.Fatalf("wide welcome panel should leave a Hermes-style spacer row between toolsets and skills:\n%s", got)
	}
	for _, line := range lines {
		if strings.Contains(line, "browser: browser_back") && strings.Contains(line, "     browser: browser_back") {
			t.Fatalf("wide welcome panel should not add extra indentation before tool rows beyond the Hermes art gap, got line %q:\n%s", line, got)
		}
		if strings.Contains(line, "browser: browser_back") && !strings.HasPrefix(line, "│  ") {
			t.Fatalf("wide welcome panel should use Hermes-style two-cell horizontal padding inside the box, got line %q:\n%s", line, got)
		}
	}
	if wideSessionIdx < 0 || toolsIdx > wideSessionIdx {
		t.Fatalf("wide welcome panel should start the boxed body with tools before session context like Hermes:\n%s", got)
	}
	if wideModelIdx < 0 || wideModelIdx > wideSessionIdx {
		t.Fatalf("wide welcome panel should render model identity above session id like Hermes:\n%s", got)
	}
	footerLineIdx, sessionLineIdx := -1, -1
	for i, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "Session: session-wide") {
			sessionLineIdx = i
		}
		if strings.Contains(line, "28 tools") {
			footerLineIdx = i
		}
	}
	if footerLineIdx < 0 || sessionLineIdx < 0 || footerLineIdx <= sessionLineIdx+1 {
		t.Fatalf("wide welcome panel should leave a blank spacer row above the footer like Hermes:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "gpt-5.5") && strings.Index(line, "gpt-5.5") > 20 {
			t.Fatalf("wide welcome panel should place model identity in the left summary column like Hermes, got line %q:\n%s", line, got)
		}
	}
	fullLike := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{
		Model:     "claude-opus-4-6",
		Provider:  "nous-research",
		Version:   "0.2.11",
		ToolCount: 28,
		Toolsets:  []string{"browser", "browser-cdp", "clarify", "code_execution", "computer_use", "cronjob", "delegation", "discord", "email"},
		SessionID: "20260611_170004_284a49",
		CWD:       "/home/xel/.gormes",
	}, 140)
	for _, line := range strings.Split(fullLike, "\n") {
		if strings.Contains(line, "claude-opus-4.6") && !strings.Contains(line, "general: dogfood, yuanbao") {
			t.Fatalf("wide welcome panel should align model summary with the Hermes general skill row, got line %q:\n%s", line, fullLike)
		}
		if strings.Contains(line, "/home/xel/.gormes") {
			if !strings.Contains(line, "        /home/xel/.gormes") || !strings.Contains(line, "github: codebase-inspection") {
				t.Fatalf("wide welcome panel should center home path under model and align it with the Hermes github row, got line %q:\n%s", line, fullLike)
			}
		}
	}
	if strings.Contains(got, "│ ⚕ Gormes") || strings.Contains(got, "│ Go-native Hermes-compatible agent") {
		t.Fatalf("wide welcome panel should not add extra identity rows above tools; title/art already carry identity:\n%s", got)
	}
	panelCloseIdx := strings.Index(got, "╰")
	if panelCloseIdx < 0 || toolsIdx > panelCloseIdx {
		t.Fatalf("wide welcome panel should keep tools/skills inside the boxed Hermes panel:\n%s", got)
	}
	welcomeIdx := strings.Index(got, "Welcome to Gormes!")
	if welcomeIdx < 0 || welcomeIdx < panelCloseIdx {
		t.Fatalf("wide welcome copy should render below the boxed panel like Hermes:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 140 {
			t.Fatalf("wide welcome line width=%d, want <=140:\n%q\n\n%s", w, line, got)
		}
	}
}

func TestWelcomePanel_FullWidthDoesNotDuplicateBrandHeader(t *testing.T) {
	got := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{
		Model:            "gpt-5.5",
		Version:          "0.2.11",
		VersionDateAlias: "v2026.6.5",
		VersionGitCommit: "d221e369abcdef",
	}, 72)

	if count := strings.Count(got, "⚕ Gormes"); count != 1 {
		t.Fatalf("full-width welcome panel rendered duplicate product headers (%d):\n%s", count, got)
	}
	firstLine := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(firstLine, "Gormes v0.2.11 (2026.6.5) · upstream d221e369") {
		t.Fatalf("full-width welcome panel should carry version, Hermes-style date alias, and upstream commit in the top border:\n%s", got)
	}
	unknownCommit := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{Version: "0.2.11", VersionGitCommit: "unknown"}, 72)
	if strings.Contains(unknownCommit, "upstream unknown") {
		t.Fatalf("welcome panel should omit unknown commit provenance:\n%s", unknownCommit)
	}
	if strings.Contains(got, "\nversion 0.2.11") {
		t.Fatalf("full-width welcome panel repeated the boxed version as context:\n%s", got)
	}
	if strings.Contains(got, "Gormes Agent") {
		t.Fatalf("full-width welcome panel kept deprecated Gormes Agent wording:\n%s", got)
	}
	if !strings.Contains(got, "│  ⚕ Gormes") {
		t.Fatalf("full-width welcome panel should use Hermes-style two-cell horizontal padding inside the box:\n%s", got)
	}
}

func TestWelcomePanel_WrapsStartupSummaryToWidth(t *testing.T) {
	SetWelcomeContextWithDetails(
		"0.2.11",
		35,
		0,
		[]string{
			"browser",
			"clarify",
			"code_execution",
			"email",
			"homeassistant",
			"image_gen",
			"kanban",
			"memory",
			"messaging",
			"skills",
			"terminal",
		},
		[]string{"creative: architecture-diagram, ascii-art, ascii-video, b-roll, skill-name-that-should-not-wrap-onto-a-second-line"},
	)
	defer SetWelcomeContext("", 0)

	const width = 72
	got := WelcomePanel(skin.DefaultHermesSkin(), WelcomeContext{
		Model:      "gpt-5.5",
		Provider:   "openai-codex",
		Runtime:    "codex_responses",
		CWD:        "…-openclaw/workspace-mineru/gormes-agent",
		SkillCount: 70,
	}, width)

	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Fatalf("welcome panel line width = %d, want <= %d:\n%q\n\nfull panel:\n%s", w, width, line, got)
		}
	}
	if !strings.Contains(got, "35 tools") {
		t.Fatalf("wrapped summary dropped tool count:\n%s", got)
	}
	if !strings.Contains(got, "gpt-5.5 · OpenAI Codex") || !strings.Contains(got, "35 tools · 70 skills · /help for commands") {
		t.Fatalf("welcome summary missing Hermes-style model/provider line and numeric tools/skills/help footer:\n%s", got)
	}
	if strings.Contains(got, "gpt-5.5 · 35 tools") {
		t.Fatalf("welcome summary should not merge model into the count/help footer:\n%s", got)
	}
	if strings.Contains(got, "● ") || strings.Contains(got, "provider: openai-codex") {
		t.Fatalf("welcome summary kept Gormes-specific bullet/provider footer:\n%s", got)
	}
	for _, want := range []string{"browser: browser_back, browser_click, ...", "clarify: clarify", "code_execution: execute_code", "email: himalaya", "(and 3 more toolsets…)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("wrapped summary missing Hermes-style toolset row %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "browser, clarify") {
		t.Fatalf("wrapped summary kept comma-only toolset list instead of Hermes-style rows:\n%s", got)
	}
	if strings.Contains(got, "skill-name-that-should-not-wrap-onto-a-second-line") {
		t.Fatalf("Hermes-style skill category rows should truncate instead of wrapping long tails:\n%s", got)
	}
	if !strings.Contains(got, "✦ Tip: /voice tts") {
		t.Fatalf("short startup tip dropped Hermes-style voice tip:\n%s", got)
	}
}

func TestWelcomePanel_MinimalChromeDegrades(t *testing.T) {
	s := skin.DefaultHermesSkin()
	ctx := WelcomeContext{Model: "anthropic/x"}
	got := WelcomePanel(s, ctx, 50) // < hermesMinimalChromeWidth (64)

	for _, banned := range []string{"╭", "│", "╰"} {
		if strings.Contains(got, banned) {
			t.Fatalf("minimal-chrome welcome panel must drop boxed banner marker %q at width 50:\n%s", banned, got)
		}
	}
	if !strings.Contains(got, "Welcome to Gormes!") || !strings.Contains(got, "/help") {
		t.Fatalf("minimal-chrome welcome panel dropped the slash tip:\n%s", got)
	}
	if !strings.Contains(got, "⚕ Gormes") {
		t.Fatalf("minimal-chrome welcome panel dropped identity:\n%s", got)
	}
}
