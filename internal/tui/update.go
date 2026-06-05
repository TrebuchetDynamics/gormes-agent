package tui

import (
	"bytes"
	"reflect"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/keymap"
)

// Hermes keybinding types stay re-exported from package tui for the public TUI seam.
type HermesKeyKind = keymap.HermesKeyKind
type HermesKeyEvent = keymap.HermesKeyEvent
type HermesPhase = keymap.HermesPhase
type HermesBusyInputMode = keymap.HermesBusyInputMode
type HermesInputState = keymap.HermesInputState
type HermesAction = keymap.HermesAction
type HermesKeyDecision = keymap.HermesKeyDecision

const (
	HermesKeyEnter     = keymap.HermesKeyEnter
	HermesKeyCtrlJ     = keymap.HermesKeyCtrlJ
	HermesKeyCtrlC     = keymap.HermesKeyCtrlC
	HermesKeyCtrlD     = keymap.HermesKeyCtrlD
	HermesKeyCtrlL     = keymap.HermesKeyCtrlL
	HermesKeyUp        = keymap.HermesKeyUp
	HermesKeyDown      = keymap.HermesKeyDown
	HermesKeyRune      = keymap.HermesKeyRune
	HermesKeySpace     = keymap.HermesKeySpace
	HermesKeyEscape    = keymap.HermesKeyEscape
	HermesKeyTab       = keymap.HermesKeyTab
	HermesKeyBackspace = keymap.HermesKeyBackspace
	HermesKeyDelete    = keymap.HermesKeyDelete

	HermesPhaseIdle    = keymap.HermesPhaseIdle
	HermesPhaseRunning = keymap.HermesPhaseRunning

	HermesBusyInterrupt = keymap.HermesBusyInterrupt
	HermesBusyQueue     = keymap.HermesBusyQueue
	HermesBusySteer     = keymap.HermesBusySteer

	HermesActionNone                  = keymap.HermesActionNone
	HermesActionInsertNewline         = keymap.HermesActionInsertNewline
	HermesActionSubmit                = keymap.HermesActionSubmit
	HermesActionInterrupt             = keymap.HermesActionInterrupt
	HermesActionQueueForNextTurn      = keymap.HermesActionQueueForNextTurn
	HermesActionSteer                 = keymap.HermesActionSteer
	HermesActionCancel                = keymap.HermesActionCancel
	HermesActionCancelModal           = keymap.HermesActionCancelModal
	HermesActionClearDraft            = keymap.HermesActionClearDraft
	HermesActionDeleteCharUnderCursor = keymap.HermesActionDeleteCharUnderCursor
	HermesActionExit                  = keymap.HermesActionExit
	HermesActionForceQuit             = keymap.HermesActionForceQuit
	HermesActionForceRedraw           = keymap.HermesActionForceRedraw
	HermesActionHistoryPrev           = keymap.HermesActionHistoryPrev
	HermesActionHistoryNext           = keymap.HermesActionHistoryNext
	HermesActionMoveCursorUp          = keymap.HermesActionMoveCursorUp
	HermesActionMoveCursorDown        = keymap.HermesActionMoveCursorDown
	HermesActionConsumeWithEvidence   = keymap.HermesActionConsumeWithEvidence
	HermesActionToggleVoiceRecording  = keymap.HermesActionToggleVoiceRecording
)

func normalizeHermesBusyInputMode(mode HermesBusyInputMode) HermesBusyInputMode {
	return keymap.NormalizeHermesBusyInputMode(mode)
}

func ResolveHermesKey(ev HermesKeyEvent, st HermesInputState) HermesKeyDecision {
	return keymap.ResolveHermesKey(ev, st)
}

// submittedMsg is emitted after a submit Cmd completes. It carries no data —
// its only role is to signal back into Update so the inFlight flag is
// authoritatively set on the same goroutine that reads it.
type submittedMsg struct{}

// cancelledMsg is the symmetric signal for cancel Cmds.
type cancelledMsg struct{}

type activeTurnTickMsg struct{}

func activeTurnTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return activeTurnTickMsg{} })
}

// Update is the Bubble Tea event loop. MUST NOT block: every kernel
// interaction is dispatched via tea.Cmd returned values.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if isHermesShiftEnterCSIMessage(msg) {
		m.editor.InsertString("\n")
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.editor.SetWidth(maxInt(msg.Width, 20))
		if m.modelPicker != nil {
			m.modelPicker.Width = msg.Width
			m.modelPicker.Height = msg.Height
		}

	case tea.KeyMsg:
		if m.transientPage != nil && (msg.Type == tea.KeyEscape || msg.Type == tea.KeyCtrlC) {
			m.transientPage = nil
			m.statusMessage = "page closed"
			return m, tea.Batch(cmds...)
		}
		if m.modelPicker != nil {
			if cmd := m.updateModelPickerForKey(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}
		if handled := m.handleSlashCompletionKey(msg); handled {
			return m, tea.Batch(cmds...)
		}
		if got, ok := resolveVoiceRecordTeaKey(msg, m.voiceRecordKey); ok && got.Action == HermesActionToggleVoiceRecording {
			if m.voiceToggle == nil {
				key := tools.ResolveVoiceRecordKey(m.voiceRecordKey, tools.VoiceRecordKeyOptions{})
				m.statusMessage = "voice recording toggle unavailable in native TUI (" + string(key.Evidence) + "; key " + key.Display + ")"
				return m, tea.Batch(cmds...)
			}
			result, err := m.voiceToggle(VoiceToggleRequest{Action: "record", SessionID: m.SessionID()})
			if err != nil {
				m.transientPage = nil
				m.statusMessage = "voice: " + err.Error()
				return m, tea.Batch(cmds...)
			}
			if strings.TrimSpace(result.RecordKey) != "" {
				binding := tools.ResolveVoiceRecordKey(result.RecordKey, tools.VoiceRecordKeyOptions{})
				m.voiceRecordKey = binding.Raw
			}
			lines := renderVoiceToggleLines("status", result)
			m.transientPage = &TransientPageState{Title: "Voice", Body: strings.Join(lines, "\n")}
			if len(lines) > 0 {
				m.statusMessage = lines[0]
			} else {
				m.statusMessage = "voice: no status"
			}
			return m, tea.Batch(cmds...)
		}
		if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
			if handled := m.handleHistoryNavigationKey(msg.Type); handled {
				return m, tea.Batch(cmds...)
			}
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			// In-flight: cancel the turn. Idle: quit.
			if m.inFlight {
				cmds = append(cmds, m.cancelCmd())
			} else {
				return m, tea.Quit
			}
		case tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyCtrlL:
			m.forceLocalRedraw()
		case tea.KeyEnter:
			if msg.Alt {
				// Alt+Enter is treated as newline-in-editor on many terminals
				// that do not forward Shift+Enter. Fall through to let the
				// textarea insert a newline naturally.
				break
			}
			text := m.editor.Value()
			// Busy guard runs before slash dispatch and submit so that
			// overlapping user input during a long-running CLI command is
			// rejected uniformly. The evaluator itself decides which slash
			// commands bypass the busy state (e.g. /stop, /help) so the
			// operator can always abort the active command.
			if text != "" && m.busyGuard != nil {
				if v := m.busyGuard.EvaluateInput(text); v.Rejected {
					m.editor.Reset()
					if v.Evidence != "" {
						m.statusMessage = v.Evidence
					}
					return m, tea.Batch(cmds...)
				}
			}
			if m.slashRegistry != nil {
				if res := m.slashRegistry.Dispatch(text, &m); res.Handled {
					m.recordInputHistory(text)
					m.editor.Reset()
					if res.EditorText != "" {
						m.editor.SetValue(res.EditorText)
					}
					if res.StatusMessage != "" {
						m.statusMessage = res.StatusMessage
					}
					if res.Cmd != nil {
						cmds = append(cmds, res.Cmd)
					}
					return m, tea.Batch(cmds...)
				}
			}
			if text != "" {
				phase := HermesPhaseIdle
				if m.turnActive() {
					phase = HermesPhaseRunning
				}
				decision := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyEnter}, HermesInputState{
					Text:          text,
					Phase:         phase,
					BusyInputMode: m.busyInputMode,
				})
				switch decision.Action {
				case HermesActionSubmit:
					m.recordInputHistory(decision.SubmitText)
					m.editor.Reset()
					if m.offlineSmoke {
						m.applyOfflineSmokeTurn(decision.SubmitText)
						return m, tea.Batch(cmds...)
					}
					m.inFlight = true
					cmds = append(cmds, m.submitCmd(decision.SubmitText))
				case HermesActionQueueForNextTurn:
					m.recordInputHistory(decision.SubmitText)
					m.queueFollowUpDraft(decision.SubmitText)
				case HermesActionSteer:
					m.recordInputHistory(decision.SubmitText)
					if cmd := m.queueSteeringDraft(decision.SubmitText); cmd != nil {
						cmds = append(cmds, cmd)
					}
				case HermesActionInterrupt:
					m.recordInputHistory(decision.SubmitText)
					m.queueInterruptDraft(decision.SubmitText)
					cmds = append(cmds, m.cancelCmd())
				case HermesActionConsumeWithEvidence:
					m.recordInputHistory(text)
					m.editor.Reset()
					m.statusMessage = decision.Evidence
				}
			}
			// Return early so textarea's own Enter handling does not insert
			// a newline on the now-empty editor.
			return m, tea.Batch(cmds...)
		}

	case frameMsg:
		m.frame = kernel.RenderFrame(msg)
		// Authoritative inFlight reset: the kernel reports PhaseIdle on
		// success and PhaseFailed on terminal failure.
		if m.frame.Phase == kernel.PhaseIdle || m.frame.Phase == kernel.PhaseFailed {
			m.inFlight = false
			m.spinnerFrame = 0
			m.steeringMessages = QueuedMessages{}
			if cmd := m.drainQueuedFollowUp(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else if turnIsActive(m.frame.Phase) {
			cmds = append(cmds, activeTurnTickCmd())
		}
		// Extract and store active panel state so View() can render it.
		m.ExtractPanelStateFromFrame(m.frame)
		// Refresh the placeholder AFTER the inFlight transition so the editor
		// hint reflects the post-frame state (idle prompt vs. busy-time
		// affordances).
		m.editor.Placeholder = m.RunningPlaceholder()
		cmds = append(cmds, m.waitFrame())

	case submittedMsg, cancelledMsg:
		// No-op — submit/cancel are fire-and-forget. The render-frame pump
		// provides authoritative feedback via frameMsg / m.frame.Phase.

	case activeTurnTickMsg:
		if turnIsActive(m.frame.Phase) {
			m.spinnerFrame++
			cmds = append(cmds, activeTurnTickCmd())
			return m, tea.Batch(cmds...)
		}

	case modelPickerConfirmedMsg:
		cmd := m.handleModelPickerConfirmed(ModelPickerResult(msg))
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case modelSessionSetMsg:
		m.handleModelSessionSet(msg)
		return m, tea.Batch(cmds...)

	case usageAccountMsg:
		m.handleUsageAccount(msg)
		return m, tea.Batch(cmds...)
	}

	// Forward the message to the textarea for cursor / input handling.
	beforeEditorValue := m.editor.Value()
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	if m.editor.Value() != beforeEditorValue {
		m.resetInputHistoryNavigation()
		m.resetSlashCompletionDismissalForInput(m.editor.Value())
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) handleSlashCompletionKey(msg tea.KeyMsg) bool {
	if msg.Alt {
		return false
	}
	menu, ok := m.activeSlashCompletionMenu()
	if !ok {
		return false
	}
	switch msg.Type {
	case tea.KeyUp:
		idx := m.ensureSlashCompletionSelection(menu)
		m.slashCompletion.index = wrapSlashCompletionIndex(idx, -1, len(menu.completions))
		return true
	case tea.KeyDown:
		idx := m.ensureSlashCompletionSelection(menu)
		m.slashCompletion.index = wrapSlashCompletionIndex(idx, 1, len(menu.completions))
		return true
	case tea.KeyEscape:
		m.slashCompletion.dismissedFor = m.editor.Value()
		return true
	case tea.KeyTab:
		m.acceptSlashCompletion(menu, slashCompletionAcceptTab)
		return true
	case tea.KeyEnter:
		return m.acceptSlashCompletion(menu, slashCompletionAcceptEnter)
	default:
		return false
	}
}

func (m *Model) acceptSlashCompletion(menu slashCompletionMenu, trigger slashCompletionAcceptTrigger) bool {
	idx := m.ensureSlashCompletionSelection(menu)
	if idx < 0 || idx >= len(menu.completions) {
		return false
	}
	next, changed := slashCompletionAcceptedText(m.editor.Value(), menu.completions[idx], trigger)
	if !changed {
		return false
	}
	m.editor.SetValue(next)
	m.slashCompletion.dismissedFor = next
	m.slashCompletion.key = ""
	m.slashCompletion.index = 0
	m.resetInputHistoryNavigation()
	return true
}

func (m *Model) handleHistoryNavigationKey(key tea.KeyType) bool {
	if m.inputHistory == nil {
		return false
	}
	ev := HermesKeyEvent{}
	switch key {
	case tea.KeyUp:
		ev.Kind = HermesKeyUp
	case tea.KeyDown:
		ev.Kind = HermesKeyDown
	default:
		return false
	}
	decision := ResolveHermesKey(ev, m.currentHermesInputState())
	switch decision.Action {
	case HermesActionHistoryPrev:
		if text, ok := m.inputHistory.PrevFrom(m.editor.Value()); ok {
			m.editor.SetValue(text)
		}
		return true
	case HermesActionHistoryNext:
		if text, ok := m.inputHistory.Next(); ok {
			m.editor.SetValue(text)
		}
		return true
	case HermesActionMoveCursorUp, HermesActionMoveCursorDown, HermesActionNone:
		return false
	default:
		return false
	}
}

func (m *Model) currentHermesInputState() HermesInputState {
	phase := HermesPhaseIdle
	if m.turnActive() {
		phase = HermesPhaseRunning
	}
	lineCount := m.editor.LineCount()
	if lineCount < 1 {
		lineCount = 1
	}
	return HermesInputState{
		Text:          m.editor.Value(),
		LineCount:     lineCount,
		CursorRow:     m.editor.Line(),
		Phase:         phase,
		BusyInputMode: m.busyInputMode,
	}
}

func (m *Model) recordInputHistory(text string) {
	if m.inputHistory == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	m.inputHistory.Append(text)
}

func (m *Model) resetInputHistoryNavigation() {
	if m.inputHistory != nil {
		m.inputHistory.ResetNavigation()
	}
}

func resolveVoiceRecordTeaKey(msg tea.KeyMsg, raw string) (HermesKeyDecision, bool) {
	ev, ok := hermesKeyEventFromTea(msg)
	if !ok {
		return HermesKeyDecision{}, false
	}
	return ResolveHermesKey(ev, HermesInputState{VoiceRecordKey: raw}), true
}

func hermesKeyEventFromTea(msg tea.KeyMsg) (HermesKeyEvent, bool) {
	ev := HermesKeyEvent{Alt: msg.Alt}
	switch msg.Type {
	case tea.KeyCtrlJ:
		ev.Kind = HermesKeyCtrlJ
	case tea.KeyCtrlC:
		ev.Kind = HermesKeyCtrlC
	case tea.KeyCtrlD:
		ev.Kind = HermesKeyCtrlD
	case tea.KeyCtrlL:
		ev.Kind = HermesKeyCtrlL
	case tea.KeyUp:
		ev.Kind = HermesKeyUp
	case tea.KeyDown:
		ev.Kind = HermesKeyDown
	case tea.KeySpace:
		ev.Kind = HermesKeySpace
	case tea.KeyEnter:
		ev.Kind = HermesKeyEnter
	case tea.KeyEscape:
		ev.Kind = HermesKeyEscape
	case tea.KeyTab:
		ev.Kind = HermesKeyTab
	case tea.KeyBackspace:
		ev.Kind = HermesKeyBackspace
	case tea.KeyDelete:
		ev.Kind = HermesKeyDelete
	case tea.KeyRunes:
		if len(msg.Runes) != 1 {
			return HermesKeyEvent{}, false
		}
		ev.Kind = HermesKeyRune
		ev.Ch = string(msg.Runes[0])
	default:
		if ch, ok := teaCtrlRune(msg.Type); ok {
			ev.Kind = HermesKeyRune
			ev.Ch = ch
			ev.Ctrl = true
			return ev, true
		}
		return HermesKeyEvent{}, false
	}
	return ev, true
}

var hermesShiftEnterCSISequences = [][]byte{
	[]byte("\x1b[13;2u"),
	[]byte("\x1b[27;2;13~"),
	[]byte("\x1b[27;2;13u"),
}

func isHermesShiftEnterCSIMessage(msg tea.Msg) bool {
	raw, ok := byteSliceMessage(msg)
	if !ok {
		return false
	}
	for _, seq := range hermesShiftEnterCSISequences {
		if bytes.Equal(raw, seq) {
			return true
		}
	}
	return false
}

func byteSliceMessage(msg tea.Msg) ([]byte, bool) {
	value := reflect.ValueOf(msg)
	if !value.IsValid() || value.Kind() != reflect.Slice || value.Type().Elem().Kind() != reflect.Uint8 {
		return nil, false
	}
	raw := make([]byte, value.Len())
	for i := 0; i < value.Len(); i++ {
		raw[i] = byte(value.Index(i).Uint())
	}
	return raw, true
}

func teaCtrlRune(kind tea.KeyType) (string, bool) {
	switch kind {
	case tea.KeyCtrlA:
		return "a", true
	case tea.KeyCtrlB:
		return "b", true
	case tea.KeyCtrlC:
		return "c", true
	case tea.KeyCtrlD:
		return "d", true
	case tea.KeyCtrlE:
		return "e", true
	case tea.KeyCtrlF:
		return "f", true
	case tea.KeyCtrlG:
		return "g", true
	case tea.KeyCtrlH:
		return "h", true
	case tea.KeyCtrlI:
		return "i", true
	case tea.KeyCtrlJ:
		return "j", true
	case tea.KeyCtrlK:
		return "k", true
	case tea.KeyCtrlL:
		return "l", true
	case tea.KeyCtrlM:
		return "m", true
	case tea.KeyCtrlN:
		return "n", true
	case tea.KeyCtrlO:
		return "o", true
	case tea.KeyCtrlP:
		return "p", true
	case tea.KeyCtrlQ:
		return "q", true
	case tea.KeyCtrlR:
		return "r", true
	case tea.KeyCtrlS:
		return "s", true
	case tea.KeyCtrlT:
		return "t", true
	case tea.KeyCtrlU:
		return "u", true
	case tea.KeyCtrlV:
		return "v", true
	case tea.KeyCtrlW:
		return "w", true
	case tea.KeyCtrlX:
		return "x", true
	case tea.KeyCtrlY:
		return "y", true
	case tea.KeyCtrlZ:
		return "z", true
	default:
		return "", false
	}
}

func (m Model) turnActive() bool {
	return m.inFlight || turnIsActive(m.frame.Phase)
}

func (m *Model) queueFollowUpDraft(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		m.editor.Reset()
		return
	}
	m.queuedMessages.Enqueue(text)
	m.editor.Reset()
	m.statusMessage = "queued follow-up for next turn"
}

func (m *Model) queueInterruptDraft(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		m.editor.Reset()
		return
	}
	m.queuedMessages.Enqueue(text)
	m.editor.Reset()
	m.statusMessage = "interrupt requested; draft queued for next turn"
}

func (m *Model) queueSteeringDraft(text string) tea.Cmd {
	text = strings.TrimSpace(text)
	if text == "" {
		m.editor.Reset()
		return nil
	}
	m.editor.Reset()
	if m.steer == nil {
		m.queuedMessages.Enqueue(text)
		m.statusMessage = "steer unavailable; queued follow-up for next turn"
		return nil
	}
	m.steeringMessages.Enqueue(text)
	m.statusMessage = "steer queued for active turn"
	return m.steerCmd(text)
}

func (m *Model) drainQueuedFollowUp() tea.Cmd {
	for {
		text, ok := m.queuedMessages.Dequeue()
		if !ok {
			return nil
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if m.offlineSmoke {
			m.applyOfflineSmokeTurn(text)
			m.statusMessage = "queued follow-up handled offline"
			return nil
		}
		m.inFlight = true
		m.statusMessage = "queued follow-up sent"
		return m.submitCmd(text)
	}
}

// submitCmd wraps the submit callback as a tea.Cmd so it runs off the
// Update goroutine.
func (m Model) submitCmd(text string) tea.Cmd {
	submit := m.submit
	return func() tea.Msg {
		submit(text)
		return submittedMsg{}
	}
}

// steerCmd wraps the steer callback as a tea.Cmd so active-turn guidance uses
// the same non-blocking adapter pattern as submit/cancel.
func (m Model) steerCmd(text string) tea.Cmd {
	steer := m.steer
	return func() tea.Msg {
		if steer != nil {
			steer(text)
		}
		return nil
	}
}

func (m *Model) forceLocalRedraw() {
	// Clear the local view by zeroing the frame's visible content.
	// Kernel history is unchanged; next real frame repopulates.
	m.frame.History = nil
	m.frame.DraftText = ""
	m.frame.LastError = ""
}

func (m *Model) applyOfflineSmokeTurn(text string) {
	if m.frame.Model == "" {
		m.frame.Model = "offline-smoke"
		m.frame.Telemetry.Model = "offline-smoke"
	}
	m.frame.Phase = kernel.PhaseIdle
	m.frame.DraftText = ""
	m.frame.LastError = ""
	m.frame.History = append(m.frame.History,
		llm.Message{Role: "user", Content: text},
		llm.Message{Role: "assistant", Content: "Offline smoke test received your message locally. No provider call was made."},
	)
	m.inFlight = false
	m.statusMessage = "offline smoke: no provider call made"
}

// cancelCmd wraps the cancel callback as a tea.Cmd.
func (m Model) cancelCmd() tea.Cmd {
	cancel := m.cancel
	return func() tea.Msg {
		cancel()
		return cancelledMsg{}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
