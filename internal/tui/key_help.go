package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// KeyHelp is a compact, reusable keybinding hint for focused TUI modules.
// It mirrors the admin shell's help rows without importing the admin package.
type KeyHelp struct {
	Keys        []string
	Description string
}

// KeyHelpProvider is the tiny seam focusable modules can implement when they
// want the active surface to advertise its local shortcuts.
type KeyHelpProvider interface {
	KeyHelp() []KeyHelp
}

func RenderKeyHelpBar(width int, skin HermesSkin, items []KeyHelp) string {
	return RenderKeyBindingHelpBar(width, skin, keyHelpBindings(items))
}

func RenderKeyBindingHelpBar(width int, skin HermesSkin, bindings []key.Binding) string {
	if width < 24 || len(bindings) == 0 {
		return ""
	}
	styles := SkinStylesFor(skin)
	model := help.New()
	model.Width = width
	model.ShortSeparator = styles.Dim.Render(" · ")
	model.Ellipsis = "…"
	model.Styles.ShortKey = styles.Label
	model.Styles.ShortDesc = styles.Dim
	out := model.ShortHelpView(bindings)
	return hermesStatusTrimToWidth(out, width)
}

func keyHelpBindings(items []KeyHelp) []key.Binding {
	bindings := make([]key.Binding, 0, len(items))
	for _, item := range items {
		keys := cleanKeyHelpKeys(item.Keys)
		desc := strings.TrimSpace(item.Description)
		if len(keys) == 0 || desc == "" {
			continue
		}
		bindings = append(bindings, key.NewBinding(
			key.WithKeys(keys...),
			key.WithHelp(strings.Join(keys, "/"), desc),
		))
	}
	return bindings
}

func cleanKeyHelpKeys(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			out = append(out, key)
		}
	}
	return out
}
