package skin

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin/style"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type ChatPalette = style.ChatPalette
type SkinStyles = style.SkinStyles
type ChatStyles = style.ChatStyles

func NormalizeStyleSkin(skin HermesSkin) HermesSkin { return style.NormalizeStyleSkin(skin) }

func ChatPaletteFor(skin HermesSkin) ChatPalette { return style.ChatPaletteFor(skin) }

func Foreground(hex string) lipgloss.Style { return style.Foreground(hex) }

func RenderStyle(enabled bool, skinStyle lipgloss.Style, value string) string {
	return style.RenderStyle(enabled, skinStyle, value)
}

func SkinStylesFor(skin HermesSkin) SkinStyles { return style.SkinStylesFor(skin) }

func ChatStylesFor(skin HermesSkin) ChatStyles { return style.ChatStylesFor(skin) }

func ApplyTextareaSkin(input *textarea.Model, skin HermesSkin) { style.ApplyTextareaSkin(input, skin) }

func ApplyTextInputSkin(input *textinput.Model, skin HermesSkin) {
	style.ApplyTextInputSkin(input, skin)
}
