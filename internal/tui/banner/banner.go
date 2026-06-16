// Package banner renders the Gormes welcome/startup panel.
package banner

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar"
)

const (
	gormesLogo = ` ██████╗  ██████╗ ██████╗ ███╗   ███╗███████╗███████╗       █████╗  ██████╗ ███████╗███╗   ██╗████████╗
██╔════╝ ██╔═══██╗██╔══██╗████╗ ████║██╔════╝██╔════╝      ██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝
██║  ███╗██║   ██║██████╔╝██╔████╔██║█████╗  ███████╗█████╗███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║
██║   ██║██║   ██║██╔══██╗██║╚██╔╝██║██╔══╝  ╚════██║╚════╝██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║
╚██████╔╝╚██████╔╝██║  ██║██║ ╚═╝ ██║███████╗███████║      ██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║
 ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝╚══════╝╚══════╝      ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝`

	caduceusArt = `⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⡀⠀⣀⣀⠀⢀⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⢀⣠⣴⣾⣿⣿⣇⠸⣿⣿⠇⣸⣿⣿⣷⣦⣄⡀⠀⠀⠀⠀⠀⠀
⠀⢀⣠⣴⣶⠿⠋⣩⡿⣿⡿⠻⣿⡇⢠⡄⢸⣿⠟⢿⣿⢿⣍⠙⠿⣶⣦⣄⡀⠀
⠀⠀⠉⠉⠁⠶⠟⠋⠀⠉⠀⢀⣈⣁⡈⢁⣈⣁⡀⠀⠉⠀⠙⠻⠶⠈⠉⠉⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣴⣿⡿⠛⢁⡈⠛⢿⣿⣦⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠿⣿⣦⣤⣈⠁⢠⣴⣿⠿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠻⢿⣿⣦⡉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⢷⣦⣈⠛⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣴⠦⠈⠙⠿⣦⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⣿⣤⡈⠁⢤⣿⠇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠛⠷⠄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⠑⢶⣄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⠁⢰⡆⠈⡿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠳⠈⣡⠞⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀`

	welcomeDefault = "Welcome to Gormes! Type your message or /help for commands.\n✦ Tip: /voice tts toggles TTS-only mode — agent replies out loud but you still type your prompts."
)

func bannerLogoColors(s skin.HermesSkin) []string {
	return skin.BannerLogoColors(s)
}

func bannerCaduceusColors(s skin.HermesSkin) []string {
	return skin.BannerCaduceusColors(s)
}

func bannerLogo(s skin.HermesSkin) string {
	var b strings.Builder
	lines := strings.Split(gormesLogo, "\n")
	colors := bannerLogoColors(s)
	for i, line := range lines {
		colorIdx := i
		if colorIdx >= len(colors) {
			colorIdx = len(colors) - 1
		}
		sty := skin.Foreground(colors[colorIdx]).Render(line)
		b.WriteString(sty)
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func bannerCaduceus() string {
	return bannerCaduceusWithSkin(skin.DefaultHermesSkin())
}

func bannerCaduceusWithSkin(s skin.HermesSkin) string {
	var b strings.Builder
	lines := strings.Split(caduceusArt, "\n")
	colors := bannerCaduceusColors(s)
	for i, line := range lines {
		colorIdx := i / 3
		if colorIdx >= len(colors) {
			colorIdx = len(colors) - 1
		}
		b.WriteString(skin.Foreground(colors[colorIdx]).Render(line))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func bannerWelcome() string {
	return bannerWelcomeWithSkin(skin.DefaultHermesSkin())
}

func bannerWelcomeWithSkin(s skin.HermesSkin) string {
	return skin.SkinStylesFor(s).Title.Render(welcomeDefault)
}

// WelcomeContext carries the live session data the welcome panel renders.
// Every field is optional: empty fields are omitted so the panel degrades
// gracefully when a value is not reachable from the render frame.
type WelcomeContext struct {
	Model            string
	Provider         string
	Runtime          string
	CWD              string
	Profile          string
	SessionID        string
	Version          string // best-effort; "" => version line omitted
	VersionDateAlias string // optional vYYYY.M.D alias rendered as a Hermes-style title date
	VersionGitCommit string // optional source commit rendered as a Hermes-style upstream suffix
	ToolCount        int    // 0 => tools line omitted
	SkillCount       int    // 0 => skills count omitted while keeping the skills footer label
	Toolsets         []string
	SkillRows        []string
}

// welcomeSeed is the process-wide startup seam: cmd/gormes seeds the real
// release version and agent tool count here (via SetWelcomeContext, wired
// through Options) because main.Version is unimportable from internal/tui
// and the tool count is absent from kernel.RenderFrame. Zero value = unset,
// in which case WelcomePanel keeps the R1 best-effort/omit behavior.
var welcomeSeed WelcomeContext

// SetWelcomeContext seeds the welcome panel with the operator-facing release
// version and agent tool count. Safe to call with zero values (resets to the
// R1 best-effort/omit behavior); idempotent and called once at startup.
func SetWelcomeContext(version string, toolCount int, toolsets ...string) {
	SetWelcomeContextWithSkillCount(version, toolCount, 0, toolsets...)
}

func SetWelcomeContextWithSkillCount(version string, toolCount int, skillCount int, toolsets ...string) {
	SetWelcomeContextWithDetails(version, toolCount, skillCount, toolsets, nil)
}

func SetWelcomeContextWithDetails(version string, toolCount int, skillCount int, toolsets []string, skillRows []string) {
	SetWelcomeContextWithBuildDetails(version, "", toolCount, skillCount, toolsets, skillRows)
}

func SetWelcomeContextWithBuildDetails(version string, versionDateAlias string, toolCount int, skillCount int, toolsets []string, skillRows []string) {
	SetWelcomeContextWithBuildProvenance(version, versionDateAlias, "", toolCount, skillCount, toolsets, skillRows)
}

func SetWelcomeContextWithBuildProvenance(version string, versionDateAlias string, gitCommit string, toolCount int, skillCount int, toolsets []string, skillRows []string) {
	welcomeSeed.Version = version
	welcomeSeed.VersionDateAlias = versionDateAlias
	welcomeSeed.VersionGitCommit = gitCommit
	welcomeSeed.ToolCount = toolCount
	welcomeSeed.SkillCount = skillCount
	welcomeSeed.Toolsets = append([]string(nil), toolsets...)
	welcomeSeed.SkillRows = append([]string(nil), skillRows...)
}

// welcomePalette is the small set of colors the welcome panel needs. It is
// always derived from the active HermesSkin's banner tokens (never hardcoded)
// so every built-in skin keeps theming the panel.
type welcomePalette struct {
	border string
	title  string
	accent string
	dim    string
}

func welcomePaletteFor(s skin.HermesSkin) welcomePalette {
	palette := skin.WelcomePaletteFor(s)
	return welcomePalette{
		border: palette.Border,
		title:  palette.Title,
		accent: palette.Accent,
		dim:    palette.Dim,
	}
}

// WelcomePanel renders the session-aware empty-transcript intro: a bordered,
// skin-themed panel that preserves the "⚕ Gormes" caduceus identity and the
// pinned intro phrasing while adding live session context. Under
// HermesSkin.UseMinimalChrome (terminal width < 64) it degrades to a compact
// non-bordered form. Layout/composition is patterned after ccx-go's
// RenderWelcomeInline; no code is copied and colors come from the skin.
func WelcomePanel(s skin.HermesSkin, ctx WelcomeContext, width int) string {
	styles := skin.SkinStylesFor(s)
	titleStyle := styles.Title
	accentStyle := styles.BannerAccent
	dimStyle := styles.BannerDim
	contentWidth := welcomeContentWidth(width)
	showCaduceus := welcomeShowCaduceusArt(width)
	if showCaduceus {
		contentWidth = width - welcomeCaduceusWidth() - lipgloss.Width(welcomeArtGap())
		if contentWidth < 40 {
			showCaduceus = false
			contentWidth = welcomeContentWidth(width)
		}
	}

	var body []string
	if s.UseMinimalChrome(width) {
		body = append(body,
			titleStyle.Render("⚕ Gormes"),
			dimStyle.Render("Go-native Hermes-compatible agent"),
		)
	}

	var ctxLines []string
	addCtx := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		appendWelcomeWrapped(&ctxLines, accentStyle.Render(label+" "), value, contentWidth, dimStyle)
	}
	// Overlay the startup seam: explicit frame-derived ctx values win;
	// otherwise fall back to the cmd/gormes-seeded version/tool count;
	// otherwise omit (R1 best-effort behavior).
	version := strings.TrimSpace(ctx.Version)
	if version == "" {
		version = strings.TrimSpace(welcomeSeed.Version)
	}
	versionDateAlias := strings.TrimSpace(ctx.VersionDateAlias)
	if versionDateAlias == "" {
		versionDateAlias = strings.TrimSpace(welcomeSeed.VersionDateAlias)
	}
	versionGitCommit := strings.TrimSpace(ctx.VersionGitCommit)
	if versionGitCommit == "" {
		versionGitCommit = strings.TrimSpace(welcomeSeed.VersionGitCommit)
	}
	toolCount := ctx.ToolCount
	if toolCount <= 0 {
		toolCount = welcomeSeed.ToolCount
	}
	skillCount := ctx.SkillCount
	if skillCount <= 0 {
		skillCount = welcomeSeed.SkillCount
	}
	toolsets := ctx.Toolsets
	if len(toolsets) == 0 {
		toolsets = welcomeSeed.Toolsets
	}
	skillRows := ctx.SkillRows
	if len(skillRows) == 0 {
		skillRows = welcomeSeed.SkillRows
	}

	if cwd := strings.TrimSpace(ctx.CWD); cwd != "" {
		appendWelcomeWrapped(&ctxLines, "", cwd, contentWidth, dimStyle)
	}
	if id := strings.TrimSpace(ctx.SessionID); id != "" {
		addCtx("Session:", id)
	}
	if s.UseMinimalChrome(width) {
		addCtx("Version:", version)
	}
	sessionSections := welcomeSessionSections(toolCount, toolsets, skillRows, contentWidth, dimStyle, accentStyle)
	identityLines := welcomeIdentityLines(ctx, contentWidth, dimStyle)
	footerLines := welcomeFooterLines(toolCount, skillCount, nil, contentWidth, dimStyle)
	if len(sessionSections) > 0 {
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, sessionSections...)
	}
	if len(identityLines) > 0 {
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, identityLines...)
	}
	if len(ctxLines) > 0 {
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, ctxLines...)
	}
	if len(footerLines) > 0 {
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, footerLines...)
	}

	if len(body) > 0 {
		body = append(body, "")
	}
	appendWelcomeWrapped(&body, "", "Welcome to Gormes! Type your message or /help for commands.", contentWidth, dimStyle)
	appendWelcomeWrapped(&body, "", "✦ Tip: /voice tts toggles TTS-only mode — agent replies out loud but you still type your prompts.", contentWidth, dimStyle)
	content := strings.Join(body, "\n")

	if s.UseMinimalChrome(width) {
		return content
	}

	var sections []string
	if showCaduceus {
		left := "\n" + bannerCaduceusWithSkin(s)
		leftSummary := welcomeWideLeftSummary(identityLines, ctxLines)
		if len(leftSummary) > 0 {
			left += "\n\n" + strings.Join(leftSummary, "\n")
		}
		rightLines := append([]string{}, sessionSections...)
		if len(footerLines) > 0 {
			if len(rightLines) > 0 {
				rightLines = append(rightLines, "")
			}
			rightLines = append(rightLines, footerLines...)
		}
		panelContent := welcomeSideBySide(left, strings.Join(rightLines, "\n"))
		var introLines []string
		appendWelcomeWrapped(&introLines, "", "Welcome to Gormes! Type your message or /help for commands.", welcomeContentWidth(width), dimStyle)
		appendWelcomeWrapped(&introLines, "", "✦ Tip: /voice tts toggles TTS-only mode — agent replies out loud but you still type your prompts.", welcomeContentWidth(width), dimStyle)
		sections = []string{welcomeWidePanelBox(s, version, versionDateAlias, versionGitCommit, panelContent, width), strings.Join(introLines, "\n")}
	} else {
		bannerBox := welcomeBannerBox(s, version, versionDateAlias, versionGitCommit, width)
		sections = []string{bannerBox, content}
	}
	if welcomeShowHeroLogo(width) {
		sections = append([]string{bannerLogo(s)}, sections...)
	}
	return strings.Join(sections, "\n\n")
}

func welcomeShowHeroLogo(width int) bool {
	return width >= welcomeLogoWidth()+2
}

func welcomeShowCaduceusArt(width int) bool {
	return width >= 110
}

func welcomeArtGap() string {
	return "   "
}

func welcomeCaduceusWidth() int {
	maxWidth := 0
	for _, line := range strings.Split(caduceusArt, "\n") {
		if w := lipgloss.Width(line); w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

func welcomeWideLeftSummary(identityLines []string, ctxLines []string) []string {
	leftSummary := append([]string{}, identityLines...)
	if len(identityLines) == 0 || len(ctxLines) == 0 {
		return append(leftSummary, ctxLines...)
	}
	targetWidth := lipgloss.Width(identityLines[0])
	for i, line := range ctxLines {
		if i == 0 && !strings.Contains(line, "Session:") && targetWidth > lipgloss.Width(line) {
			pad := (targetWidth - lipgloss.Width(line)) / 2
			line = strings.Repeat(" ", pad) + line
		}
		leftSummary = append(leftSummary, line)
	}
	return leftSummary
}

func welcomeSideBySide(left, right string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	leftWidth := 0
	for _, line := range leftLines {
		if w := lipgloss.Width(line); w > leftWidth {
			leftWidth = w
		}
	}
	gap := welcomeArtGap()
	count := len(leftLines)
	if len(rightLines) > count {
		count = len(rightLines)
	}
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		leftLine := ""
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		rightLine := ""
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}
		if rightLine == "" {
			out = append(out, leftLine)
			continue
		}
		pad := leftWidth - lipgloss.Width(leftLine)
		if pad < 0 {
			pad = 0
		}
		out = append(out, leftLine+strings.Repeat(" ", pad)+gap+rightLine)
	}
	return strings.Join(out, "\n")
}

func welcomeLogoWidth() int {
	maxWidth := 0
	for _, line := range strings.Split(gormesLogo, "\n") {
		if w := lipgloss.Width(line); w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

func welcomeContentWidth(width int) int {
	if width < 20 {
		return 20
	}
	return width
}

func welcomeSessionSections(toolCount int, toolsets []string, skillRows []string, width int, dimStyle, accentStyle lipgloss.Style) []string {
	if toolCount <= 0 && len(toolsets) == 0 {
		return nil
	}
	var lines []string
	lines = append(lines, accentStyle.Render("Available Tools"))
	if len(toolsets) > 0 {
		for _, line := range welcomeToolsetRows(toolsets) {
			appendWelcomeSingleLine(&lines, "", line, width, dimStyle)
		}
	} else if toolCount > 0 {
		appendWelcomeWrapped(&lines, "", fmt.Sprintf("%d tools", toolCount), width, dimStyle)
	}
	lines = append(lines, "")
	lines = append(lines, accentStyle.Render("Available Skills"))
	if len(skillRows) > 0 {
		for _, line := range welcomeSkillRows(skillRows) {
			appendWelcomeSingleLine(&lines, "", line, width, dimStyle)
		}
	} else {
		for _, line := range fallbackWelcomeSkillRows() {
			appendWelcomeSingleLine(&lines, "", line, width, dimStyle)
		}
	}
	return lines
}

func fallbackWelcomeSkillRows() []string {
	return []string{
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
	}
}

func welcomeSkillRows(rows []string) []string {
	clean := make([]string, 0, len(rows))
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row != "" {
			clean = append(clean, row)
		}
	}
	const limit = 14
	if len(clean) <= limit {
		return clean
	}
	out := append([]string(nil), clean[:limit]...)
	out = append(out, fmt.Sprintf("(and %d more categories…)", len(clean)-limit))
	return out
}

func welcomeIdentityLines(ctx WelcomeContext, width int, dimStyle lipgloss.Style) []string {
	var lines []string
	if model := welcomeModelLabel(ctx.Model); model != "" {
		if provider := welcomeProviderLabel(ctx.Provider); provider != "" {
			model += " · " + provider
		}
		appendWelcomeWrapped(&lines, "", model, width, dimStyle)
	}
	return lines
}

func welcomeFooterLines(toolCount int, skillCount int, toolsets []string, width int, dimStyle lipgloss.Style) []string {
	var lines []string
	var footer []string
	if toolCount > 0 {
		footer = append(footer, fmt.Sprintf("%d tools", toolCount))
		if skillCount > 0 {
			footer = append(footer, fmt.Sprintf("%d skills", skillCount))
		} else {
			footer = append(footer, "skills")
		}
	}
	if len(toolsets) > 0 {
		footer = append(footer, "toolsets: "+summarizeWelcomeToolsets(toolsets))
	}
	if toolCount > 0 {
		footer = append(footer, "/help for commands")
	}
	if len(footer) > 0 {
		appendWelcomeWrapped(&lines, "", strings.Join(footer, " · "), width, dimStyle)
	}
	return lines
}

func welcomeSummaryLines(ctx WelcomeContext, toolCount int, skillCount int, toolsets []string, width int, dimStyle lipgloss.Style) []string {
	lines := welcomeIdentityLines(ctx, width, dimStyle)
	lines = append(lines, welcomeFooterLines(toolCount, skillCount, toolsets, width, dimStyle)...)
	return lines
}

func welcomeModelLabel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	if i := strings.LastIndex(model, "/"); i >= 0 {
		model = model[i+1:]
	}
	model = strings.TrimSuffix(model, ".gguf")
	model = strings.ReplaceAll(model, "_", "-")
	parts := strings.Split(model, "-")
	if len(parts) >= 2 && bannerSingleDigit(parts[len(parts)-2]) && bannerSingleDigit(parts[len(parts)-1]) {
		parts[len(parts)-2] = parts[len(parts)-2] + "." + parts[len(parts)-1]
		parts = parts[:len(parts)-1]
		model = strings.Join(parts, "-")
	}
	return strings.TrimSpace(model)
}

func bannerSingleDigit(value string) bool {
	return len(value) == 1 && value[0] >= '0' && value[0] <= '9'
}

func welcomeProviderLabel(provider string) string {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return ""
	}
	switch strings.ToLower(provider) {
	case "anthropic":
		return "Anthropic"
	case "openai":
		return "OpenAI"
	case "openai-codex", "openai_codex":
		return "OpenAI Codex"
	case "openrouter":
		return "OpenRouter"
	case "gemini", "google", "google_code_assist", "gemini_cloudcode":
		return "Google"
	default:
		return titleWords(strings.NewReplacer("-", " ", "_", " ").Replace(provider))
	}
}

func titleWords(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		lower := strings.ToLower(word)
		if lower == "" {
			continue
		}
		words[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(words, " ")
}

func summarizeWelcomeToolsets(toolsets []string) string {
	clean := cleanWelcomeToolsets(toolsets)
	const limit = 6
	if len(clean) <= limit {
		return strings.Join(clean, ", ")
	}
	return fmt.Sprintf("%s (and %d more toolsets…)", strings.Join(clean[:limit], ", "), len(clean)-limit)
}

func welcomeToolsetRows(toolsets []string) []string {
	clean := cleanWelcomeToolsets(toolsets)
	const limit = 8
	rows := make([]string, 0, len(clean)+1)
	shown := len(clean)
	if shown > limit {
		shown = limit
	}
	for _, toolset := range clean[:shown] {
		rows = append(rows, toolset+": "+welcomeToolsetPreview(toolset))
	}
	if remaining := len(clean) - shown; remaining > 0 {
		rows = append(rows, fmt.Sprintf("(and %d more toolsets…)", remaining))
	}
	return rows
}

func welcomeToolsetPreview(toolset string) string {
	switch strings.ToLower(strings.TrimSpace(toolset)) {
	case "browser":
		return "browser_back, browser_click, ..."
	case "browser-cdp", "browser_cdp":
		return "browser_cdp, browser_dialog"
	case "clarify":
		return "clarify"
	case "code_execution":
		return "execute_code"
	case "computer_use":
		return "computer_use"
	case "cronjob":
		return "cronjob"
	case "delegation":
		return "delegate_task"
	case "email":
		return "himalaya"
	case "terminal":
		return "terminal"
	case "skills":
		return "/skills list"
	default:
		return strings.TrimSpace(toolset)
	}
}

func cleanWelcomeToolsets(toolsets []string) []string {
	clean := make([]string, 0, len(toolsets))
	for _, toolset := range toolsets {
		toolset = strings.TrimSpace(toolset)
		if toolset != "" {
			clean = append(clean, toolset)
		}
	}
	return clean
}

func appendWelcomeSingleLine(lines *[]string, prefix, text string, width int, style lipgloss.Style) {
	available := width - lipgloss.Width(prefix)
	if available < 1 {
		available = width
		prefix = ""
	}
	text = strings.TrimSpace(text)
	if lipgloss.Width(text) > available {
		text = statusbar.HermesTrimToWidth(text, available)
	}
	*lines = append(*lines, prefix+style.Render(text))
}

func appendWelcomeWrapped(lines *[]string, prefix, text string, width int, style lipgloss.Style) {
	available := width - lipgloss.Width(prefix)
	if available < 8 {
		available = width
		prefix = ""
	}
	wrapped := wrapWelcomeWords(text, available)
	if len(wrapped) == 0 {
		*lines = append(*lines, prefix)
		return
	}
	continuation := ""
	if prefix != "" {
		continuation = strings.Repeat(" ", lipgloss.Width(prefix))
	}
	for i, line := range wrapped {
		p := prefix
		if i > 0 {
			p = continuation
		}
		*lines = append(*lines, p+style.Render(line))
	}
}

func wrapWelcomeWords(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if width <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	lines := make([]string, 0, 1)
	current := ""
	for _, word := range words {
		if lipgloss.Width(word) > width {
			word = statusbar.HermesTrimToWidth(word, width)
		}
		if current == "" {
			current = word
			continue
		}
		candidate := current + " " + word
		if lipgloss.Width(candidate) > width {
			lines = append(lines, current)
			current = word
			continue
		}
		current = candidate
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func welcomeWidePanelBox(s skin.HermesSkin, version string, versionDateAlias string, gitCommit string, content string, width int) string {
	styles := skin.SkinStylesFor(s)
	border := styles.BannerBorder

	innerW := width - 6
	if innerW < 36 {
		innerW = 36
	}
	borderTitle := welcomePanelTitle(version, versionDateAlias, gitCommit)

	body := strings.Split(content, "\n")

	lines := make([]string, 0, len(body)+2)
	lines = append(lines, border.Render("╭"+welcomeTitledRule(borderTitle, innerW+4)+"╮"))
	for _, line := range body {
		lines = append(lines, border.Render("│  ")+padWelcomeRenderedLine(line, innerW)+border.Render("  │"))
	}
	lines = append(lines, border.Render("╰"+strings.Repeat("─", innerW+4)+"╯"))
	return strings.Join(lines, "\n")
}

func padWelcomeRenderedLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) > width {
		value = statusbar.HermesTrimToWidth(value, width)
	}
	padding := width - lipgloss.Width(value)
	if padding > 0 {
		value += strings.Repeat(" ", padding)
	}
	return value
}

func welcomeBannerBox(s skin.HermesSkin, version string, versionDateAlias string, gitCommit string, width int) string {
	styles := skin.SkinStylesFor(s)
	border := styles.BannerBorder
	title := styles.Title
	dim := styles.BannerDim

	innerW := width - 6
	if innerW < 36 {
		innerW = 36
	}
	if innerW > 68 {
		innerW = 68
	}
	line1 := fitWelcomeLine("⚕ Gormes", innerW)
	line2 := fitWelcomeLine("Go-native Hermes-compatible agent", innerW)
	borderTitle := welcomePanelTitle(version, versionDateAlias, gitCommit)

	top := border.Render("╭" + welcomeTitledRule(borderTitle, innerW+4) + "╮")
	mid1 := border.Render("│  ") + title.Render(line1) + border.Render("  │")
	mid2 := border.Render("│  ") + dim.Render(line2) + border.Render("  │")
	bot := border.Render("╰" + strings.Repeat("─", innerW+4) + "╯")
	return strings.Join([]string{top, mid1, mid2, bot}, "\n")
}

func welcomePanelTitle(version string, versionDateAlias string, gitCommit string) string {
	borderTitle := "Gormes"
	if version = strings.TrimSpace(version); version != "" {
		borderTitle += " v" + strings.TrimPrefix(version, "v")
	}
	if versionDateAlias = strings.TrimSpace(versionDateAlias); versionDateAlias != "" {
		borderTitle += " (" + strings.TrimPrefix(versionDateAlias, "v") + ")"
	}
	if short := welcomeShortGitCommit(gitCommit); short != "" {
		borderTitle += " · upstream " + short
	}
	return borderTitle
}

func welcomeShortGitCommit(gitCommit string) string {
	gitCommit = strings.TrimSpace(gitCommit)
	if gitCommit == "" || strings.EqualFold(gitCommit, "unknown") {
		return ""
	}
	if len(gitCommit) > 8 {
		return gitCommit[:8]
	}
	return gitCommit
}

func welcomeTitledRule(title string, width int) string {
	if width <= 0 {
		return ""
	}
	label := " " + strings.TrimSpace(title) + " "
	if strings.TrimSpace(title) == "" || lipgloss.Width(label) > width-2 {
		return strings.Repeat("─", width)
	}
	remaining := width - lipgloss.Width(label)
	left := remaining / 2
	right := remaining - left
	return strings.Repeat("─", left) + label + strings.Repeat("─", right)
}

func fitWelcomeLine(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) > width {
		value = truncateEllipsis(value, width)
	}
	padding := width - lipgloss.Width(value)
	if padding > 0 {
		value += strings.Repeat(" ", padding)
	}
	return value
}

func shortSessionID(id string) string {
	if id == "" {
		return "new"
	}
	if len(id) <= 8 {
		return id
	}
	return id[:8] + "…"
}

func truncateEllipsis(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
