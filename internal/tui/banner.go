package tui

import (
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
	addCtx("model", ctx.Model)
	addCtx("provider", strings.TrimSpace(ctx.Provider+" "+ctx.Runtime))
	addCtx("cwd", ctx.CWD)
	if id := strings.TrimSpace(ctx.SessionID); id != "" {
		addCtx("session", shortSessionID(id))
	}
	addCtx("version", ctx.Version)
	if len(ctxLines) > 0 {
		body = append(body, "")
		body = append(body, ctxLines...)
	}

	body = append(body, "")
	body = append(body, dimStyle.Render("Type your message or /help for commands."))
	content := strings.Join(body, "\n")

	if skin.UseMinimalChrome(width) {
		return content
	}

	// Frame with skin-colored horizontal rules only. Gormes' bottom-pinned
	// chrome forbids vertical pipe pairs on the response/identity row (the
	// no-sidebar contract), so the panel must never wrap content in a box
	// border. This mirrors ccx-go's header-rule pattern, not its side box.
	ruleW := width - 2
	if ruleW < 8 {
		ruleW = 8
	}
	if ruleW > 78 {
		ruleW = 78
	}
	rule := lipgloss.NewStyle().
		Foreground(lipgloss.Color(pal.border)).
		Render(strings.Repeat("─", ruleW))
	return strings.Join([]string{rule, content, rule}, "\n")
}
