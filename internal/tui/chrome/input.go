package chrome

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/ansitext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/keyhelp"
)

// TextInput is the pure presentation input for a labeled TUI text input.
// Editing behavior remains owned by the caller's Bubble Tea widget.
type TextInput struct {
	Width  int
	Label  string
	Hint   string
	Value  string
	Styles TextInputStyles
}

// TextInputStyles are the only styles needed by the reusable input chrome.
type TextInputStyles struct {
	Label lipgloss.Style
	Dim   lipgloss.Style
}

// ComposerInput wraps a rendered chat textarea with contextual input chrome.
type ComposerInput struct {
	Width     int
	Prompt    string
	Draft     string
	Focused   bool
	Multiline bool
	Styles    TextInputStyles
	KeyHelp   keyhelp.Styles
}

func RenderTextInput(in TextInput) string {
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
	gap := "  "
	hint := strings.TrimSpace(in.Hint)
	if hint == "" {
		return in.Styles.Label.Render(label) + "\n" + value
	}
	maxHintWidth := in.Width - lipgloss.Width(label) - lipgloss.Width(gap)
	if maxHintWidth < 8 {
		return value
	}
	hint = ansitext.TrimToWidth(hint, maxHintWidth)
	header := in.Styles.Label.Render(label) + in.Styles.Dim.Render(gap+hint)
	return lipgloss.JoinVertical(lipgloss.Left, header, value)
}

func RenderComposerInput(in ComposerInput) string {
	prompt := strings.TrimRight(in.Prompt, "\n")
	if ComposerInputExtraRows(in.Width) == 0 || strings.TrimSpace(prompt) == "" {
		return prompt
	}

	title := "Ask Gormes"
	if !in.Focused {
		title = "Composer paused"
	}
	return RenderTextInput(TextInput{
		Width:  in.Width,
		Label:  title,
		Hint:   keyhelp.RenderBar(composerInputHintWidth(in.Width, title), in.KeyHelp, ComposerKeyHelp(in.Draft, in.Multiline)),
		Value:  prompt,
		Styles: in.Styles,
	})
}

func ComposerKeyHelp(draft string, multiline bool) []keyhelp.Item {
	if multiline {
		return []keyhelp.Item{{Keys: []string{"Enter"}, Description: "send"}}
	}
	trimmed := strings.TrimSpace(draft)
	if strings.HasPrefix(trimmed, "/") {
		return []keyhelp.Item{{Keys: []string{"Tab"}, Description: "complete"}, {Keys: []string{"↑", "↓"}, Description: "select"}}
	}
	if trimmed == "" {
		return []keyhelp.Item{{Keys: []string{"Enter"}, Description: "send"}, {Keys: []string{"/"}, Description: "commands"}}
	}
	return []keyhelp.Item{{Keys: []string{"Enter"}, Description: "send"}, {Keys: []string{"Shift+Enter"}, Description: "newline"}}
}

func composerInputHintWidth(width int, title string) int {
	return width - lipgloss.Width(title) - lipgloss.Width("  ")
}

func ComposerInputExtraRows(width int) int {
	if width < 40 {
		return 0
	}
	return 1
}

func ShowComposerInput(width int, height int) bool {
	return width >= 40 && height >= 18
}
