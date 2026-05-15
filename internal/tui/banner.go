package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

var logoGradientColors = []string{
	"#1D4ED8", "#2563EB", "#3B82F6", "#60A5FA", "#93C5FD", "#BFDBFE",
}

var caduceusGradient = []struct{ line, color string }{
	{line: "⠀⠀⠀⠀⠀⣀⡀⣀⣀⢀⣀⡀", color: "#CD7F32"},
	{line: "⠀⠀⠀⣴⣾⣿⣿⣇⠸⣿⣿⠇⣸⣿⣿", color: "#CD7F32"},
	{line: "⢀⣠⣶⠿⠋⣩⣿⠻⣿⡇⢠⡄⢸⣿⠟", color: "#FFBF00"},
	{line: "⠉⠉⠁⠶⠟⠋⠉⢀⣈⣁⡈⢁⣈⣁⡀", color: "#FFBF00"},
	{line: "⠀⠀⠀⠀⠀⣴⣿⡿⠛⢁⡈⠛⢿⣿⣦", color: "#FFD700"},
	{line: "⠀⠀⠀⠀⠀⠿⣿⣦⣤⣈⠁⢠⣴⣿⠿", color: "#FFD700"},
	{line: "⠀⠀⠀⠀⠀⠀⠈⠉⠻⢿⣿⣦⡉⠁", color: "#FFBF00"},
	{line: "⠀⠀⠀⠀⠀⠀⠀⠀⠘⢷⣦⣈⠛⠃", color: "#FFBF00"},
	{line: "⠀⠀⠀⠀⠀⠀⠀⢠⣴⠦⠈⠙⠿⣦⡄", color: "#CD7F32"},
	{line: "⠀⠀⠀⠀⠀⠀⠀⠸⣿⣤⡈⠁⢤⣿⠇", color: "#CD7F32"},
	{line: "⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠛⠷⠄", color: "#B8860B"},
	{line: "⠀⠀⠀⠀⠀⠀⠀⢀⣀⠑⢶⣄⡀", color: "#B8860B"},
	{line: "⠀⠀⠀⠀⠀⠀⠀⣿⠁⢰⡆⠈", color: "#B8860B"},
	{line: "⣿⣿⠁⠈⠳⠈⠠⠋⠁", color: "#B8860B"},
	{line: "⣿⠁", color: "#B8860B"},
	{line: "⠈⠁", color: "#B8860B"},
}

func bannerLogo(skin HermesSkin) string {
	var b strings.Builder
	lines := strings.Split(gormesLogo, "\n")
	for i, line := range lines {
		colorIdx := i
		if colorIdx >= len(logoGradientColors) {
			colorIdx = len(logoGradientColors) - 1
		}
		s := lipgloss.NewStyle().Foreground(lipgloss.Color(logoGradientColors[colorIdx])).Render(line)
		b.WriteString(s)
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func bannerCaduceus() string {
	var b strings.Builder
	for _, entry := range caduceusGradient {
		s := lipgloss.NewStyle().Foreground(lipgloss.Color(entry.color)).Render(entry.line)
		b.WriteString(s)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func bannerWelcome() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true).Render(welcomeDefault)
}

// welcomeContext carries the live session data the welcome panel renders.
// Every field is optional: empty fields are omitted so the panel degrades
// gracefully when a value is not reachable from the render frame.
type welcomeContext struct {
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
// through tui.Options) because main.Version is unimportable from internal/tui
// and the tool count is absent from kernel.RenderFrame. Zero value = unset,
// in which case welcomePanel keeps the R1 best-effort/omit behavior.
var welcomeSeed welcomeContext

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

func welcomePaletteFor(skin HermesSkin) welcomePalette {
	return welcomePalette{
		border: skin.Colors.BannerBorder,
		title:  skin.Colors.BannerTitle,
		accent: skin.Colors.BannerAccent,
		dim:    skin.Colors.BannerDim,
	}
}

// welcomePanel renders the session-aware empty-transcript intro: a bordered,
// skin-themed panel that preserves the "⚕ Gormes" caduceus identity and the
// pinned intro phrasing while adding live session context. Under
// HermesSkin.UseMinimalChrome (terminal width < 64) it degrades to a compact
// non-bordered form. Layout/composition is patterned after ccx-go's
// RenderWelcomeInline; no code is copied and colors come from the skin.
func welcomePanel(skin HermesSkin, ctx welcomeContext, width int) string {
	pal := welcomePaletteFor(skin)
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(pal.title)).Bold(true)
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(pal.accent))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(pal.dim))

	body := []string{
		titleStyle.Render("⚕ Gormes"),
		dimStyle.Render("Go-native Hermes-compatible agent"),
	}

	var ctxLines []string
	addCtx := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		ctxLines = append(ctxLines, accentStyle.Render(label+" ")+dimStyle.Render(value))
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

	addCtx("cwd", ctx.CWD)
	if id := strings.TrimSpace(ctx.SessionID); id != "" {
		addCtx("session", shortSessionID(id))
	}
	addCtx("version", version)
	if len(ctxLines) > 0 {
		body = append(body, "")
		body = append(body, ctxLines...)
	}
	if summary := welcomeSummary(ctx, toolCount, toolsets, dimStyle, accentStyle); summary != "" {
		body = append(body, "")
		body = append(body, summary)
	}

	body = append(body, "")
	body = append(body, dimStyle.Render("Welcome to Gormes."))
	body = append(body, dimStyle.Render("Type your message or /help for commands."))
	body = append(body, dimStyle.Render("Tip: Type /new for a fresh session, /model to switch models, or /skills list."))
	content := strings.Join(body, "\n")

	if skin.UseMinimalChrome(width) {
		return content
	}

	banner := welcomeBannerBox(skin, version, width)
	return strings.Join([]string{banner, content}, "\n\n")
}

func welcomeSummary(ctx welcomeContext, toolCount int, toolsets []string, dimStyle, accentStyle lipgloss.Style) string {
	var parts []string
	if model := strings.TrimSpace(ctx.Model); model != "" {
		parts = append(parts, model)
	}
	if toolCount > 0 {
		parts = append(parts, fmt.Sprintf("%d tools", toolCount))
	}
	if len(toolsets) > 0 {
		parts = append(parts, "toolsets: "+strings.Join(toolsets, ", "))
	}
	if provider := strings.TrimSpace(ctx.Provider + " " + ctx.Runtime); provider != "" {
		parts = append(parts, "provider: "+provider)
	}
	if len(parts) == 0 {
		return ""
	}
	return accentStyle.Render("● ") + dimStyle.Render(strings.Join(parts, " · "))
}

func welcomeBannerBox(skin HermesSkin, version string, width int) string {
	pal := welcomePaletteFor(skin)
	border := lipgloss.NewStyle().Foreground(lipgloss.Color(pal.border))
	title := lipgloss.NewStyle().Foreground(lipgloss.Color(pal.title)).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(pal.dim))

	innerW := width - 4
	if innerW < 36 {
		innerW = 36
	}
	if innerW > 68 {
		innerW = 68
	}
	line1 := fitWelcomeLine("⚕ Gormes - Go-native Hermes-compatible agent", innerW)
	line2Text := "Gormes"
	if version = strings.TrimSpace(version); version != "" {
		line2Text += " v" + strings.TrimPrefix(version, "v")
	}
	line2 := fitWelcomeLine(line2Text, innerW)

	top := border.Render("╔" + strings.Repeat("═", innerW+2) + "╗")
	mid1 := border.Render("║ ") + title.Render(line1) + border.Render(" ║")
	mid2 := border.Render("║ ") + dim.Render(line2) + border.Render(" ║")
	bot := border.Render("╚" + strings.Repeat("═", innerW+2) + "╝")
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
