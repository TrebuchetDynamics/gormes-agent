// Package tui — Hermes-compatible modal/tool-progress panel renderers.
//
// These renderers are pure functions (state in → string/[]string out). They
// own no goroutines, never read time.Now(), and never store secret bytes.
// They are the Go-native ports of upstream Hermes prompt_toolkit fragments:
//
//   - RenderToolSpinner       ↔ cli.py:_render_spinner_text
//   - RenderToolScrollback    ↔ cli.py:_on_tool_progress (tool.completed)
//   - RenderApprovalPanel     ↔ cli.py:_get_approval_display_fragments
//   - RenderClarifyPanel      ↔ cli.py:_get_clarify_display
//   - RenderSecretPanel       ↔ cli.py:_sudo_password_callback / _get_secret_display
//
// Live tool/approval/clarify wiring is intentionally out of scope; this row
// only ports renderer contracts so later live work cannot invent a separate
// TUI language.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// =====================================================================
// Tool spinner
// =====================================================================

// ToolSpinnerState is the state injected by the kernel/tool layer when a
// tool.started event is observed. The renderer is pure — Elapsed must be
// supplied by the caller (typically via a poll tick); RenderToolSpinner
// must never read time.Now().
type ToolSpinnerState struct {
	ToolName string
	Preview  string
	Elapsed  time.Duration
}

// RenderToolSpinner returns the live spinner/status text for a running tool.
// Mirrors cli.py:_render_spinner_text — when Elapsed > 0 the text is
// suffixed with " (Xm Ys)" for >= 60s and " (Xs)" with a single decimal for
// shorter runs; when Elapsed == 0 only the body is returned (matches the
// Python "(elapsed_str)" omission when t0 == 0).
func RenderToolSpinner(s ToolSpinnerState) string {
	body := strings.TrimSpace(s.Preview)
	if body == "" {
		body = strings.TrimSpace(s.ToolName)
	}
	if body == "" {
		return ""
	}
	if s.Elapsed <= 0 {
		return "  " + body
	}
	return "  " + body + "  (" + formatToolElapsed(s.Elapsed) + ")"
}

// formatToolElapsed mirrors the Hermes elapsed_str branches:
//   - >= 60s   → "{m}m {s}s"
//   - <  60s   → "{x.y}s" (one decimal place)
func formatToolElapsed(d time.Duration) string {
	if d >= time.Minute {
		mins := int(d / time.Minute)
		secs := int((d % time.Minute) / time.Second)
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	secs := float64(d) / float64(time.Second)
	return fmt.Sprintf("%.1fs", secs)
}

// =====================================================================
// Tool scrollback
// =====================================================================

// ToolScrollMode selects whether scrollback prints every completion or
// suppresses consecutive duplicates. Mirrors Hermes' tool_progress_mode
// values "all" and "new".
type ToolScrollMode int

const (
	// ToolScrollAll prints every tool.completed line in order.
	ToolScrollAll ToolScrollMode = iota
	// ToolScrollNew suppresses a tool.completed line whose ToolName equals
	// the previous rendered ToolName, matching cli.py's
	// `_last_scrollback_tool` dedup.
	ToolScrollNew
)

// ToolCompletion is one tool.completed event ready for scrollback rendering.
type ToolCompletion struct {
	ToolName string
	Detail   string
	Error    bool
}

// RenderToolScrollback renders stacked tool-completion lines. Errors append
// " [error]" to match upstream. In ToolScrollNew mode, a completion whose
// ToolName matches the previously rendered ToolName is suppressed.
func RenderToolScrollback(items []ToolCompletion, mode ToolScrollMode) []string {
	out := make([]string, 0, len(items))
	lastTool := ""
	for _, item := range items {
		if mode == ToolScrollNew && item.ToolName == lastTool {
			continue
		}
		line := "  " + item.Detail
		if strings.TrimSpace(item.Detail) == "" {
			line = "  " + item.ToolName
		}
		if item.Error {
			line = line + " [error]"
		}
		out = append(out, line)
		lastTool = item.ToolName
	}
	return out
}

// =====================================================================
// Approval panel
// =====================================================================

// ApprovalChoice enumerates the dangerous-command approval choices Hermes
// exposes. Order matters: the renderer numbers them 1..N in the order they
// appear in ApprovalPanelState.Choices.
type ApprovalChoice int

const (
	// ApprovalOnce — allow this single invocation.
	ApprovalOnce ApprovalChoice = iota
	// ApprovalSession — allow for the rest of this CLI session.
	ApprovalSession
	// ApprovalAlways — add to the persistent allowlist.
	ApprovalAlways
	// ApprovalDeny — deny and tell the agent.
	ApprovalDeny
	// ApprovalView — expand a long command in place. When the user picks
	// this choice the caller must set ViewExpanded=true and remove
	// ApprovalView from the Choices slice before re-rendering.
	ApprovalView
)

// approvalLabel returns the upstream Hermes label for an approval choice.
func approvalLabel(c ApprovalChoice) string {
	switch c {
	case ApprovalOnce:
		return "Allow once"
	case ApprovalSession:
		return "Allow for this session"
	case ApprovalAlways:
		return "Add to permanent allowlist"
	case ApprovalDeny:
		return "Deny"
	case ApprovalView:
		return "Show full command"
	default:
		return "?"
	}
}

// ApprovalPanelState is everything the renderer needs to draw the
// dangerous-command approval modal.
type ApprovalPanelState struct {
	Description, Command string
	Choices              []ApprovalChoice
	Selected             ApprovalChoice
	// ViewExpanded reflects whether the user already chose ApprovalView;
	// when true the full command renders verbatim and the "Show full
	// command" choice should be absent from Choices.
	ViewExpanded bool
	Width        int
	Height       int
}

// RenderApprovalPanel mirrors cli.py:_get_approval_display_fragments. It
// always keeps the title + command + every choice on-screen; long
// descriptions truncate. When ViewExpanded is true the command renders
// verbatim with no "..." suffix.
func RenderApprovalPanel(s ApprovalPanelState) string {
	return renderApprovalPanel(s, SkinStyles{}, false)
}

func RenderApprovalPanelWithSkin(s ApprovalPanelState, skin HermesSkin) string {
	return renderApprovalPanel(s, SkinStylesFor(skin), true)
}

func renderApprovalPanel(s ApprovalPanelState, styles SkinStyles, styled bool) string {
	title := renderSkinStyle(styled, styles.Critical, "⚠️  Dangerous Command")
	cmd := s.Command
	if !s.ViewExpanded && len(cmd) > 70 {
		cmd = cmd[:70] + "..."
	}

	lines := []string{title, ""}
	lines = append(lines, renderSkinStyle(styled, styles.Bad, cmd), "")
	// When ViewExpanded is true the "Show full command" choice has done
	// its job; mirror cli.py:_handle_approval_selection which drops it
	// from state["choices"] before re-rendering.
	visible := make([]ApprovalChoice, 0, len(s.Choices))
	for _, c := range s.Choices {
		if s.ViewExpanded && c == ApprovalView {
			continue
		}
		visible = append(visible, c)
	}
	for i, choice := range visible {
		num := numberedPrefix(i)
		marker := "  "
		choiceStyle := styles.Normal
		if choice == s.Selected {
			marker = "❯ "
			choiceStyle = styles.Selected
		}
		line := marker + num + ". " + approvalLabel(choice)
		lines = append(lines, renderSkinStyle(styled, choiceStyle, line))
	}
	if strings.TrimSpace(s.Description) != "" {
		desc := s.Description
		// Soft cap so a multi-paragraph tirith finding cannot push the
		// command/choices off-screen. The renderer is pure, so the cap
		// is height-driven only when Height is supplied.
		if s.Height > 0 {
			budget := s.Height - len(lines) - 2 // border + spacer
			if budget < 1 {
				budget = 1
			}
			desc = clampLines(desc, budget)
		}
		lines = append(lines, "")
		lines = append(lines, renderSkinStyle(styled, styles.Dim, desc))
	}
	return boxifyWithStyles(lines, styles, styled)
}

// =====================================================================
// Clarify panel
// =====================================================================

// ClarifyPanelState is everything the renderer needs to draw the clarify
// modal: a question, a list of choices (Other is appended automatically),
// the currently highlighted index, and an optional timeout hint.
type ClarifyPanelState struct {
	Question      string
	Choices       []string
	Selected      int
	TimeoutHint   string
	Width, Height int
}

// RenderClarifyPanel mirrors cli.py:_get_clarify_display. It numbers every
// choice 1..N, marks the selected one with "❯", appends an "Other" option
// numbered N+1, and surfaces the timeout hint inline at the bottom.
func RenderClarifyPanel(s ClarifyPanelState) string {
	return renderClarifyPanel(s, SkinStyles{}, false)
}

func RenderClarifyPanelWithSkin(s ClarifyPanelState, skin HermesSkin) string {
	return renderClarifyPanel(s, SkinStylesFor(skin), true)
}

func renderClarifyPanel(s ClarifyPanelState, styles SkinStyles, styled bool) string {
	title := renderSkinStyle(styled, styles.Title, "Hermes needs your input")
	lines := []string{title, "", renderSkinStyle(styled, styles.Text, s.Question), ""}

	for i, choice := range s.Choices {
		num := numberedPrefix(i)
		marker := "  "
		choiceStyle := styles.Normal
		if i == s.Selected {
			marker = "❯ "
			choiceStyle = styles.Selected
		}
		lines = append(lines, renderSkinStyle(styled, choiceStyle, marker+num+". "+choice))
	}

	otherIdx := len(s.Choices)
	otherNum := numberedPrefix(otherIdx)
	otherMarker := "  "
	otherStyle := styles.Normal
	if s.Selected == otherIdx {
		otherMarker = "❯ "
		otherStyle = styles.Selected
	}
	lines = append(lines, renderSkinStyle(styled, otherStyle, otherMarker+otherNum+". Other (type your answer)"))

	if strings.TrimSpace(s.TimeoutHint) != "" {
		lines = append(lines, "", renderSkinStyle(styled, styles.Dim, "  ↑/↓ to select, Enter to confirm  "+s.TimeoutHint))
	}

	if s.Height > 0 {
		// Trim from the question if the panel would not fit, keeping
		// every choice + Other + hint on-screen.
		budget := s.Height - 2 // borders
		if len(lines) > budget && budget > 0 {
			// Drop description-style padding lines first, never the
			// numbered choices.
			lines = compactClarifyLines(lines, budget)
		}
	}

	return boxifyWithStyles(lines, styles, styled)
}

// compactClarifyLines drops blank padding rows first so the numbered choices
// always survive the budget. If even after dropping blanks the budget is
// exceeded, the question is truncated with an ellipsis line.
func compactClarifyLines(lines []string, budget int) []string {
	if budget <= 0 || len(lines) <= budget {
		return lines
	}
	pruned := make([]string, 0, len(lines))
	for _, l := range lines {
		if len(pruned) >= budget && strings.TrimSpace(l) == "" {
			continue
		}
		pruned = append(pruned, l)
	}
	if len(pruned) > budget {
		// Truncate the question line (index 2 after title + blank); fall
		// back to head-trimming to preserve the choices/hint at the tail.
		excess := len(pruned) - budget
		if excess > 0 && len(pruned) > 3 {
			pruned = append(pruned[:2], pruned[2+excess:]...)
			pruned = append([]string{pruned[0], pruned[1], "… (question truncated)"}, pruned[2:]...)
		}
	}
	return pruned
}

// =====================================================================
// Secret / sudo panel
// =====================================================================

// SecretPanelMode distinguishes the sudo password prompt from the generic
// skill-secret capture prompt. The visual chrome is similar; the title and
// hint text differ.
type SecretPanelMode int

const (
	// SecretPanelSudo is the sudo password capture modal.
	SecretPanelSudo SecretPanelMode = iota
	// SecretPanelArbitrary is the skill/secret capture modal.
	SecretPanelArbitrary
)

// SecretPanelState carries everything the renderer needs to draw a hidden-
// input modal. SecretLen is the length of the user's typed buffer; the
// renderer never receives or echoes the actual bytes.
type SecretPanelState struct {
	Mode       SecretPanelMode
	PromptText string
	// Countdown is the remaining timeout. 0 means no countdown is shown.
	Countdown time.Duration
	// SecretLen is the length (in runes) of the typed buffer so the
	// renderer can show a mask hint of the correct width if it chooses.
	// The actual secret bytes never enter the panel state.
	SecretLen int
	Hint      string
}

// RenderSecretPanel mirrors cli.py:_get_secret_display and the sudo hint
// line. The renderer prints the title, prompt, hint, and (optionally) a
// countdown like "(45s)". It NEVER echoes secret bytes — by construction it
// cannot, since SecretPanelState only carries the length.
func RenderSecretPanel(s SecretPanelState) string {
	return renderSecretPanel(s, SkinStyles{}, false)
}

func RenderSecretPanelWithSkin(s SecretPanelState, skin HermesSkin) string {
	return renderSecretPanel(s, SkinStylesFor(skin), true)
}

func renderSecretPanel(s SecretPanelState, styles SkinStyles, styled bool) string {
	title := "🔑 Skill Setup Required"
	titleStyle := styles.Title
	if s.Mode == SecretPanelSudo {
		title = "🔒 Sudo Password Required"
		titleStyle = styles.Critical
	}
	prompt := strings.TrimSpace(s.PromptText)
	if prompt == "" {
		prompt = "Enter secret"
	}
	lines := []string{renderSkinStyle(styled, titleStyle, title), "", renderSkinStyle(styled, styles.Prompt, prompt)}
	if strings.TrimSpace(s.Hint) != "" {
		lines = append(lines, "", renderSkinStyle(styled, styles.Dim, s.Hint))
	}
	if s.Countdown > 0 {
		secs := int(s.Countdown / time.Second)
		lines = append(lines, renderSkinStyle(styled, styles.Warn, fmt.Sprintf("  (%ds)", secs)))
	}
	return boxifyWithStyles(lines, styles, styled)
}

// =====================================================================
// Shared panel chrome helpers
// =====================================================================

// numberedPrefix returns Hermes' 1..9, then 0, then space prefix used for
// quick-selection of choices in the approval/clarify panels. Mirrors the
// num_prefix branches in cli.py.
func numberedPrefix(i int) string {
	switch {
	case i < 9:
		return string(rune('1' + i))
	case i == 9:
		return "0"
	default:
		return " "
	}
}

// boxify wraps body lines in a Unicode rounded box, using each line's own
// length to size the box. Empty lines are preserved as visual gaps.
func boxifyWithStyles(body []string, styles SkinStyles, styled bool) string {
	width := 0
	for _, line := range body {
		if w := visibleWidth(line); w > width {
			width = w
		}
	}
	if width < 30 {
		width = 30
	}

	var b strings.Builder
	b.WriteString(renderSkinStyle(styled, styles.Separator, "╭"+strings.Repeat("─", width+2)+"╮"))
	b.WriteString("\n")
	for _, line := range body {
		pad := width - visibleWidth(line)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(renderSkinStyle(styled, styles.Separator, "│ "))
		b.WriteString(line)
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(renderSkinStyle(styled, styles.Separator, " │"))
		b.WriteString("\n")
	}
	b.WriteString(renderSkinStyle(styled, styles.Separator, "╰"+strings.Repeat("─", width+2)+"╯"))
	b.WriteString("\n")
	return b.String()
}

func visibleWidth(s string) int {
	return lipgloss.Width(s)
}

// clampLines truncates a multi-line string body to at most n rows, appending
// a "…" marker when content was dropped. Used by RenderApprovalPanel to keep
// long descriptions from pushing the choice list off-screen.
func clampLines(s string, n int) string {
	if n <= 0 {
		return ""
	}
	rows := strings.Split(s, "\n")
	if len(rows) <= n {
		return s
	}
	keep := n - 1
	if keep < 1 {
		keep = 1
	}
	return strings.Join(rows[:keep], "\n") + "\n… (description truncated)"
}
