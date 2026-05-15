package tui

import "github.com/charmbracelet/lipgloss"

// chatPalette is the resolved set of semantic transcript-role colors for a
// skin. Every field is sourced from HermesSkinColors so that switching the
// active skin re-themes the whole chat surface; nothing here is a hardcoded
// palette (ccx-go's flat styles.go is a naming donor only, not its colors).
type chatPalette struct {
	user       string
	assistant  string
	toolName   string
	toolOutput string
	errorc     string
	prompt     string
	separator  string
	status     string
}

func chatPaletteFor(skin HermesSkin) chatPalette {
	c := skin.Colors
	return chatPalette{
		user:       c.UILabel,
		assistant:  c.SessionBorder,
		toolName:   c.UIAcent,
		toolOutput: c.BannerDim,
		errorc:     c.UIError,
		prompt:     c.Prompt,
		separator:  c.SessionBorder,
		status:     c.UILabel,
	}
}

// chatStyles holds the lipgloss style for each semantic transcript role,
// derived from the active HermesSkin. view.go renders every role/chrome
// element through these named styles instead of scattered inline
// lipgloss.NewStyle calls.
type chatStyles struct {
	User       lipgloss.Style
	Assistant  lipgloss.Style
	ToolName   lipgloss.Style
	ToolOutput lipgloss.Style
	Error      lipgloss.Style
	Prompt     lipgloss.Style
	Separator  lipgloss.Style
	Status     lipgloss.Style
}

func chatStylesFor(skin HermesSkin) chatStyles {
	p := chatPaletteFor(skin)
	fg := func(hex string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
	}
	return chatStyles{
		User:       fg(p.user).Bold(true),
		Assistant:  fg(p.assistant),
		ToolName:   fg(p.toolName).Bold(true),
		ToolOutput: fg(p.toolOutput),
		Error:      fg(p.errorc).Bold(true),
		Prompt:     fg(p.prompt).Bold(true),
		Separator:  fg(p.separator),
		Status:     fg(p.status),
	}
}

// defaultChatStyles is the package-level style set view.go renders through.
// The TUI currently reads DefaultHermesSkin() throughout; per-session skin
// selection wiring is tracked as a later row, and chatStylesFor already
// accepts any skin so that wiring is a one-line swap.
func defaultChatStyles() chatStyles { return chatStylesFor(DefaultHermesSkin()) }
