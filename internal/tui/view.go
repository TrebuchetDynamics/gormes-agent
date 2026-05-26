package tui

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/trace"
)

// Transcript chrome renders through the Gormes-owned semantic style system
// (styles.go), resolved from the active HermesSkin so every built-in skin
// re-themes the chat surface. No role/chrome color is hardcoded here.

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

	conv := conversationViewportTailWithSkinAndDetails(m.frame, convW, convH, m.compactTranscript, m.detailsState, m.currentSkin())

	editorW := m.width
	if editorW < 10 {
		editorW = 10
	}
	editor.SetWidth(editorW)
	prompt := m.renderComposerPrompt(editor)

	statusBar := ""
	statusBarMode := normalizeStatusBarMode(m.statusBarMode)
	if statusBarMode != StatusBarModeOff {
		statusBar = RenderHermesStatusBarWithSkin(hermesStatusModelFromFrame(m.frame), m.width, m.currentSkin())
		if footer, ok := m.renderExtensionFooter(m.width); ok {
			statusBar = footer
		} else {
			statusBar = m.renderExtensionStatusLine(statusBar, m.width)
		}
	}

	hint := m.renderHermesHint()
	completions := renderSlashCompletionMenuWithSkin(editor.Value(), m.width, m.currentSkin())

	// Render the active modal panel if one is present.
	panel := m.RenderActivePanel(m.width, m.height)
	if m.transientPage != nil && panel == "" {
		panel = RenderTransientPageWithSkin(*m.transientPage, m.width, convH, m.currentSkin())
	}
	if m.modelPicker != nil {
		picker := *m.modelPicker
		picker.Width = m.width
		picker.Height = m.height
		panel = RenderModelPickerWithSkin(picker, m.currentSkin())
	}

	// Render todo panel if there are active tasks for the current session.
	todoPanel := m.renderTodoPanel(convW)

	return RenderHermesChrome(HermesChromeInput{
		Width:                 m.width,
		Conversation:          conv,
		Spinner:               hint,
		Panel:                 panel,
		TodoPanel:             todoPanel,
		ExtensionWidgetsAbove: m.renderExtensionWidgets(kernel.ExtensionUIWidgetAboveEditor, m.width),
		QueuedMessages:        m.renderQueuedMessageWidgets(m.width),
		StatusBar:             statusBar,
		StatusBarMode:         statusBarMode,
		Prompt:                prompt,
		ExtensionWidgetsBelow: m.renderExtensionWidgets(kernel.ExtensionUIWidgetBelowEditor, m.width),
		Completions:           completions,
	})
}

func (m Model) renderComposerPrompt(editor textarea.Model) string {
	return renderComposerPromptWithFocus(editor, m.currentSkin(), m.composerChatFocused())
}

func renderComposerPromptWithFocus(editor textarea.Model, skin HermesSkin, focused bool) string {
	ApplyTextareaSkin(&editor, skin)
	if focused {
		_ = editor.Focus()
	} else {
		editor.Blur()
	}
	return editor.View()
}

func (m Model) composerChatFocused() bool {
	return !m.IsPanelActive() && m.transientPage == nil && m.modelPicker == nil
}

func (m Model) renderHermesHint() string {
	skin := m.currentSkin()
	if working, ok := m.extensionWorkingIndicator(); ok {
		return renderHermesHintWithExtensionWorking(m.frame, m.statusMessage, m.width, m.spinnerFrame, m.indicatorStyle, working, skin)
	}
	return renderHermesHintWithIndicatorForSkin(m.frame, m.statusMessage, m.width, m.spinnerFrame, m.indicatorStyle, skin)
}

func renderHermesHint(f kernel.RenderFrame, statusMessage string, width int, spinnerFrame int) string {
	return renderHermesHintWithIndicator(f, statusMessage, width, spinnerFrame, IndicatorStyleKaomoji)
}

func renderHermesHintWithIndicator(f kernel.RenderFrame, statusMessage string, width int, spinnerFrame int, indicator IndicatorStyle) string {
	return renderHermesHintWithIndicatorForSkin(f, statusMessage, width, spinnerFrame, indicator, DefaultHermesSkin())
}

func renderHermesHintWithIndicatorForSkin(f kernel.RenderFrame, statusMessage string, width int, spinnerFrame int, indicator IndicatorStyle, skin HermesSkin) string {
	var parts []string
	if f.Phase != kernel.PhaseIdle && f.Phase != kernel.PhaseFailed {
		parts = append(parts, RenderIndicatorFrame(indicator, spinnerFrame))
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
	text := strings.Join(parts, " · ")
	if width > 0 {
		text = RenderMarkdownSoftWrapTrim(text, width)
	}
	_, styles := conversationChatStyles(skin)
	return styles.Assistant.Render(text)
}

func renderHermesHintWithExtensionWorking(f kernel.RenderFrame, statusMessage string, width int, spinnerFrame int, indicator IndicatorStyle, working extensionUIWorking, skin HermesSkin) string {
	var parts []string
	if f.Phase != kernel.PhaseIdle && f.Phase != kernel.PhaseFailed {
		if !working.hideIndicator {
			if len(working.frames) > 0 {
				idx := spinnerFrame % len(working.frames)
				if idx < 0 {
					idx = 0
				}
				parts = append(parts, working.frames[idx])
			} else {
				parts = append(parts, RenderIndicatorFrame(indicator, spinnerFrame))
			}
		}
		label := strings.TrimSpace(working.text)
		if label == "" {
			label = strings.ToLower(f.Phase.String())
		}
		if label != "" {
			parts = append(parts, label)
		}
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
	text := strings.Join(parts, " · ")
	if width > 0 {
		text = RenderMarkdownSoftWrapTrim(text, width)
	}
	_, styles := conversationChatStyles(skin)
	return styles.Assistant.Render(text)
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

func conversationChatStyles(skin HermesSkin) (HermesSkin, chatStyles) {
	skin = NormalizeStyleSkin(skin)
	return skin, chatStylesFor(skin)
}

func conversationViewportTail(f kernel.RenderFrame, width, height int) string {
	return conversationViewportTailWithMode(f, width, height, false)
}

func conversationViewportTailWithMode(f kernel.RenderFrame, width, height int, forceCompact bool) string {
	return conversationViewportTailWithDetails(f, width, height, forceCompact, DefaultDetailsState())
}

func conversationViewportTailWithDetails(f kernel.RenderFrame, width, height int, forceCompact bool, details DetailsState) string {
	return conversationViewportTailWithSkinAndDetails(f, width, height, forceCompact, details, DefaultHermesSkin())
}

func conversationViewportTailWithSkinAndDetails(f kernel.RenderFrame, width, height int, forceCompact bool, details DetailsState, skin HermesSkin) string {
	skin, styles := conversationChatStyles(skin)
	if width < 4 {
		width = 4
	}
	if height < 1 {
		height = 1
	}
	wrapWidth := width - 4
	compact := forceCompact || width < 8 || height < 3
	forced := conversationForcedBlocksWithDetailsAndSkin(f, wrapWidth, compact, details, skin, styles)
	maxLines := height + 1 + len(forced)

	var visible []string
	for i := len(f.History) - 1; i >= 0; i-- {
		block := conversationMessageBlockAtWithSkin(f.History, i, wrapWidth, compact, skin, styles)
		omitted := i
		candidate := append([]string{block}, visible...)
		if omitted > 0 {
			candidate = append([]string{omittedHistorySentinelWithStyles(omitted, width, styles)}, candidate...)
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
		lines = append(lines, omittedHistorySentinelWithStyles(omitted, width, styles))
	}
	lines = append(lines, visible...)
	lines = append(lines, forced...)
	if len(lines) == 0 {
		return conversationEmptyIntroWithSkin(f, width, compact, skin)
	}
	return strings.Join(lines, "\n\n")
}

func conversationForcedBlocks(f kernel.RenderFrame, wrapWidth int, compact bool) []string {
	return conversationForcedBlocksWithDetails(f, wrapWidth, compact, DefaultDetailsState())
}

func conversationForcedBlocksWithDetails(f kernel.RenderFrame, wrapWidth int, compact bool, details DetailsState) []string {
	skin, styles := conversationChatStyles(DefaultHermesSkin())
	return conversationForcedBlocksWithDetailsAndSkin(f, wrapWidth, compact, details, skin, styles)
}

func conversationForcedBlocksWithDetailsAndSkin(f kernel.RenderFrame, wrapWidth int, compact bool, details DetailsState, skin HermesSkin, styles chatStyles) []string {
	details = NormalizeDetailsState(details)
	var blocks []string
	hasFinal := frameHasFinalAssistant(f)
	if !hasFinal {
		if progress := conversationToolProgressBlockWithModeAndStyles(f, wrapWidth, compact, details.SectionMode(DetailsSectionTools), styles); progress != "" {
			blocks = append(blocks, progress)
		}
	}
	if f.DraftText != "" && !draftDuplicatesFinalAssistant(f) {
		blocks = append(blocks, conversationDraftBlockWithSkin(f.DraftText, wrapWidth, compact, skin, styles))
	}
	if f.LastError != "" {
		blocks = append(blocks, conversationErrorBlockWithStyles(f.LastError, wrapWidth, compact, styles))
	}
	// R3 streaming feedback: when a turn is active but nothing concrete has
	// surfaced yet (no tool trace, draft, or error), show the reused
	// thinking indicator so the user is never left wondering. Suppressed the
	// moment any real signal exists so it never disturbs transcript order.
	if len(blocks) == 0 && !hasFinal && turnIsActive(f.Phase) {
		if think := conversationThinkingBlockWithModeAndSkin(compact, details.SectionMode(DetailsSectionThinking), skin, styles); think != "" {
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
	return conversationThinkingBlockWithMode(compact, DetailsModeExpanded)
}

func conversationThinkingBlockWithMode(compact bool, mode DetailsMode) string {
	_, styles := conversationChatStyles(DefaultHermesSkin())
	return conversationThinkingBlockWithModeAndStyles(compact, mode, styles)
}

func conversationThinkingBlockWithModeAndStyles(compact bool, mode DetailsMode, styles chatStyles) string {
	return conversationThinkingBlockWithModeAndSkin(compact, mode, DefaultHermesSkin(), styles)
}

func conversationThinkingBlockWithModeAndSkin(compact bool, mode DetailsMode, skin HermesSkin, styles chatStyles) string {
	if mode == DetailsModeHidden {
		return ""
	}
	t := RenderThinkingWithSkin(ThinkingState{Visible: true}, skin)
	if t == "" {
		return ""
	}
	if compact || mode == DetailsModeCollapsed {
		return compactViewportText(t)
	}
	return styles.Assistant.Render(t)
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

func conversationToolProgressBlock(f kernel.RenderFrame, wrapWidth int, compact bool) string {
	return conversationToolProgressBlockWithMode(f, wrapWidth, compact, DetailsModeExpanded)
}

func conversationToolProgressBlockWithMode(f kernel.RenderFrame, wrapWidth int, compact bool, mode DetailsMode) string {
	_, styles := conversationChatStyles(DefaultHermesSkin())
	return conversationToolProgressBlockWithModeAndStyles(f, wrapWidth, compact, mode, styles)
}

func conversationToolProgressBlockWithModeAndStyles(f kernel.RenderFrame, wrapWidth int, compact bool, mode DetailsMode, styles chatStyles) string {
	if mode == DetailsModeHidden {
		return ""
	}
	if trail := RenderToolTrail(conversationToolTrailNodes(f)); trail != "" {
		if compact || mode == DetailsModeCollapsed {
			return compactViewportText(trail)
		}
		if wrapWidth > 0 {
			trail = RenderMarkdownSoftWrapTrim(trail, wrapWidth)
		}
		return styles.ToolOutput.Render(trail)
	}
	texts := make([]string, 0, len(f.SoulEvents))
	for _, event := range f.SoulEvents {
		texts = append(texts, event.Text)
	}
	progress := trace.FormatBlock(texts)
	if progress == "" {
		return ""
	}
	if compact || mode == DetailsModeCollapsed {
		return compactViewportText(progress)
	}
	if wrapWidth > 0 {
		progress = RenderMarkdownSoftWrapTrim(progress, wrapWidth)
	}
	return styles.ToolOutput.Render(progress)
}

func conversationToolTrailNodes(f kernel.RenderFrame) []ToolCallNode {
	seen := map[string]int{}
	var nodes []ToolCallNode
	upsert := func(key, label string, status ToolCallStatus) {
		key = strings.TrimSpace(key)
		label = strings.TrimSpace(label)
		if key == "" {
			return
		}
		if label == "" {
			label = key
		}
		if idx, ok := seen[key]; ok {
			nodes[idx].Status = status
			return
		}
		seen[key] = len(nodes)
		nodes = append(nodes, ToolCallNode{Name: label, Status: status})
	}
	for _, event := range f.SoulEvents {
		key, label, status, ok := toolTrailEvent(event.Text)
		if ok {
			upsert(key, label, status)
		}
	}
	for _, msg := range f.History {
		if msg.Role == "tool" {
			upsert(msg.Name, msg.Name, ToolCallDone)
		}
	}
	return nodes
}

func toolTrailEvent(text string) (string, string, ToolCallStatus, bool) {
	text = strings.TrimSpace(text)
	switch {
	case strings.HasPrefix(text, "tool error:"):
		name, _, _ := strings.Cut(strings.TrimSpace(strings.TrimPrefix(text, "tool error:")), ":")
		return name, name, ToolCallError, strings.TrimSpace(name) != ""
	case strings.HasPrefix(text, "tool cancelled:"):
		name := strings.TrimSpace(strings.TrimPrefix(text, "tool cancelled:"))
		return name, name, ToolCallError, name != ""
	case strings.HasPrefix(text, "tool done:"), strings.HasPrefix(text, "tool completed:"):
		text = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "tool done:"), "tool completed:"))
		name, _, _ := strings.Cut(text, ":")
		return name, name, ToolCallDone, strings.TrimSpace(name) != ""
	case strings.HasPrefix(text, "tool: "):
		payload := strings.TrimSpace(strings.TrimPrefix(text, "tool: "))
		name, arg, hasArg := strings.Cut(payload, ":")
		name = strings.TrimSpace(name)
		label := name
		if hasArg && strings.TrimSpace(arg) != "" {
			label = name + ": " + strings.TrimSpace(arg)
		}
		return name, label, ToolCallRunning, name != ""
	default:
		return "", "", ToolCallRunning, false
	}
}

func conversationMessageBlock(msg hermes.Message, wrapWidth int, compact bool) string {
	skin, styles := conversationChatStyles(DefaultHermesSkin())
	return conversationMessageBlockWithSkin(msg, wrapWidth, compact, skin, styles)
}

func conversationMessageBlockWithSkin(msg hermes.Message, wrapWidth int, compact bool, skin HermesSkin, styles chatStyles) string {
	if msg.Role == "tool" {
		return conversationToolResultBlockWithStyles(msg, wrapWidth, compact, styles)
	}
	content := msg.Content
	if compact {
		content = compactViewportText(content)
	} else {
		content = RenderMarkdownSoftWrapTrim(content, wrapWidth)
	}
	return transcriptRowWithSkin(msg.Role, content, skin, styles)
}

func conversationMessageBlockAt(history []hermes.Message, idx, wrapWidth int, compact bool) string {
	skin, styles := conversationChatStyles(DefaultHermesSkin())
	return conversationMessageBlockAtWithSkin(history, idx, wrapWidth, compact, skin, styles)
}

func conversationMessageBlockAtWithSkin(history []hermes.Message, idx, wrapWidth int, compact bool, skin HermesSkin, styles chatStyles) string {
	block := conversationMessageBlockWithSkin(history[idx], wrapWidth, compact, skin, styles)
	if !compact && conversationNeedsTurnSeparator(history, idx) {
		return styles.Separator.Render("───") + "\n\n" + block
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
	_, styles := conversationChatStyles(DefaultHermesSkin())
	return conversationToolResultBlockWithStyles(msg, wrapWidth, compact, styles)
}

func conversationToolResultBlockWithStyles(msg hermes.Message, wrapWidth int, compact bool, styles chatStyles) string {
	name := strings.TrimSpace(msg.Name)
	if name == "" {
		name = "tool"
	}
	content := strings.TrimSpace(msg.Content)
	if compact {
		return styles.ToolOutput.Render("⚡ " + name + " " + compactViewportText(content))
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
	if wrapWidth < 24 {
		out := []string{"⚡ " + name}
		bodyWidth := wrapWidth - 2
		if bodyWidth < 8 {
			bodyWidth = 8
		}
		for _, line := range lines {
			wrapped := RenderMarkdownSoftWrapTrim(line, bodyWidth)
			for _, part := range strings.Split(wrapped, "\n") {
				out = append(out, "│ "+part)
			}
		}
		return styles.ToolOutput.Render(strings.Join(out, "\n"))
	}
	out := []string{"   ╭─ ⚡ " + name}
	for i, line := range lines {
		lines[i] = "   │ " + line
	}
	out = append(out, lines...)
	out = append(out, "   ╰─")
	return styles.ToolOutput.Render(strings.Join(out, "\n"))
}

func conversationDraftBlock(draft string, wrapWidth int, compact bool) string {
	skin, styles := conversationChatStyles(DefaultHermesSkin())
	return conversationDraftBlockWithSkin(draft, wrapWidth, compact, skin, styles)
}

func conversationDraftBlockWithSkin(draft string, wrapWidth int, compact bool, skin HermesSkin, styles chatStyles) string {
	if compact {
		draft = compactViewportText(draft)
	} else {
		draft = RenderMarkdownSoftWrapTrim(draft, wrapWidth)
	}
	return transcriptRowWithSkin("assistant", draft, skin, styles)
}

func conversationErrorBlock(lastError string, wrapWidth int, compact bool) string {
	_, styles := conversationChatStyles(DefaultHermesSkin())
	return conversationErrorBlockWithStyles(lastError, wrapWidth, compact, styles)
}

func conversationErrorBlockWithStyles(lastError string, wrapWidth int, compact bool, styles chatStyles) string {
	if compact {
		lastError = compactViewportText(lastError)
		return styles.Error.Render("err:") + " " + lastError
	}
	lastError = RenderMarkdownSoftWrapTrim(strings.Join(strings.Fields(lastError), " "), errorBodyWidth(wrapWidth))
	lines := strings.Split(lastError, "\n")
	prefix := styles.Error.Render("err:") + " "
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

func omittedHistorySentinel(count, width int) string {
	_, styles := conversationChatStyles(DefaultHermesSkin())
	return omittedHistorySentinelWithStyles(count, width, styles)
}

func omittedHistorySentinelWithStyles(count, width int, styles chatStyles) string {
	text := fmt.Sprintf("... %d earlier history messages omitted ...", count)
	if width >= 20 {
		text = RenderMarkdownSoftWrapTrim(text, width)
	}
	return styles.Assistant.Render(text)
}

func renderedLineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func conversationEmptyIntro(f kernel.RenderFrame, width int, compact bool) string {
	return conversationEmptyIntroWithSkin(f, width, compact, DefaultHermesSkin())
}

func conversationEmptyIntroWithSkin(f kernel.RenderFrame, width int, compact bool, skin HermesSkin) string {
	skin, styles := conversationChatStyles(skin)
	if compact {
		return styles.Assistant.Render("⚕ Gormes · /help for commands")
	}
	ctx := welcomeContext{
		Model:     f.Model,
		Provider:  f.ProviderStatus.Provider,
		Runtime:   f.ProviderStatus.Runtime,
		CWD:       hermesWorkingDirLabel(),
		SessionID: f.SessionID,
		Version:   buildInfoVersion(),
	}
	return welcomePanel(skin, ctx, width)
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
	skin, styles := conversationChatStyles(DefaultHermesSkin())
	return transcriptRowWithSkin(role, content, skin, styles)
}

func transcriptRowWithSkin(role, content string, skin HermesSkin, styles chatStyles) string {
	glyph := transcriptGlyphForSkin(role, skin)
	style := transcriptGlyphStyleForStyles(role, styles)
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
	return transcriptGlyphForSkin(role, DefaultHermesSkin())
}

func transcriptGlyphForSkin(role string, skin HermesSkin) string {
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
	_, styles := conversationChatStyles(DefaultHermesSkin())
	return transcriptGlyphStyleForStyles(role, styles)
}

func transcriptGlyphStyleForStyles(role string, styles chatStyles) lipgloss.Style {
	switch role {
	case "user":
		return styles.User
	case "assistant":
		return styles.Assistant
	default:
		return styles.Assistant
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
	return RenderTodoPanelWithSkin(items, width, m.currentSkin())
}
