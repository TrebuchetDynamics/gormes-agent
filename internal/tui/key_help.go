package tui

import (
	"github.com/charmbracelet/bubbles/key"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/keyhelp"
)

// KeyHelp is a compact, reusable keybinding hint for focused TUI modules.
// It mirrors the admin shell's help rows without importing the admin package.
type KeyHelp = keyhelp.Item

// KeyHelpProvider is the tiny seam focusable modules can implement when they
// want the active surface to advertise its local shortcuts.
type KeyHelpProvider interface {
	KeyHelp() []KeyHelp
}

func RenderKeyHelpBar(width int, skin HermesSkin, items []KeyHelp) string {
	return keyhelp.RenderBar(width, keyHelpStyles(skin), items)
}

func RenderKeyBindingHelpBar(width int, skin HermesSkin, bindings []key.Binding) string {
	return keyhelp.RenderBindingBar(width, keyHelpStyles(skin), bindings)
}

func keyHelpStyles(skin HermesSkin) keyhelp.Styles {
	styles := SkinStylesFor(skin)
	return keyhelp.Styles{
		Separator: styles.Dim,
		Key:       styles.Label,
		Desc:      styles.Dim,
	}
}
