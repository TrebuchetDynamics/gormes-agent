package tui

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tooltrace"
)

// Transcript chrome renders through the Gormes-owned semantic style system
// (styles.go), resolved from the active HermesSkin so every built-in skin
// re-themes the chat surface. No role/chrome color is hardcoded here.
var (
	chatChrome     = defaultChatStyles()
	muted          = chatChrome.Assistant // generic dim: hints, sentinels, assistant body
	userStyle      = chatChrome.User
	errStyle       = chatChrome.Error
	separatorStyle = chatChrome.Separator
	toolOutStyle   = chatChrome.ToolOutput
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
		return fmt.Sprintf(
			"terminal too small: %d×%d — resize to at least 20×10 for the Bubble Tea chat UI",
			m.width,
			m.height,
		)
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

	hint := renderHermesHint(m.frame, m.statusMessage)
	completions := renderSlashCompletionMenu(editor.Value(), m.width)

	// Render the active modal panel if one is present.
	panel := m.RenderActivePanel(m.width, m.height)
	if m.modelPicker != nil {
		picker := *m.modelPicker
		picker.Width = m.width
		picker.Height = m.height
		panel = RenderModelPicker(picker)
	}

	// Render todo panel if there are active tasks for the current session.
	todoPanel := m.renderTodoPanel(convW)

	return RenderHermesChrome(HermesChromeInput{
		Width:        m.width,
		Conversation: conv,
		Spinner:      hint,
		Panel:        panel,
		TodoPanel:    todoPanel,
		StatusBar:    statusBar,
		Prompt:       prompt,
		Completions:  completions,
	})
}

func renderHermesHint(f kernel.RenderFrame, statusMessage string) string {
	var parts []string
	if f.Phase != kernel.PhaseIdle && f.Phase != kernel.PhaseFailed {
		parts = append(parts, strings.ToLower(f.Phase.String()))
		if f.SessionID != "" {
			parts = append(parts, "session "+shortSessionID(f.SessionID))
		}
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
	wrapWidth := width - 4
	compact := width < 8 || height < 3
	forced := conversationForcedBlocks(f, wrapWidth, compact)
	maxLines := height + 1 + len(forced)

	var visible []string
	for i := len(f.History) - 1; i >= 0; i-- {
		block := conversationMessageBlockAt(f.History, i, wrapWidth, compact)
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
		return conversationEmptyIntro(f, width, compact)
	}
	return strings.Join(lines, "\n\n")
}

func conversationForcedBlocks(f kernel.RenderFrame, wrapWidth int, compact bool) []string {
	var blocks []string
	hasFinal := frameHasFinalAssistant(f)
	if !hasFinal {
		if progress := conversationToolProgressBlock(f, compact); progress != "" {
			blocks = append(blocks, progress)
		}
	}
	if f.DraftText != "" && !draftDuplicatesFinalAssistant(f) {
		blocks = append(blocks, conversationDraftBlock(f.DraftText, wrapWidth, compact))
	}
	if f.LastError != "" {
		blocks = append(blocks, conversationErrorBlock(f.LastError, wrapWidth, compact))
	}
	// R3 streaming feedback: when a turn is active but nothing concrete has
	// surfaced yet (no tool trace, draft, or error), show the reused
	// thinking indicator so the user is never left wondering. Suppressed the
	// moment any real signal exists so it never disturbs transcript order.
	if len(blocks) == 0 && !hasFinal && turnIsActive(f.Phase) {
		if think := conversationThinkingBlock(compact); think != "" {
			blocks = append(blocks, think)
		}
	}
	return blocks
}

func turnIsActive(p kernel.Phase) bool {
	switch p {
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseFinalizing, kernel.PhaseReconnecting:
		return true
	default:
		return false
	}
}

// conversationThinkingBlock reuses thinking.go's RenderThinking (it is not
// reimplemented here) to render the live "reasoning" indicator.
func conversationThinkingBlock(compact bool) string {
	t := RenderThinking(ThinkingState{Visible: true})
	if t == "" {
		return ""
	}
	if compact {
		return compactViewportText(t)
	}
	return muted.Render(t)
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
	return toolOutStyle.Render(progress)
}

func conversationMessageBlock(msg hermes.Message, wrapWidth int, compact bool) string {
	if msg.Role == "tool" {
		return conversationToolResultBlock(msg, wrapWidth, compact)
	}
	content := msg.Content
	if compact {
		content = compactViewportText(content)
	} else {
		content = RenderMarkdownSoftWrapTrim(content, wrapWidth)
	}
	return transcriptRow(msg.Role, content)
}

func conversationMessageBlockAt(history []hermes.Message, idx, wrapWidth int, compact bool) string {
	block := conversationMessageBlock(history[idx], wrapWidth, compact)
	if !compact && conversationNeedsTurnSeparator(history, idx) {
		return separatorStyle.Render("───") + "\n\n" + block
	}
	return block
}

func conversationNeedsTurnSeparator(history []hermes.Message, idx int) bool {
	if idx < 0 || idx >= len(history) || history[idx].Role != "user" {
		return false
	}
	for i := 0; i < idx; i++ {
		if history[i].Role == "user" {
			return true
		}
	}
	return false
}

func conversationToolResultBlock(msg hermes.Message, wrapWidth int, compact bool) string {
	name := strings.TrimSpace(msg.Name)
	if name == "" {
		name = "tool"
	}
	content := strings.TrimSpace(msg.Content)
	if compact {
		return toolOutStyle.Render("⚡ " + name + " " + compactViewportText(content))
	}
	content = RenderMarkdownSoftWrapTrim(content, wrapWidth)
	lines := strings.Split(content, "\n")
	// R3: collapse long tool output to a head plus a summary (ccx-go
	// RenderToolOutputInline pattern) so a verbose tool result never floods
	// the transcript. Short output (fidelity small-content) stays intact.
	const collapseOver, collapseHead = 5, 3
	if len(lines) > collapseOver {
		hidden := len(lines) - collapseHead
		kept := append([]string{}, lines[:collapseHead]...)
		kept = append(kept, fmt.Sprintf("[+%d more lines]", hidden))
		lines = kept
	}
	out := []string{"   ╭─ ⚡ " + name}
	for i, line := range lines {
		lines[i] = "   │ " + line
	}
	out = append(out, lines...)
	out = append(out, "   ╰─")
	return toolOutStyle.Render(strings.Join(out, "\n"))
}

func conversationDraftBlock(draft string, wrapWidth int, compact bool) string {
	if compact {
		draft = compactViewportText(draft)
	} else {
		draft = RenderMarkdownSoftWrapTrim(draft, wrapWidth)
	}
	return transcriptRow("assistant", draft)
}

func conversationErrorBlock(lastError string, wrapWidth int, compact bool) string {
	if compact {
		lastError = compactViewportText(lastError)
		return errStyle.Render("err:") + " " + lastError
	}
	lastError = RenderMarkdownSoftWrapTrim(strings.Join(strings.Fields(lastError), " "), errorBodyWidth(wrapWidth))
	lines := strings.Split(lastError, "\n")
	prefix := errStyle.Render("err:") + " "
	continuation := strings.Repeat(" ", lipgloss.Width("err:")+1)
	for i, line := range lines {
		if i == 0 {
			lines[i] = prefix + line
			continue
		}
		lines[i] = continuation + line
	}
	return strings.Join(lines, "\n")
}

func errorBodyWidth(wrapWidth int) int {
	width := wrapWidth - lipgloss.Width("err:") - 1
	if width < 8 {
		return 8
	}
	return width
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

func conversationEmptyIntro(f kernel.RenderFrame, width int, compact bool) string {
	if compact {
		return muted.Render("⚕ Gormes · /help for commands")
	}
	ctx := welcomeContext{
		Model:     f.Model,
		Provider:  f.ProviderStatus.Provider,
		Runtime:   f.ProviderStatus.Runtime,
		CWD:       hermesWorkingDirLabel(),
		SessionID: f.SessionID,
		Version:   buildInfoVersion(),
	}
	return welcomePanel(DefaultHermesSkin(), ctx, width)
}

// buildInfoVersion returns the operator-facing module version when the binary
// carries one, or "" for source/dev builds. internal/tui cannot import the
// cmd/gormes main.Version symbol; the seeded value is wired later by the
// "Gormes welcome panel version/tool-count wiring" follow-up row.
func buildInfoVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	v := strings.TrimSpace(info.Main.Version)
	if v == "" || v == "(devel)" || v == "unknown" {
		return ""
	}
	return v
}

func transcriptRow(role, content string) string {
	glyph := transcriptGlyph(role)
	style := transcriptGlyphStyle(role)
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return style.Render(glyph)
	}

	prefix := style.Render(glyph) + " "
	continuation := strings.Repeat(" ", lipgloss.Width(glyph)+1)
	for i, line := range lines {
		if i == 0 {
			lines[i] = prefix + line
			continue
		}
		lines[i] = continuation + line
	}
	return strings.Join(lines, "\n")
}

func transcriptGlyph(role string) string {
	skin := DefaultHermesSkin()
	switch role {
	case "user":
		prompt := strings.TrimSpace(skin.PromptSymbol)
		if prompt == "" {
			return "❯"
		}
		return prompt
	case "assistant":
		toolPrefix := strings.TrimSpace(skin.ToolPrefix)
		if toolPrefix == "" {
			return "┊"
		}
		return toolPrefix
	case "system":
		return "·"
	}
	role = strings.TrimSpace(role)
	if role == "" {
		return "·"
	}
	return role
}

func transcriptGlyphStyle(role string) lipgloss.Style {
	switch role {
	case "user":
		return userStyle
	case "assistant":
		return muted
	default:
		return muted
	}
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

func (m Model) renderTodoPanel(width int) string {
	if m.todoReader == nil {
		return ""
	}
	items := m.todoReader(m.SessionID())
	if len(items) == 0 {
		return ""
	}
	return RenderTodoPanel(items, width)
}
