package keyhelp

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

type Item struct {
	Keys        []string
	Description string
}

type Styles struct {
	Separator lipgloss.Style
	Key       lipgloss.Style
	Desc      lipgloss.Style
}

func RenderBar(width int, styles Styles, items []Item) string {
	return RenderBindingBar(width, styles, bindings(items))
}

func RenderBindingBar(width int, styles Styles, bindings []key.Binding) string {
	if width < 24 || len(bindings) == 0 {
		return ""
	}
	model := help.New()
	model.Width = width
	model.ShortSeparator = styles.Separator.Render(" · ")
	model.Ellipsis = "…"
	model.Styles.ShortKey = styles.Key
	model.Styles.ShortDesc = styles.Desc
	return trimToWidth(model.ShortHelpView(bindings), width)
}

func bindings(items []Item) []key.Binding {
	bindings := make([]key.Binding, 0, len(items))
	for _, item := range items {
		keys := cleanKeys(item.Keys)
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

func cleanKeys(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			out = append(out, key)
		}
	}
	return out
}

func trimToWidth(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= maxWidth {
		return text
	}
	const ellipsis = "..."
	ellipsisWidth := lipgloss.Width(ellipsis)
	if maxWidth <= ellipsisWidth {
		out := ""
		w := 0
		for _, r := range ellipsis {
			rw := lipgloss.Width(string(r))
			if w+rw > maxWidth {
				break
			}
			out += string(r)
			w += rw
		}
		return out
	}
	var b strings.Builder
	used := 0
	for _, r := range text {
		rw := lipgloss.Width(string(r))
		if used+rw+ellipsisWidth > maxWidth {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return strings.TrimRight(b.String(), " \t") + ellipsis
}
