package skin

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
type ChatPalette struct {
	User       string
	Assistant  string
	ToolName   string
	ToolOutput string
	Error      string
	Prompt     string
	Separator  string
	Status     string
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

func ChatPaletteFor(skin HermesSkin) ChatPalette {
	skin = NormalizeStyleSkin(skin)
	c := skin.Colors
	return ChatPalette{
		User:       c.UILabel,
		Assistant:  c.SessionBorder,
		ToolName:   c.UIAcent,
		ToolOutput: c.BannerDim,
		Error:      c.UIError,
		Prompt:     c.Prompt,
		Separator:  c.SessionBorder,
		Status:     c.UILabel,
	}
}

func Foreground(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
}

func RenderStyle(enabled bool, style lipgloss.Style, value string) string {
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
	p := ChatPaletteFor(skin)
	c := skin.Colors
	return SkinStyles{
		User:         Foreground(p.User).Bold(true),
		Assistant:    Foreground(p.Assistant),
		ToolName:     Foreground(p.ToolName).Bold(true),
		ToolOutput:   Foreground(p.ToolOutput),
		Error:        Foreground(p.Error).Bold(true),
		Prompt:       Foreground(p.Prompt).Bold(true),
		Placeholder:  Foreground(c.Placeholder).Italic(true),
		Text:         Foreground(c.BannerText),
		Selected:     Foreground(c.UIAcent).Bold(true),
		Normal:       Foreground(c.BannerText),
		Dim:          Foreground(c.StatusBarDim),
		Separator:    Foreground(p.Separator),
		Status:       Foreground(c.StatusBarText).Background(lipgloss.Color(c.StatusBarBackground)),
		Title:        Foreground(c.BannerTitle).Bold(true),
		Label:        Foreground(c.UILabel).Bold(true),
		Accent:       Foreground(c.UIAcent).Bold(true),
		BannerBorder: Foreground(c.BannerBorder),
		BannerAccent: Foreground(c.BannerAccent).Bold(true),
		BannerDim:    Foreground(c.BannerDim),
		ActivePill:   Foreground(c.StatusBarBackground).Background(lipgloss.Color(c.UIAcent)).Bold(true),
		Good:         Foreground(c.StatusBarGood).Bold(true),
		Warn:         Foreground(c.StatusBarWarn).Bold(true),
		Bad:          Foreground(c.StatusBarBad).Bold(true),
		Critical:     Foreground(c.StatusBarCritical).Bold(true),
		Cursor:       Foreground(c.UIAcent).Reverse(true),
		FocusLine:    Foreground(c.BannerText).Background(lipgloss.Color(c.StatusBarBackground)),
	}
}

// chatStyles holds the lipgloss style for each semantic transcript role,
// derived from the active HermesSkin. view.go renders every role/chrome
// element through these named styles instead of scattered inline
// lipgloss.NewStyle calls.
type ChatStyles struct {
	User       lipgloss.Style
	Assistant  lipgloss.Style
	ToolName   lipgloss.Style
	ToolOutput lipgloss.Style
	Error      lipgloss.Style
	Prompt     lipgloss.Style
	Separator  lipgloss.Style
	Status     lipgloss.Style
}

func ChatStylesFor(skin HermesSkin) ChatStyles {
	styles := SkinStylesFor(skin)
	return ChatStyles{
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
