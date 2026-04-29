package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

var (
	border    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	muted     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
	botStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// View renders the bottom-pinned Hermes-compatible chrome. Layout matches
// cli.py:_build_tui_layout_children: scrollback/conversation, optional
// spinner/hint row, single-line status footer, bronze input rule, prompt +
// input area, optional bottom rule (dropped on minimal widths), and reserved
// voice/image/completion slots. The TUI runs in normal scrollback rather
// than alt-screen so output persists after exit, matching Hermes.
//
// View never panics on any non-negative (width, height) input.
func (m Model) View() string {
	if m.width < 20 || m.height < 10 {
		return "terminal too narrow — resize to at least 20×10"
	}

	editorHeight := m.editor.Height()
	if editorHeight < 1 {
		editorHeight = 1
	}
	editorBlockH := editorHeight + 2 // border adds 2 rows
	chromeOverhead := 1 + 1 + 1      // status bar + top rule + bottom rule
	if DefaultHermesSkin().UseMinimalChrome(m.width) {
		chromeOverhead = 1 + 1 // status bar + top rule (bottom rule dropped)
	}
	convH := m.height - editorBlockH - chromeOverhead - 1 // -1 for spinner/hint reserve
	if convH < 3 {
		convH = 3
	}

	convW := m.width
	if convW < 4 {
		convW = 4
	}

	conv := conversationViewportTail(m.frame, convW, convH)

	editorW := m.width - 2
	if editorW < 10 {
		editorW = 10
	}
	prompt := border.Width(editorW).Render(m.editor.View())

	statusBar := RenderHermesStatusBar(hermesStatusModelFromFrame(m.frame), m.width)

	hint := muted.Render(fmt.Sprintf(
		"phase: %s · session: %s · %s%s",
		m.frame.Phase, shortSessionID(m.frame.SessionID), m.mouseStatus(), statusSuffix(m.statusMessage),
	))

	return RenderHermesChrome(HermesChromeInput{
		Width:        m.width,
		Conversation: conv,
		Spinner:      hint,
		StatusBar:    statusBar,
		Prompt:       prompt,
	})
}

// hermesStatusModelFromFrame projects the kernel render frame onto the data
// shape expected by RenderHermesStatusBar.
func hermesStatusModelFromFrame(f kernel.RenderFrame) HermesStatusModel {
	out := HermesStatusModel{
		ModelName: f.Model,
	}
	if f.ContextStatus != nil {
		out.ContextTokens = f.ContextStatus.LastTotalTokens
		out.ContextLength = f.ContextStatus.ContextLength
	}
	return out
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
	if f.DraftText != "" {
		blocks = append(blocks, conversationDraftBlock(f.DraftText, wrap, compact))
	}
	if f.LastError != "" {
		blocks = append(blocks, conversationErrorBlock(f.LastError, compact))
	}
	return blocks
}

func conversationMessageBlock(msg hermes.Message, wrap lipgloss.Style, compact bool) string {
	content := msg.Content
	if compact {
		content = compactViewportText(content)
	} else {
		content = wrap.Render(content)
	}
	return roleTag(msg.Role) + " " + content
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
