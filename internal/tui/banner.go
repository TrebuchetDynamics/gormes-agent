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

	welcomeDefault = "Welcome to Gormes Agent! Type your message or /help for commands."
)

var logoGradientColors = []string{
	"#FFD700", "#FFD700", "#FFBF00", "#FFBF00", "#CD7F32", "#CD7F32",
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
