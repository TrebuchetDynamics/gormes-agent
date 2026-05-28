package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TextInputChrome is the small reusable presentation wrapper around TUI text
// inputs. It deliberately owns presentation only; Bubble Tea textinput/textarea
// widgets still own editing behaviour.
type TextInputChrome struct {
	Width int
	Label string
	Hint  string
	Value string
	Skin  HermesSkin
}

func RenderTextInputChrome(in TextInputChrome) string {
	value := strings.TrimRight(in.Value, "\n")
	if strings.TrimSpace(value) == "" {
		return value
	}
	if in.Width < 40 {
		return value
	}
	label := strings.TrimSpace(in.Label)
	if label == "" {
		return value
	}
	styles := SkinStylesFor(in.Skin)
	gap := "  "
	hint := strings.TrimSpace(in.Hint)
	if hint == "" {
		return styles.Label.Render(label) + "\n" + value
	}
	maxHintWidth := in.Width - lipgloss.Width(label) - lipgloss.Width(gap)
	if maxHintWidth < 8 {
		return value
	}
	hint = trimToWidth(hint, maxHintWidth)
	header := styles.Label.Render(label) + styles.Dim.Render(gap+hint)
	return lipgloss.JoinVertical(lipgloss.Left, header, value)
}

// ComposerInputChrome wraps the Bubble Tea chat textarea with a contextual
// input affordance. The textarea still owns editing behaviour.
type ComposerInputChrome struct {
	Width     int
	Prompt    string
	Draft     string
	Skin      HermesSkin
	Focused   bool
	Multiline bool
}

func RenderComposerInputChrome(in ComposerInputChrome) string {
	prompt := strings.TrimRight(in.Prompt, "\n")
	if composerInputChromeExtraRows(in.Width) == 0 || strings.TrimSpace(prompt) == "" {
		return prompt
	}

	title := "Ask Gormes"
	if !in.Focused {
		title = "Composer paused"
	}
	return RenderTextInputChrome(TextInputChrome{
		Width: in.Width,
		Label: title,
		Hint:  RenderKeyHelpBar(composerInputHintWidth(in.Width, title), in.Skin, in.KeyHelp()),
		Value: prompt,
		Skin:  in.Skin,
	})
}

func (in ComposerInputChrome) KeyHelp() []KeyHelp {
	if in.Multiline {
		return []KeyHelp{{Keys: []string{"Enter"}, Description: "send"}}
	}
	trimmed := strings.TrimSpace(in.Draft)
	if strings.HasPrefix(trimmed, "/") {
		return []KeyHelp{{Keys: []string{"Tab"}, Description: "complete"}, {Keys: []string{"↑", "↓"}, Description: "select"}}
	}
	if trimmed == "" {
		return []KeyHelp{{Keys: []string{"Enter"}, Description: "send"}, {Keys: []string{"/"}, Description: "commands"}}
	}
	return []KeyHelp{{Keys: []string{"Enter"}, Description: "send"}, {Keys: []string{"Shift+Enter"}, Description: "newline"}}
}

func composerInputHintWidth(width int, title string) int {
	return width - lipgloss.Width(title) - lipgloss.Width("  ")
}

func composerInputChromeExtraRows(width int) int {
	if width < 40 {
		return 0
	}
	return 1
}

func showComposerInputChrome(width int, height int) bool {
	return width >= 40 && height >= 18
}
