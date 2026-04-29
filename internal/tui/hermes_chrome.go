package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HermesChromeInput is the pure input to RenderHermesChrome. The TUI renderer
// pre-builds each section (conversation tail, optional spinner/hint, status
// bar, prompt+input area, optional voice/image/completion rows) and asks this
// helper to assemble the bottom-pinned ordering Hermes' prompt_toolkit run
// loop produces in cli.py:_build_tui_layout_children.
type HermesChromeInput struct {
	// Width is the terminal column budget. Used to pick the input rule and
	// minimal-chrome thresholds; never used to truncate caller-rendered
	// section text.
	Width int

	// Conversation is the pre-rendered scrollback/output region (history
	// tail + optional draft + optional last-error block).
	Conversation string

	// Spinner is the optional spinner/hint line shown above the status bar
	// while a turn is in flight or while a TUI hint is forced. Empty means
	// the row is dropped.
	Spinner string

	// StatusBar is the single-line Hermes-compatible footer rendered by
	// RenderHermesStatusBar.
	StatusBar string

	// Prompt is the pre-rendered prompt symbol + input area block.
	Prompt string

	// VoiceStatus, ImageBar, and Completions are optional rows rendered
	// below the prompt. Empty strings are dropped.
	VoiceStatus string
	ImageBar    string
	Completions string
}

// RenderHermesChrome assembles the bottom-pinned chrome stack used by
// Hermes' prompt_toolkit Application(full_screen=False) frontend. Layout
// order matches cli.py:_build_tui_layout_children:
//
//	conversation
//	(optional) spinner/hint
//	status bar
//	top input rule
//	prompt + input area
//	(optional) bottom input rule — dropped on minimal-chrome widths
//	(optional) voice status
//	(optional) image bar
//	(optional) completions menu
//
// All sections are caller-rendered strings; this helper only picks order,
// drops empty optional rows, and inserts the bronze input rule.
func RenderHermesChrome(in HermesChromeInput) string {
	width := in.Width
	if width < 1 {
		width = 1
	}
	minimal := DefaultHermesSkin().UseMinimalChrome(width)

	rule := hermesChromeInputRule(width)

	parts := make([]string, 0, 9)
	if in.Conversation != "" {
		parts = append(parts, in.Conversation)
	}
	if in.Spinner != "" {
		parts = append(parts, in.Spinner)
	}
	if in.StatusBar != "" {
		parts = append(parts, in.StatusBar)
	}
	parts = append(parts, rule)
	if in.Prompt != "" {
		parts = append(parts, in.Prompt)
	}
	if !minimal {
		parts = append(parts, rule)
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

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// HermesChromeUseAltScreen reports whether the bottom-pinned Hermes chrome
// should be rendered in the terminal alt-screen. Hermes runs prompt_toolkit
// with Application(full_screen=False), so the answer is always false: the
// TUI must live in normal scrollback and let history persist after exit.
func HermesChromeUseAltScreen() bool {
	return false
}

// HermesChromeAssistantLabel returns the response-region label rendered above
// assistant output — matching Hermes' " ⚕ Hermes " response box.
func HermesChromeAssistantLabel() string {
	return strings.TrimSpace(DefaultHermesSkin().ResponseLabel)
}

func hermesChromeInputRule(width int) string {
	if width < 1 {
		width = 1
	}
	return strings.Repeat("─", width)
}
