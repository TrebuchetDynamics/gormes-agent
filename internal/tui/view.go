package tui

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/trace"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/banner"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/extensionui"
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
	editorW := m.width
	if editorW < 10 {
		editorW = 10
	}
	editorHeight := promptHeightForValue(editor.Value(), editorW)
	if editorHeight < 1 {
		editorHeight = 1
	}
	composerChromeWidth := m.width
	if !showComposerInputChrome(m.width, m.height) {
		composerChromeWidth = 0
	}
	composerExtraRows := composerInputChromeExtraRows(composerChromeWidth)
	hint := m.renderHermesHint()
	hintOverhead := 0
	if hint != "" {
		hintOverhead = 1
	}
	statusBarMode := normalizeStatusBarMode(m.statusBarMode)
	chromeOverhead := 2 // continuous input rules above and below the prompt
	if statusBarMode != StatusBarModeOff {
		chromeOverhead++
	}
	// The composer grows with the draft (Hermes parity), but it must never
	// consume so much height that the conversation, chrome, and status bar are
	// pushed off-screen. Cap it to leave room for chrome plus a minimum
	// conversation viewport; taller drafts scroll inside the textarea.
	const minConversationHeight = 3
	maxEditorHeight := m.height - chromeOverhead - hintOverhead - minConversationHeight - composerExtraRows
	if maxEditorHeight < 1 {
		maxEditorHeight = 1
	}
	if editorHeight > maxEditorHeight {
		editorHeight = maxEditorHeight
	}
	editor.SetHeight(editorHeight)
	editorBlockH := editorHeight + composerExtraRows
	convH := m.height - editorBlockH - chromeOverhead - hintOverhead
	if convH < minConversationHeight {
		convH = minConversationHeight
	}

	convW := m.width
	if convW < 4 {
		convW = 4
	}

	conv := m.conversationViewportTailWithSkinAndDetails(convW, convH)

	editor.SetWidth(editorW)
	prompt := RenderComposerInputChrome(ComposerInputChrome{
		Width:     composerChromeWidth,
		Prompt:    m.renderComposerPrompt(editor),
		Draft:     editor.Value(),
		Skin:      m.currentSkin(),
		Focused:   m.composerChatFocused(),
		Multiline: editorHeight > 1,
	})

	statusBar := ""
	if statusBarMode != StatusBarModeOff {
		statusBar = RenderHermesStatusBarWithSkin(m.hermesStatusModelFromFrame(), m.width, m.currentSkin())
		if footer, ok := m.renderExtensionFooter(m.width); ok {
			statusBar = footer
		} else {
			statusBar = m.renderExtensionStatusLine(statusBar, m.width)
		}
	}

	completions := m.renderActiveSlashCompletionMenu(editor.Value())

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
	// Hermes' welcome copy already tells the operator how to start; the bottom
	// composer itself stays as a clean bare prompt when empty instead of showing
	// an inline placeholder sentence.
	editor.Placeholder = ""
	if focused {
		_ = editor.Focus()
	} else {
		editor.Blur()
	}
	return alignComposerContinuationPrompts(editor.View())
}

func alignComposerContinuationPrompts(view string) string {
	lines := strings.Split(view, "\n")
	if len(lines) <= 1 {
		return view
	}
	for i := 1; i < len(lines); i++ {
		if line, ok := composerContinuationPromptBlank(lines[i]); ok {
			lines[i] = line
		}
	}
	return strings.Join(lines, "\n")
}

func composerContinuationPromptBlank(line string) (string, bool) {
	plain := StripANSIForTUI(line)
	if !strings.HasPrefix(plain, "❯ ") {
		return line, false
	}
	cut := byteIndexAfterVisiblePrefix(line, "❯ ")
	if cut < 0 {
		return "  " + strings.TrimPrefix(plain, "❯ "), true
	}
	return "  " + line[cut:], true
}

func byteIndexAfterVisiblePrefix(s, visiblePrefix string) int {
	visibleIdx := 0
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			i = skipANSISequence(s, i)
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 0 {
			return -1
		}
		want, wantSize := utf8.DecodeRuneInString(visiblePrefix[visibleIdx:])
		if want == utf8.RuneError && wantSize == 0 {
			return i
		}
		if r != want {
			return -1
		}
		i += size
		visibleIdx += wantSize
		if visibleIdx >= len(visiblePrefix) {
			return i
		}
	}
	return -1
}

func skipANSISequence(s string, start int) int {
	if start+1 >= len(s) {
		return len(s)
	}
	switch s[start+1] {
	case '[':
		for i := start + 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
		return len(s)
	case ']':
		for i := start + 2; i < len(s); i++ {
			if s[i] == '\a' {
				return i + 1
			}
			if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return len(s)
	default:
		return start + 2
	}
}

func (m Model) composerChatFocused() bool {
	return !m.IsPanelActive() && m.transientPage == nil && m.modelPicker == nil
}

func (m Model) renderHermesHint() string {
	if strings.TrimSpace(m.statusMessage) != "" && m.frame.Phase == kernel.PhaseIdle {
		return ""
	}
	skin := m.currentSkin()
	if working, ok := m.extensionWorkingIndicator(); ok {
		return renderHermesHintWithExtensionWorking(m.frame, m.statusMessage, m.width, m.spinnerFrame, m.indicatorStyle, working, skin)
	}
	return renderHermesHintWithIndicatorForSkin(m.frame, m.statusMessage, m.width, m.spinnerFrame, m.indicatorStyle, skin)
}

func renderHermesHintWithIndicatorForSkin(f kernel.RenderFrame, statusMessage string, width int, spinnerFrame int, indicator IndicatorStyle, skin HermesSkin) string {
	var parts []string
	if f.Phase != kernel.PhaseIdle && f.Phase != kernel.PhaseFailed {
		parts = append(parts, RenderIndicatorFrame(indicator, spinnerFrame))
		parts = append(parts, strings.ToLower(f.Phase.String()))
	} else if statusMessage != "" {
		parts = append(parts, statusMessage)
	}
	if len(parts) == 0 {
		return ""
	}
	text := strings.Join(parts, " ")
	if width > 0 {
		text = RenderMarkdownSoftWrapTrim(text, width)
	}
	_, styles := conversationChatStyles(skin)
	return styles.Assistant.Render(text)
}

func renderHermesHintWithExtensionWorking(f kernel.RenderFrame, statusMessage string, width int, spinnerFrame int, indicator IndicatorStyle, working extensionui.Working, skin HermesSkin) string {
	var parts []string
	if f.Phase != kernel.PhaseIdle && f.Phase != kernel.PhaseFailed {
		if !working.HideIndicator {
			if len(working.Frames) > 0 {
				idx := spinnerFrame % len(working.Frames)
				if idx < 0 {
					idx = 0
				}
				parts = append(parts, working.Frames[idx])
			} else {
				parts = append(parts, RenderIndicatorFrame(indicator, spinnerFrame))
			}
		}
		label := strings.TrimSpace(working.Text)
		if label == "" {
			label = strings.ToLower(f.Phase.String())
		}
		if label != "" {
			parts = append(parts, label)
		}
	} else if statusMessage != "" {
		parts = append(parts, statusMessage)
	}
	if len(parts) == 0 {
		return ""
	}
	text := strings.Join(parts, " ")
	if width > 0 {
		text = RenderMarkdownSoftWrapTrim(text, width)
	}
	_, styles := conversationChatStyles(skin)
	return styles.Assistant.Render(text)
}

func promptHeightForValue(value string, width int) int {
	if value == "" {
		return 1
	}
	textColumns := width - lipgloss.Width("❯ ")
	if textColumns < 1 {
		textColumns = 1
	}
	lines := 0
	for _, line := range strings.Split(value, "\n") {
		lineWidth := lipgloss.Width(StripANSIForTUI(line))
		visualLines := (lineWidth + textColumns - 1) / textColumns
		if visualLines < 1 {
			visualLines = 1
		}
		lines += visualLines
	}
	if lines < 1 {
		return 1
	}
	return lines
}

// hermesStatusModelFromFrame projects the kernel render frame onto the data
// shape expected by RenderHermesStatusBar.
func hermesStatusModelFromFrame(f kernel.RenderFrame) HermesStatusModel {
	return hermesStatusModelFromFrameWithProfile(f, "")
}

func (m Model) hermesStatusModelFromFrame() HermesStatusModel {
	out := hermesStatusModelFromFrameWithProfile(m.frame, m.profileName)
	if notice := strings.TrimSpace(m.statusMessage); notice != "" && m.frame.Phase == kernel.PhaseIdle {
		out.StatusLabel = notice
	}
	if !m.sessionStartedAt.IsZero() {
		now := m.statusNow
		if now == nil {
			now = time.Now
		}
		elapsed := now().Sub(m.sessionStartedAt)
		if elapsed > 0 {
			out.SessionDuration = int64(elapsed.Seconds())
		}
	}
	return out
}

func hermesStatusModelFromFrameWithProfile(f kernel.RenderFrame, profileName string) HermesStatusModel {
	// The active profile and cwd are already visible in the welcome panel; keep
	// the chat status row focused on Hermes' model/context/timer segments and
	// avoid the Gormes-specific `profile main · cwd` tail shown in older builds.
	_ = strings.TrimSpace(profileName)
	out := HermesStatusModel{
		StatusLabel:      hermesStatusLabelFromPhase(f.Phase),
		ModelName:        f.Model,
		ReasoningEffort:  string(f.ReasoningEffort.Requested),
		PromptElapsed:    0,
		PromptLive:       false,
		HasPromptElapsed: true,
		CWDLabel:         "",
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
	return conversationViewportTailWithSkinDetailsAndProfile(f, width, height, forceCompact, details, skin, "", "")
}

func (m Model) conversationViewportTailWithSkinAndDetails(width, height int) string {
	return conversationViewportTailWithSkinDetailsAndProfile(m.frame, width, height, m.compactTranscript, m.detailsState, m.currentSkin(), m.profileName, m.profileBaseHome)
}

func conversationViewportTailWithSkinDetailsAndProfile(f kernel.RenderFrame, width, height int, forceCompact bool, details DetailsState, skin HermesSkin, profileName string, homeLabel string) string {
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
		return conversationEmptyIntroWithProfileAndSkin(f, width, compact, profileName, homeLabel, skin)
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
	// Keep active-turn waiting feedback in the single hint/status area. Hermes'
	// chat transcript stays quiet until text or tool progress exists; injecting a
	// separate "Reasoning..." assistant block makes the prompt feel noisy.
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

func lastAssistantContent(history []llm.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			return history[i].Content
		}
	}
	return ""
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

func conversationMessageBlock(msg llm.Message, wrapWidth int, compact bool) string {
	skin, styles := conversationChatStyles(DefaultHermesSkin())
	return conversationMessageBlockWithSkin(msg, wrapWidth, compact, skin, styles)
}

func conversationMessageBlockWithSkin(msg llm.Message, wrapWidth int, compact bool, skin HermesSkin, styles chatStyles) string {
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

func conversationMessageBlockAtWithSkin(history []llm.Message, idx, wrapWidth int, compact bool, skin HermesSkin, styles chatStyles) string {
	block := conversationMessageBlockWithSkin(history[idx], wrapWidth, compact, skin, styles)
	if !compact && conversationNeedsTurnSeparator(history, idx) {
		return styles.Separator.Render("───") + "\n\n" + block
	}
	return block
}

func conversationNeedsTurnSeparator(history []llm.Message, idx int) bool {
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

func conversationToolResultBlockWithStyles(msg llm.Message, wrapWidth int, compact bool, styles chatStyles) string {
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

func conversationDraftBlockWithSkin(draft string, wrapWidth int, compact bool, skin HermesSkin, styles chatStyles) string {
	if compact {
		draft = compactViewportText(draft)
	} else {
		draft = RenderMarkdownSoftWrapTrim(draft, wrapWidth)
	}
	return transcriptRowWithSkin("assistant", draft, skin, styles)
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

func conversationEmptyIntroWithSkin(f kernel.RenderFrame, width int, compact bool, skin HermesSkin) string {
	return conversationEmptyIntroWithProfileAndSkin(f, width, compact, "", "", skin)
}

func conversationEmptyIntroWithProfileAndSkin(f kernel.RenderFrame, width int, compact bool, profileName string, homeLabel string, skin HermesSkin) string {
	skin, styles := conversationChatStyles(skin)
	if compact {
		return styles.Assistant.Render("⚕ Gormes · /help for commands")
	}
	homeLabel = strings.TrimSpace(homeLabel)
	if homeLabel == "" {
		homeLabel = hermesWorkingDirLabel()
	}
	ctx := banner.WelcomeContext{
		Model:     f.Model,
		Provider:  f.ProviderStatus.Provider,
		Runtime:   f.ProviderStatus.Runtime,
		CWD:       homeLabel,
		Profile:   strings.TrimSpace(profileName),
		SessionID: f.SessionID,
		Version:   buildInfoVersion(),
	}
	return banner.WelcomePanel(skin, ctx, width)
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

func transcriptRowWithSkin(role, content string, skin HermesSkin, styles chatStyles) string {
	glyph := transcriptGlyphForSkin(role, skin)
	style := transcriptGlyphStyleForStyles(role, styles)
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return style.Render(glyph)
	}

	prefixPad := transcriptPrefixPad(role)
	prefix := prefixPad + style.Render(glyph) + " "
	continuation := strings.Repeat(" ", lipgloss.Width(prefixPad)+lipgloss.Width(glyph)+1)
	for i, line := range lines {
		if i == 0 {
			lines[i] = prefix + line
			continue
		}
		lines[i] = continuation + line
	}
	return strings.Join(lines, "\n")
}

func transcriptPrefixPad(role string) string {
	if role == "user" {
		return "  "
	}
	return ""
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
