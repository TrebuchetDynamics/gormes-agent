package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tooltrace"
)

var (
	muted     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
	botStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// View renders the bottom-pinned Hermes-compatible chrome. Layout matches
// cli.py:_build_tui_layout_children: scrollback/conversation, optional
// spinner/hint row, single-line status footer, bronze input rule, prompt +
// input area, optional bottom rule (dropped on minimal widths), and reserved
// voice/image/completion slots. The TUI runs in the alternate screen like
// current Hermes Ink so repeated full-screen render ticks do not smear stale
// frame fragments into normal scrollback.
//
// View never panics on any non-negative (width, height) input.
func (m Model) View() string {
	if m.width < 20 || m.height < 10 {
		return "terminal too narrow — resize to at least 20×10"
	}

	editor := m.editor
	editorHeight := promptHeightForValue(editor.Value())
	editor.SetHeight(editorHeight)
	if editorHeight < 1 {
		editorHeight = 1
	}
	editorBlockH := editorHeight
	chromeOverhead := 1                                   // status rule
	convH := m.height - editorBlockH - chromeOverhead - 1 // -1 for spinner/hint reserve
	if convH < 3 {
		convH = 3
	}

	convW := m.width
	if convW < 4 {
		convW = 4
	}

	conv := conversationViewportTail(m.frame, convW, convH)

	editorW := m.width
	if editorW < 10 {
		editorW = 10
	}
	editor.SetWidth(editorW)
	prompt := editor.View()

	statusBar := RenderHermesStatusBar(hermesStatusModelFromFrame(m.frame), m.width)

	hint := renderHermesHint(m.frame, m.mouseStatus(), m.statusMessage)

	// Render the active modal panel if one is present.
	panel := m.RenderActivePanel(m.width, m.height)

	return RenderHermesChrome(HermesChromeInput{
		Width:        m.width,
		Conversation: conv,
		Spinner:      hint,
		Panel:        panel,
		StatusBar:    statusBar,
		Prompt:       prompt,
	})
}

func renderHermesHint(f kernel.RenderFrame, mouseStatus, statusMessage string) string {
	var parts []string
	if f.Phase != kernel.PhaseIdle && f.Phase != kernel.PhaseFailed {
		parts = append(parts, strings.ToLower(f.Phase.String()))
		if f.SessionID != "" {
			parts = append(parts, "session "+shortSessionID(f.SessionID))
		}
	}
	if mouseStatus == "mouse: disabled" {
		parts = append(parts, mouseStatus)
	}
	if statusMessage != "" {
		parts = append(parts, statusMessage)
	}
	if len(parts) == 0 {
		return ""
	}
	return muted.Render(strings.Join(parts, " · "))
}

func promptHeightForValue(value string) int {
	if value == "" {
		return 1
	}
	lines := strings.Count(value, "\n") + 1
	if lines < 1 {
		return 1
	}
	if lines > 4 {
		return 4
	}
	return lines
}

// hermesStatusModelFromFrame projects the kernel render frame onto the data
// shape expected by RenderHermesStatusBar.
func hermesStatusModelFromFrame(f kernel.RenderFrame) HermesStatusModel {
	out := HermesStatusModel{
		StatusLabel:     hermesStatusLabelFromPhase(f.Phase),
		ModelName:       f.Model,
		ReasoningEffort: string(f.ReasoningEffort.Requested),
		CWDLabel:        hermesWorkingDirLabel(),
	}
	if f.ContextStatus != nil {
		out.ContextTokens = f.ContextStatus.LastTotalTokens
		out.ContextLength = f.ContextStatus.ContextLength
	}
	return out
}

func hermesStatusLabelFromPhase(phase kernel.Phase) string {
	switch phase {
	case kernel.PhaseIdle:
		return "ready"
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseFinalizing:
		return "running…"
	case kernel.PhaseCancelling:
		return "cancelling…"
	case kernel.PhaseReconnecting:
		return "reconnecting…"
	case kernel.PhaseFailed:
		return "error"
	default:
		return strings.ToLower(phase.String())
	}
}

func hermesWorkingDirLabel() string {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" && (cwd == home || strings.HasPrefix(cwd, home+"/")) {
		cwd = "~" + strings.TrimPrefix(cwd, home)
	}
	const max = 40
	if len([]rune(cwd)) <= max {
		return cwd
	}
	r := []rune(cwd)
	return "…" + string(r[len(r)-(max-1):])
}

// renderConv is the legacy entry point retained for tests that pre-date the
// bottom-pinned chrome. It delegates to the same viewport tail helper View
// uses and keeps the (width, height) call shape so existing fixtures stay
// addressable.
func renderConv(f kernel.RenderFrame, width, height int) string {
	return conversationViewportTail(f, width, height)
}

func conversationViewportTail(f kernel.RenderFrame, width, height int) string {
	if width < 4 {
		width = 4
	}
	if height < 1 {
		height = 1
	}
	wrap := lipgloss.NewStyle().Width(width - 4)
	compact := width < 8 || height < 3
	forced := conversationForcedBlocks(f, wrap, compact)
	maxLines := height + 1 + len(forced)

	var visible []string
	for i := len(f.History) - 1; i >= 0; i-- {
		msg := f.History[i]
		block := conversationMessageBlock(msg, wrap, compact)
		omitted := i
		candidate := append([]string{block}, visible...)
		if omitted > 0 {
			candidate = append([]string{omittedHistorySentinel(omitted)}, candidate...)
		}
		candidate = append(candidate, forced...)
		if renderedLineCount(strings.Join(candidate, "\n\n")) > maxLines && len(visible) > 0 {
			break
		}
		visible = append([]string{block}, visible...)
	}

	omitted := len(f.History) - len(visible)
	lines := make([]string, 0, len(visible)+1)
	if omitted > 0 {
		lines = append(lines, omittedHistorySentinel(omitted))
	}
	lines = append(lines, visible...)
	lines = append(lines, forced...)
	if len(lines) == 0 {
		return muted.Render("(start typing below to begin)")
	}
	return strings.Join(lines, "\n\n")
}

func conversationForcedBlocks(f kernel.RenderFrame, wrap lipgloss.Style, compact bool) []string {
	var blocks []string
	if !frameHasFinalAssistant(f) {
		if progress := conversationToolProgressBlock(f, compact); progress != "" {
			blocks = append(blocks, progress)
		}
	}
	if f.DraftText != "" && !draftDuplicatesFinalAssistant(f) {
		blocks = append(blocks, conversationDraftBlock(f.DraftText, wrap, compact))
	}
	if f.LastError != "" {
		blocks = append(blocks, conversationErrorBlock(f.LastError, compact))
	}
	return blocks
}

func frameHasFinalAssistant(f kernel.RenderFrame) bool {
	if f.Phase != kernel.PhaseIdle {
		return false
	}
	return strings.TrimSpace(lastAssistantContent(f.History)) != ""
}

func draftDuplicatesFinalAssistant(f kernel.RenderFrame) bool {
	draft := strings.TrimSpace(f.DraftText)
	if draft == "" || f.Phase != kernel.PhaseIdle {
		return false
	}
	return draft == strings.TrimSpace(lastAssistantContent(f.History))
}

func lastAssistantContent(history []hermes.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			return history[i].Content
		}
	}
	return ""
}

func conversationToolProgressBlock(f kernel.RenderFrame, compact bool) string {
	texts := make([]string, 0, len(f.SoulEvents))
	for _, event := range f.SoulEvents {
		texts = append(texts, event.Text)
	}
	progress := tooltrace.FormatBlock(texts)
	if progress == "" {
		return ""
	}
	if compact {
		return compactViewportText(progress)
	}
	return muted.Render(progress)
}

func conversationMessageBlock(msg hermes.Message, wrap lipgloss.Style, compact bool) string {
	if msg.Role == "tool" {
		return conversationToolResultBlock(msg, wrap, compact)
	}
	content := msg.Content
	if compact {
		content = compactViewportText(content)
	} else {
		content = wrap.Render(content)
	}
	return roleTag(msg.Role) + " " + content
}

func conversationToolResultBlock(msg hermes.Message, wrap lipgloss.Style, compact bool) string {
	name := strings.TrimSpace(msg.Name)
	label := "tool result"
	if name != "" {
		label += ": " + name
	}
	content := strings.TrimSpace(msg.Content)
	if compact {
		return muted.Render(label) + " " + compactViewportText(content)
	}
	content = wrap.Render(content)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = "│ " + line
	}
	return muted.Render("╭─ " + label + "\n" + strings.Join(lines, "\n") + "\n╰─")
}

func conversationDraftBlock(draft string, wrap lipgloss.Style, compact bool) string {
	if compact {
		draft = compactViewportText(draft)
	} else {
		draft = wrap.Render(draft)
	}
	return botStyle.Render(HermesChromeAssistantLabel()) + " " + draft
}

func conversationErrorBlock(lastError string, compact bool) string {
	if compact {
		lastError = compactViewportText(lastError)
	}
	return errStyle.Render("err:") + " " + lastError
}

func compactViewportText(s string) string {
	return truncateEllipsis(strings.Join(strings.Fields(s), " "), 48)
}

func omittedHistorySentinel(count int) string {
	return muted.Render(fmt.Sprintf("... %d earlier history messages omitted ...", count))
}

func renderedLineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func roleTag(role string) string {
	switch role {
	case "user":
		return userStyle.Render("you:")
	case "assistant":
		return botStyle.Render(HermesChromeAssistantLabel())
	case "system":
		return muted.Render("sys:")
	}
	return muted.Render(role + ":")
}

func truncateEllipsis(s string, n int) string {
	if n <= 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func shortSessionID(id string) string {
	if id == "" {
		return "new"
	}
	if len(id) <= 8 {
		return id
	}
	return id[:8] + "…"
}

func statusSuffix(message string) string {
	if message == "" {
		return ""
	}
	return " · " + message
}
