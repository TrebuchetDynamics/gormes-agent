package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

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

// SkinStyles is the shared Bubble Tea/Lip Gloss style set for all Gormes TUI
// surfaces. Chat, admin, and setup wizard code should derive local component
// chrome from this seam instead of rebuilding token mappings independently.
type SkinStyles struct {
	User         lipgloss.Style
	Assistant    lipgloss.Style
	ToolName     lipgloss.Style
	ToolOutput   lipgloss.Style
	Error        lipgloss.Style
	Prompt       lipgloss.Style
	Placeholder  lipgloss.Style
	Text         lipgloss.Style
	Selected     lipgloss.Style
	Normal       lipgloss.Style
	Dim          lipgloss.Style
	Separator    lipgloss.Style
	Status       lipgloss.Style
	Title        lipgloss.Style
	Label        lipgloss.Style
	Accent       lipgloss.Style
	BannerBorder lipgloss.Style
	BannerAccent lipgloss.Style
	BannerDim    lipgloss.Style
	ActivePill   lipgloss.Style
	Good         lipgloss.Style
	Warn         lipgloss.Style
	Bad          lipgloss.Style
	Critical     lipgloss.Style
	Cursor       lipgloss.Style
	FocusLine    lipgloss.Style
}

// NormalizeStyleSkin returns the default Hermes/Gormes skin when callers pass
// a zero-value skin. Subpackages use this to keep fallback policy in one place.
func NormalizeStyleSkin(skin HermesSkin) HermesSkin {
	if strings.TrimSpace(skin.Name) == "" {
		return DefaultHermesSkin()
	}
	return skin
}

func chatPaletteFor(skin HermesSkin) chatPalette {
	skin = NormalizeStyleSkin(skin)
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

func skinForeground(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
}

func renderSkinStyle(enabled bool, style lipgloss.Style, value string) string {
	if !enabled {
		return value
	}
	return style.Render(value)
}

// SkinStylesFor resolves the shared semantic style set from a skin. It is the
// common source for chat transcript styles, admin shell chrome, setup wizard
// inputs, and future Bubble Tea overlays.
func SkinStylesFor(skin HermesSkin) SkinStyles {
	skin = NormalizeStyleSkin(skin)
	p := chatPaletteFor(skin)
	c := skin.Colors
	return SkinStyles{
		User:         skinForeground(p.user).Bold(true),
		Assistant:    skinForeground(p.assistant),
		ToolName:     skinForeground(p.toolName).Bold(true),
		ToolOutput:   skinForeground(p.toolOutput),
		Error:        skinForeground(p.errorc).Bold(true),
		Prompt:       skinForeground(p.prompt).Bold(true),
		Placeholder:  skinForeground(c.Placeholder).Italic(true),
		Text:         skinForeground(c.BannerText),
		Selected:     skinForeground(c.UIAcent).Bold(true),
		Normal:       skinForeground(c.BannerText),
		Dim:          skinForeground(c.StatusBarDim),
		Separator:    skinForeground(p.separator),
		Status:       skinForeground(c.StatusBarText).Background(lipgloss.Color(c.StatusBarBackground)),
		Title:        skinForeground(c.BannerTitle).Bold(true),
		Label:        skinForeground(c.UILabel).Bold(true),
		Accent:       skinForeground(c.UIAcent).Bold(true),
		BannerBorder: skinForeground(c.BannerBorder),
		BannerAccent: skinForeground(c.BannerAccent).Bold(true),
		BannerDim:    skinForeground(c.BannerDim),
		ActivePill:   skinForeground(c.StatusBarBackground).Background(lipgloss.Color(c.UIAcent)).Bold(true),
		Good:         skinForeground(c.StatusBarGood).Bold(true),
		Warn:         skinForeground(c.StatusBarWarn).Bold(true),
		Bad:          skinForeground(c.StatusBarBad).Bold(true),
		Critical:     skinForeground(c.StatusBarCritical).Bold(true),
		Cursor:       skinForeground(c.UIAcent).Reverse(true),
		FocusLine:    skinForeground(c.BannerText).Background(lipgloss.Color(c.StatusBarBackground)),
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
	styles := SkinStylesFor(skin)
	return chatStyles{
		User:       styles.User,
		Assistant:  styles.Assistant,
		ToolName:   styles.ToolName,
		ToolOutput: styles.ToolOutput,
		Error:      styles.Error,
		Prompt:     styles.Prompt,
		Separator:  styles.Separator,
		Status:     styles.Label,
	}
}

func ApplyTextareaSkin(input *textarea.Model, skin HermesSkin) {
	if input == nil {
		return
	}
	styles := SkinStylesFor(skin)

	input.FocusedStyle.Prompt = styles.Prompt
	input.BlurredStyle.Prompt = styles.Placeholder
	input.FocusedStyle.Placeholder = styles.Placeholder
	input.BlurredStyle.Placeholder = styles.Placeholder
	input.FocusedStyle.Text = styles.Text
	input.BlurredStyle.Text = styles.Text
	input.FocusedStyle.CursorLine = styles.FocusLine
	input.BlurredStyle.CursorLine = lipgloss.NewStyle()
	input.Cursor.Style = styles.Cursor
	input.Cursor.TextStyle = styles.Text
}

func ApplyTextInputSkin(input *textinput.Model, skin HermesSkin) {
	if input == nil {
		return
	}
	styles := SkinStylesFor(skin)
	input.PromptStyle = styles.Prompt
	input.PlaceholderStyle = styles.Placeholder
	input.TextStyle = styles.Text
	input.Cursor.Style = styles.Cursor
	input.Cursor.TextStyle = styles.Text
}
