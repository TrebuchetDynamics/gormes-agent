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

	caduceusArt = `⠀⠀⠀⠀⠀⣀⡀⣀⣀⢀⣀⡀
⠀⠀⠀⣴⣾⣿⣿⣇⠸⣿⣿⠇⣸⣿⣿
⢀⣠⣶⠿⠋⣩⣿⠻⣿⡇⢠⡄⢸⣿⠟
⠉⠉⠁⠶⠟⠋⠉⢀⣈⣁⡈⢁⣈⣁⡀
⠀⠀⠀⠀⠀⣴⣿⡿⠛⢁⡈⠛⢿⣿⣦
⠀⠀⠀⠀⠀⠿⣿⣦⣤⣈⠁⢠⣴⣿⠿
⠀⠀⠀⠀⠀⠀⠈⠉⠻⢿⣿⣦⡉⠁
⠀⠀⠀⠀⠀⠀⠀⠀⠘⢷⣦⣈⠛⠃
⠀⠀⠀⠀⠀⠀⠀⢠⣴⠦⠈⠙⠿⣦⡄
⠀⠀⠀⠀⠀⠀⠀⠸⣿⣤⡈⠁⢤⣿⠇
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠛⠷⠄
⠀⠀⠀⠀⠀⠀⠀⢀⣀⠑⢶⣄⡀
⠀⠀⠀⠀⠀⠀⠀⣿⠁⢰⡆⠈
⣿⣿⠁⠈⠳⠈⠠⠋⠁
⣿⠁
⠈⠁`

	welcomeDefault = "Welcome to Gormes. Type your message or /help for commands."
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
	Model     string
	Provider  string
	Runtime   string
	CWD       string
	SessionID string
	Version   string // best-effort; "" => version line omitted
	ToolCount int    // 0 => tools line omitted
	Toolsets  []string
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
	welcomeSeed.Version = version
	welcomeSeed.ToolCount = toolCount
	welcomeSeed.Toolsets = append([]string(nil), toolsets...)
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
	toolCount := ctx.ToolCount
	if toolCount <= 0 {
		toolCount = welcomeSeed.ToolCount
	}
	toolsets := ctx.Toolsets
	if len(toolsets) == 0 {
		toolsets = welcomeSeed.Toolsets
	}

	addCtx("CWD:", ctx.CWD)
	if id := strings.TrimSpace(ctx.SessionID); id != "" {
		addCtx("Session:", shortSessionID(id))
	}
	if s.UseMinimalChrome(width) {
		addCtx("Version:", version)
	}
	if len(ctxLines) > 0 {
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, ctxLines...)
	}
	if sections := welcomeSessionSections(toolCount, toolsets, contentWidth, dimStyle, accentStyle); len(sections) > 0 {
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, sections...)
	}
	if summary := welcomeSummaryLines(ctx, toolCount, nil, contentWidth, dimStyle, accentStyle); len(summary) > 0 {
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, summary...)
	}

	if len(body) > 0 {
		body = append(body, "")
	}
	appendWelcomeWrapped(&body, "", "Welcome to Gormes.", contentWidth, dimStyle)
	appendWelcomeWrapped(&body, "", "Type your message or /help for commands.", contentWidth, dimStyle)
	appendWelcomeWrapped(&body, "", "Tip: /new fresh session · /model switch · /skills list", contentWidth, dimStyle)
	content := strings.Join(body, "\n")

	if s.UseMinimalChrome(width) {
		return content
	}

	bannerBox := welcomeBannerBox(s, version, width)
	return strings.Join([]string{bannerBox, content}, "\n\n")
}

func welcomeContentWidth(width int) int {
	if width < 20 {
		return 20
	}
	return width
}

func welcomeSessionSections(toolCount int, toolsets []string, width int, dimStyle, accentStyle lipgloss.Style) []string {
	if toolCount <= 0 && len(toolsets) == 0 {
		return nil
	}
	var lines []string
	lines = append(lines, accentStyle.Render("▾ Available Tools"))
	if len(toolsets) > 0 {
		appendWelcomeWrapped(&lines, dimStyle.Render("  "), summarizeWelcomeToolsets(toolsets), width, dimStyle)
	} else if toolCount > 0 {
		appendWelcomeWrapped(&lines, dimStyle.Render("  "), fmt.Sprintf("%d tools", toolCount), width, dimStyle)
	}
	lines = append(lines, accentStyle.Render("▸ Available Skills"))
	return lines
}

func welcomeSummaryLines(ctx WelcomeContext, toolCount int, toolsets []string, width int, dimStyle, accentStyle lipgloss.Style) []string {
	var parts []string
	if model := strings.TrimSpace(ctx.Model); model != "" {
		parts = append(parts, model)
	}
	if toolCount > 0 {
		parts = append(parts, fmt.Sprintf("%d tools", toolCount))
	}
	if len(toolsets) > 0 {
		parts = append(parts, "toolsets: "+summarizeWelcomeToolsets(toolsets))
	}
	if provider := strings.TrimSpace(ctx.Provider + " " + ctx.Runtime); provider != "" {
		parts = append(parts, "provider: "+provider)
	}
	if len(parts) == 0 {
		return nil
	}
	var lines []string
	appendWelcomeWrapped(&lines, accentStyle.Render("● "), strings.Join(parts, " · "), width, dimStyle)
	return lines
}

func summarizeWelcomeToolsets(toolsets []string) string {
	clean := make([]string, 0, len(toolsets))
	for _, toolset := range toolsets {
		toolset = strings.TrimSpace(toolset)
		if toolset != "" {
			clean = append(clean, toolset)
		}
	}
	const limit = 6
	if len(clean) <= limit {
		return strings.Join(clean, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(clean[:limit], ", "), len(clean)-limit)
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

func welcomeBannerBox(s skin.HermesSkin, version string, width int) string {
	styles := skin.SkinStylesFor(s)
	border := styles.BannerBorder
	title := styles.Title
	dim := styles.BannerDim

	innerW := width - 4
	if innerW < 36 {
		innerW = 36
	}
	if innerW > 68 {
		innerW = 68
	}
	line1 := fitWelcomeLine("⚕ Gormes Agent - Go-native Hermes-compatible agent", innerW)
	line2Text := "Gormes Agent"
	if version = strings.TrimSpace(version); version != "" {
		line2Text += " v" + strings.TrimPrefix(version, "v")
	}
	line2 := fitWelcomeLine(line2Text, innerW)

	top := border.Render("╭" + strings.Repeat("─", innerW+2) + "╮")
	mid1 := border.Render("│ ") + title.Render(line1) + border.Render(" │")
	mid2 := border.Render("│ ") + dim.Render(line2) + border.Render(" │")
	bot := border.Render("╰" + strings.Repeat("─", innerW+2) + "╯")
	return strings.Join([]string{top, mid1, mid2, bot}, "\n")
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