package chrome

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/ansitext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar"
)

// Input is the pure input to Render. The TUI renderer pre-builds each section
// and asks this helper to assemble the bottom-pinned ordering Hermes' Ink TUI
// renders in ui-tui/src/components/appLayout.tsx.
type Input struct {
	// Width is retained as the caller's terminal column budget for future
	// optional-row renderers. The chrome assembler does not synthesize
	// standalone full-width rule rows.
	Width int

	// Conversation is the pre-rendered scrollback/output region (history
	// tail + optional draft + optional last-error block).
	Conversation string

	// Spinner is the optional activity/hint line shown above the status bar
	// while a turn is in flight or while a TUI hint is forced. Empty means
	// the row is dropped.
	Spinner string

	// StatusBar is the single-line Hermes-compatible status rule rendered by
	// RenderHermesStatusBar.
	StatusBar string
	// StatusBarMode controls whether StatusBar renders above or below the
	// prompt. Empty defaults to top to preserve the historical call shape.
	StatusBarMode statusbar.Mode

	// Prompt is the pre-rendered prompt symbol + input area block.
	Prompt string

	// VoiceStatus, ImageBar, and Completions are optional rows rendered
	// below the prompt. Empty strings are dropped.
	VoiceStatus string
	ImageBar    string
	Completions string

	// Panel is the optional modal panel (approval/clarify/secret) rendered
	// above the status bar when active. Empty means no panel is active.
	Panel string

	// TodoPanel is the optional todo list rendered when the session has
	// active (pending or in-progress) tasks. Empty means no todos to show.
	TodoPanel string

	// QueuedMessages and StickyPrompt are optional ComposerPane rows rendered
	// above the status rule. They mirror Hermes Ink's queue and sticky prompt
	// placement without coupling the pure chrome assembler to queue state.
	QueuedMessages string
	StickyPrompt   string

	// ExtensionWidgetsAbove and ExtensionWidgetsBelow are Gormes-owned,
	// typed in-process extension rows. Above-editor widgets compose with
	// panel/todo/status chrome; below-editor widgets stay below the prompt.
	ExtensionWidgetsAbove string
	ExtensionWidgetsBelow string
}

// Render assembles the bottom-pinned chrome stack used by Hermes' Ink frontend.
// Layout order matches ComposerPane in ui-tui/src/components/appLayout.tsx,
// with Gormes' prompt wrapped by continuous operator-visible rules to preserve
// the Hermes chat transcript/prompt separation expected by users:
//
//	conversation
//	(optional) spinner/hint
//	(optional) modal panel
//	(optional) extension widgets above editor
//	(optional) queued messages
//	(optional) sticky prompt
//	status bar (top mode)
//	continuous input rule
//	prompt + input area
//	continuous input rule
//	(optional) extension widgets below editor
//	status bar (bottom mode)
//	(optional) voice status
//	(optional) image bar
//	(optional) completions menu
//
// All sections are caller-rendered strings; this helper only picks order,
// and drops empty optional rows.
func Render(in Input) string {
	parts := make([]string, 0, 10)
	if in.Conversation != "" {
		parts = append(parts, in.Conversation)
		if hasBottomChrome(in) {
			parts = append(parts, "")
		}
	}
	if in.Spinner != "" {
		parts = append(parts, in.Spinner)
	}
	if in.TodoPanel != "" {
		parts = append(parts, in.TodoPanel)
	}
	if in.Panel != "" {
		parts = append(parts, in.Panel)
	}
	if in.ExtensionWidgetsAbove != "" {
		parts = append(parts, in.ExtensionWidgetsAbove)
	}
	if in.QueuedMessages != "" {
		parts = append(parts, in.QueuedMessages)
	}
	if in.StickyPrompt != "" {
		parts = append(parts, in.StickyPrompt)
	}
	statusBarMode := statusbar.NormalizeMode(in.StatusBarMode)
	if in.StatusBar != "" && statusBarMode == statusbar.ModeTop {
		parts = append(parts, in.StatusBar)
	}
	if in.Prompt != "" {
		if rule := inputRule(in.Width); rule != "" {
			parts = append(parts, rule)
		}
		parts = append(parts, in.Prompt)
		if rule := inputRule(in.Width); rule != "" {
			parts = append(parts, rule)
		}
	}
	if in.ExtensionWidgetsBelow != "" {
		parts = append(parts, in.ExtensionWidgetsBelow)
	}
	if in.StatusBar != "" && statusBarMode == statusbar.ModeBottom {
		parts = append(parts, in.StatusBar)
	}
	if in.VoiceStatus != "" {
		parts = append(parts, in.VoiceStatus)
	}
	if in.ImageBar != "" {
		parts = append(parts, in.ImageBar)
	}
	if in.Completions != "" {
		parts = append(parts, in.Completions)
	}

	return TrimTrailingLineWhitespace(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func hasBottomChrome(in Input) bool {
	return in.Spinner != "" || in.TodoPanel != "" || in.Panel != "" || in.ExtensionWidgetsAbove != "" || in.QueuedMessages != "" || in.StickyPrompt != "" || in.StatusBar != "" || in.Prompt != "" || in.ExtensionWidgetsBelow != "" || in.VoiceStatus != "" || in.ImageBar != "" || in.Completions != ""
}

func inputRule(width int) string {
	if width < 8 {
		return ""
	}
	return strings.Repeat("─", width)
}

func TrimTrailingLineWhitespace(s string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if strings.TrimRight(ansitext.StripForTUI(line), " \t") == "❯" && lipgloss.Width(ansitext.StripForTUI(line)) > lipgloss.Width("❯") {
			// Hermes' PromptPrefix reserves one cell after the prompt glyph so the
			// cursor/input starts after a visible gap; keep that single composer
			// cell while still trimming textarea right-padding on the row. Detect via
			// ANSI-stripped text so styled prompt glyphs keep the same visible gap.
			lines[i] = trimmed + " "
			continue
		}
		lines[i] = trimmed
	}
	return strings.Join(lines, "\n")
}

// UseAltScreen reports whether the bottom-pinned Hermes chrome should be
// rendered in the terminal alt-screen.
func UseAltScreen() bool {
	return true
}
