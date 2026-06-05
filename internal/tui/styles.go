package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"
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
type SkinStyles = skin.SkinStyles

// NormalizeStyleSkin returns the default Hermes/Gormes skin when callers pass
// a zero-value skin. Subpackages use this to keep fallback policy in one place.
func NormalizeStyleSkin(s HermesSkin) HermesSkin { return skin.NormalizeStyleSkin(s) }

func chatPaletteFor(s HermesSkin) chatPalette {
	p := skin.ChatPaletteFor(s)
	return chatPalette{
		user:       p.User,
		assistant:  p.Assistant,
		toolName:   p.ToolName,
		toolOutput: p.ToolOutput,
		errorc:     p.Error,
		prompt:     p.Prompt,
		separator:  p.Separator,
		status:     p.Status,
	}
}

func skinForeground(hex string) lipgloss.Style { return skin.Foreground(hex) }

func renderSkinStyle(enabled bool, style lipgloss.Style, value string) string {
	return skin.RenderStyle(enabled, style, value)
}

// SkinStylesFor resolves the shared semantic style set from a skin. It is the
// common source for chat transcript styles, admin shell chrome, setup wizard
// inputs, and future Bubble Tea overlays.
func SkinStylesFor(s HermesSkin) SkinStyles { return skin.SkinStylesFor(s) }

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

func chatStylesFor(s HermesSkin) chatStyles {
	styles := skin.ChatStylesFor(s)
	return chatStyles{
		User:       styles.User,
		Assistant:  styles.Assistant,
		ToolName:   styles.ToolName,
		ToolOutput: styles.ToolOutput,
		Error:      styles.Error,
		Prompt:     styles.Prompt,
		Separator:  styles.Separator,
		Status:     styles.Status,
	}
}

func ApplyTextareaSkin(input *textarea.Model, s HermesSkin) { skin.ApplyTextareaSkin(input, s) }

func ApplyTextInputSkin(input *textinput.Model, s HermesSkin) { skin.ApplyTextInputSkin(input, s) }
