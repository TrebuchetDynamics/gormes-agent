package keymap

import (
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// HermesKeyKind is the closed set of physical key inputs the Hermes-parity
// resolver understands. The resolver intentionally rejects any other kind so
// the visual chrome layer (Bubble Tea Update) keeps owning textarea-internal
// keystrokes (printable runes, Backspace, Home/End, etc.). Only the keys that
// participate in Hermes's prompt_toolkit keybinding contract appear here.
type HermesKeyKind int

const (
	// HermesKeyEnter is the Enter key. Combined with the Alt modifier on the
	// HermesKeyEvent, it covers both plain Enter (submit / interrupt / queue
	// / steer / consume-with-evidence) and Alt+Enter (insert newline).
	HermesKeyEnter HermesKeyKind = iota
	// HermesKeyCtrlJ is c-j. Hermes binds it to "insert newline" because
	// most terminals deliver Ctrl+Enter as c-j.
	HermesKeyCtrlJ
	// HermesKeyCtrlC is c-c. Routes to modal cancel, active-turn interrupt,
	// idle draft clear, or process exit depending on state.
	HermesKeyCtrlC
	// HermesKeyCtrlD is c-d. Deletes the char under the cursor when the
	// draft has any text and exits only when the draft is empty.
	HermesKeyCtrlD
	// HermesKeyCtrlL is c-l. Forces a clean full-screen repaint; never
	// touches editor state, kernel state, or the in-flight phase.
	HermesKeyCtrlL
	// HermesKeyUp is the Up arrow. Browses draft history when the cursor is
	// already on the first line; otherwise moves the cursor up one row.
	HermesKeyUp
	// HermesKeyDown is the Down arrow. Browses draft history when the
	// cursor is already on the last line; otherwise moves the cursor down.
	HermesKeyDown
	// HermesKeyRune is a printable rune plus modifier metadata. It is used
	// for configurable shortcuts such as voice.record_key.
	HermesKeyRune
	// HermesKeySpace is the Space key with modifier metadata.
	HermesKeySpace
	// HermesKeyEscape is Escape with modifier metadata.
	HermesKeyEscape
	// HermesKeyTab is Tab with modifier metadata.
	HermesKeyTab
	// HermesKeyBackspace is Backspace with modifier metadata.
	HermesKeyBackspace
	// HermesKeyDelete is Delete with modifier metadata.
	HermesKeyDelete
)

// HermesKeyEvent is the pure-data input to ResolveHermesKey. It carries no
// terminal-mode metadata or Bubble Tea identity — Update is responsible for
// adapting tea.KeyMsg into this shape before consulting the resolver.
type HermesKeyEvent struct {
	Kind  HermesKeyKind
	Ch    string
	Ctrl  bool
	Alt   bool
	Meta  bool
	Super bool
	Shift bool
}

// HermesPhase mirrors the only kernel-phase distinction the keybinding
// resolver depends on: whether a turn is currently in flight. Renderer and
// status-bar concerns live elsewhere.
type HermesPhase int

const (
	// HermesPhaseIdle reports that the kernel is not currently running a
	// turn. Plain Enter submits a new turn; Ctrl+C either clears the draft
	// or exits.
	HermesPhaseIdle HermesPhase = iota
	// HermesPhaseRunning reports that a turn is in flight. Plain Enter
	// honors busy_input_mode (interrupt/queue/steer); Ctrl+C cancels the
	// turn and a second Ctrl+C within 2s force-exits.
	HermesPhaseRunning
)

// HermesBusyInputMode is the closed set of values the Hermes
// `display.busy_input_mode` config slot accepts. The default is "interrupt"
// to preserve the prompt_toolkit baseline; "queue" defers Enter into the
// next-turn queue, and "steer" injects mid-run after the next tool call.
type HermesBusyInputMode string

const (
	// HermesBusyInterrupt cancels the running turn and replaces it with the
	// new prompt. Default upstream Hermes behavior.
	HermesBusyInterrupt HermesBusyInputMode = "interrupt"
	// HermesBusyQueue stashes the new prompt for the next turn instead of
	// interrupting.
	HermesBusyQueue HermesBusyInputMode = "queue"
	// HermesBusySteer injects the new prompt mid-run after the next tool
	// call via agent.steer(); falls back to queue when steer is unavailable.
	HermesBusySteer HermesBusyInputMode = "steer"
)

func NormalizeHermesBusyInputMode(mode HermesBusyInputMode) HermesBusyInputMode {
	switch mode {
	case HermesBusyQueue, HermesBusySteer, HermesBusyInterrupt:
		return mode
	default:
		return HermesBusyInterrupt
	}
}

// HermesInputState is the pure read-only snapshot the resolver consults. The
// caller (Bubble Tea Update) builds it from the textarea + the latest kernel
// frame; the resolver never reaches back into the model.
type HermesInputState struct {
	// Text is the current editor draft.
	Text string
	// LineCount is the number of lines in Text. 0 is treated as 1 (an empty
	// draft is a single empty line).
	LineCount int
	// CursorRow is the cursor's 0-indexed row within Text. Used by Up/Down
	// to decide between cursor movement and history browsing.
	CursorRow int
	// Phase reports whether a turn is currently in flight.
	Phase HermesPhase
	// BusyInputMode picks among interrupt/queue/steer when an Enter on
	// plain text fires while Phase==HermesPhaseRunning. Zero value is
	// treated as HermesBusyInterrupt.
	BusyInputMode HermesBusyInputMode
	// VoiceRecordKey is the configured Hermes-compatible voice.record_key
	// shortcut. Empty, malformed, reserved, or unsupported values fall
	// back to Ctrl+B via the shared tools resolver.
	VoiceRecordKey string
	// ModalActive reports that an approval/clarify/sudo/secret panel is up
	// and should claim Ctrl+C ahead of the active-turn interrupt branch.
	ModalActive bool
	// LastCtrlCAt is the timestamp of the previous Ctrl+C press while a
	// turn was in flight. The resolver uses it to decide whether the next
	// Ctrl+C is a force-exit (within 2s) or another cancel request.
	LastCtrlCAt time.Time
	// Now is the wall clock the resolver should use. Tests pass a frozen
	// value; production callers pass time.Now().
	Now time.Time
}

// HermesAction is the closed set of decisions the resolver returns. Update
// translates each into a Bubble Tea command (or a state mutation) without
// re-deriving the decision.
type HermesAction int

const (
	// HermesActionNone is the zero value. The caller should let the
	// underlying Bubble Tea textarea handle the key normally (printable
	// rune, modifier-less navigation, etc.).
	HermesActionNone HermesAction = iota
	// HermesActionInsertNewline asks the editor to insert a literal newline
	// at the cursor without submitting.
	HermesActionInsertNewline
	// HermesActionSubmit asks the kernel to enqueue a fresh turn carrying
	// the resolver's SubmitText.
	HermesActionSubmit
	// HermesActionInterrupt asks the kernel to cancel the running turn and
	// replace it with the SubmitText payload.
	HermesActionInterrupt
	// HermesActionQueueForNextTurn parks SubmitText so it submits after the
	// running turn completes.
	HermesActionQueueForNextTurn
	// HermesActionSteer injects SubmitText into the running turn after the
	// next tool call.
	HermesActionSteer
	// HermesActionCancel asks the kernel to cancel the running turn (no
	// follow-up submission).
	HermesActionCancel
	// HermesActionCancelModal closes the active approval/clarify/sudo/
	// secret panel without affecting the kernel turn.
	HermesActionCancelModal
	// HermesActionClearDraft empties the editor draft (and any pending
	// attachments at the call site) without exiting.
	HermesActionClearDraft
	// HermesActionDeleteCharUnderCursor deletes the rune under the cursor.
	HermesActionDeleteCharUnderCursor
	// HermesActionExit shuts down the program cleanly. ExitProcess is also
	// set on the decision.
	HermesActionExit
	// HermesActionForceQuit shuts down immediately. ExitProcess is set; the
	// caller should bypass any clean-shutdown formality.
	HermesActionForceQuit
	// HermesActionForceRedraw triggers a full clean-screen repaint.
	HermesActionForceRedraw
	// HermesActionHistoryPrev requests the previous draft history entry.
	HermesActionHistoryPrev
	// HermesActionHistoryNext requests the next draft history entry.
	HermesActionHistoryNext
	// HermesActionMoveCursorUp moves the textarea cursor up one row.
	HermesActionMoveCursorUp
	// HermesActionMoveCursorDown moves the textarea cursor down one row.
	HermesActionMoveCursorDown
	// HermesActionConsumeWithEvidence consumes the draft (clear the editor)
	// and surfaces Evidence to the operator without dispatching the slash
	// command or submitting prompt text. Used for unported / unavailable
	// slash commands so they never leak to the model as prompt content.
	HermesActionConsumeWithEvidence
	// HermesActionToggleVoiceRecording asks the voice layer to toggle local
	// push-to-talk recording for the configured voice.record_key shortcut.
	HermesActionToggleVoiceRecording
)

// HermesKeyDecision is the resolver's typed return value. Action is closed;
// the remaining fields are populated only when relevant to the action.
type HermesKeyDecision struct {
	Action      HermesAction
	SubmitText  string
	Evidence    string
	ExitProcess bool
	// LastCtrlCAt, when non-zero, is the timestamp the caller should record
	// in HermesInputState.LastCtrlCAt for the next Ctrl+C press to consult.
	LastCtrlCAt time.Time
}

// hermesForceQuitWindow is the double-press window matching upstream
// handle_ctrl_c's `now - self._last_ctrl_c_time < 2.0` guard.
const hermesForceQuitWindow = 2 * time.Second

// ResolveHermesKey turns one HermesKeyEvent + the current HermesInputState
// into a HermesKeyDecision. The resolver is pure: it allocates no goroutines,
// performs no IO, and never mutates its inputs. The caller (Bubble Tea
// Update) is responsible for translating the decision into editor mutations
// and tea.Cmds.
func ResolveHermesKey(ev HermesKeyEvent, st HermesInputState) HermesKeyDecision {
	if matchesHermesVoiceRecordKey(ev, st.VoiceRecordKey) {
		return HermesKeyDecision{Action: HermesActionToggleVoiceRecording}
	}

	switch ev.Kind {
	case HermesKeyEnter:
		if ev.Alt || ev.Shift {
			return HermesKeyDecision{Action: HermesActionInsertNewline}
		}
		return resolveHermesEnter(st)
	case HermesKeyCtrlJ:
		return HermesKeyDecision{Action: HermesActionInsertNewline}
	case HermesKeyCtrlC:
		return resolveHermesCtrlC(st)
	case HermesKeyCtrlD:
		if st.Text != "" {
			return HermesKeyDecision{Action: HermesActionDeleteCharUnderCursor}
		}
		return HermesKeyDecision{Action: HermesActionExit, ExitProcess: true}
	case HermesKeyCtrlL:
		return HermesKeyDecision{Action: HermesActionForceRedraw}
	case HermesKeyUp:
		if isOnFirstHermesRow(st) {
			return HermesKeyDecision{Action: HermesActionHistoryPrev}
		}
		return HermesKeyDecision{Action: HermesActionMoveCursorUp}
	case HermesKeyDown:
		if isOnLastHermesRow(st) {
			return HermesKeyDecision{Action: HermesActionHistoryNext}
		}
		return HermesKeyDecision{Action: HermesActionMoveCursorDown}
	}
	return HermesKeyDecision{Action: HermesActionNone}
}

func matchesHermesVoiceRecordKey(ev HermesKeyEvent, raw string) bool {
	keyEvent, ok := voiceRecordKeyEventFromHermes(ev)
	if !ok {
		return false
	}
	return tools.MatchesVoiceRecordKey(raw, keyEvent, tools.VoiceRecordKeyOptions{})
}

func voiceRecordKeyEventFromHermes(ev HermesKeyEvent) (tools.VoiceRecordKeyEvent, bool) {
	out := tools.VoiceRecordKeyEvent{
		Ctrl:  ev.Ctrl,
		Alt:   ev.Alt,
		Meta:  ev.Meta,
		Super: ev.Super,
		Shift: ev.Shift,
	}
	switch ev.Kind {
	case HermesKeyRune:
		key := strings.ToLower(strings.TrimSpace(ev.Ch))
		if key == "" {
			return tools.VoiceRecordKeyEvent{}, false
		}
		out.Key = key
	case HermesKeySpace:
		out.Key = "space"
	case HermesKeyEnter:
		out.Key = "enter"
	case HermesKeyEscape:
		out.Key = "escape"
		out.Escape = true
	case HermesKeyTab:
		out.Key = "tab"
	case HermesKeyBackspace:
		out.Key = "backspace"
	case HermesKeyDelete:
		out.Key = "delete"
	case HermesKeyCtrlC:
		out.Key = "c"
		out.Ctrl = true
	case HermesKeyCtrlD:
		out.Key = "d"
		out.Ctrl = true
	case HermesKeyCtrlL:
		out.Key = "l"
		out.Ctrl = true
	case HermesKeyCtrlJ:
		out.Key = "j"
		out.Ctrl = true
	default:
		return tools.VoiceRecordKeyEvent{}, false
	}
	return out, true
}

func resolveHermesEnter(st HermesInputState) HermesKeyDecision {
	trimmed := strings.TrimSpace(st.Text)
	if trimmed == "" {
		return HermesKeyDecision{Action: HermesActionNone}
	}
	if strings.HasPrefix(trimmed, "/") {
		if verdict := evaluateUnportedSlash(trimmed, st.Phase == HermesPhaseRunning); verdict.consume {
			return HermesKeyDecision{Action: HermesActionConsumeWithEvidence, Evidence: verdict.evidence}
		}
	}
	if st.Phase == HermesPhaseRunning {
		mode := st.BusyInputMode
		if mode == "" {
			mode = HermesBusyInterrupt
		}
		switch mode {
		case HermesBusyQueue:
			return HermesKeyDecision{Action: HermesActionQueueForNextTurn, SubmitText: trimmed}
		case HermesBusySteer:
			return HermesKeyDecision{Action: HermesActionSteer, SubmitText: trimmed}
		default:
			return HermesKeyDecision{Action: HermesActionInterrupt, SubmitText: trimmed}
		}
	}
	return HermesKeyDecision{Action: HermesActionSubmit, SubmitText: trimmed}
}

type unportedSlashVerdict struct {
	consume  bool
	evidence string
}

func evaluateUnportedSlash(input string, busy bool) unportedSlashVerdict {
	verdict := cli.EvaluateActiveTurnVerdict(input, busy)
	if verdict.Allowed {
		return unportedSlashVerdict{}
	}
	if verdict.Evidence == "" {
		return unportedSlashVerdict{}
	}
	return unportedSlashVerdict{consume: true, evidence: verdict.Evidence}
}

func resolveHermesCtrlC(st HermesInputState) HermesKeyDecision {
	if st.ModalActive {
		return HermesKeyDecision{Action: HermesActionCancelModal}
	}
	if st.Phase == HermesPhaseRunning {
		now := st.Now
		if !st.LastCtrlCAt.IsZero() && !now.IsZero() && now.Sub(st.LastCtrlCAt) < hermesForceQuitWindow {
			return HermesKeyDecision{Action: HermesActionForceQuit, ExitProcess: true}
		}
		recordedAt := now
		if recordedAt.IsZero() {
			recordedAt = time.Unix(0, 1)
		}
		return HermesKeyDecision{Action: HermesActionCancel, LastCtrlCAt: recordedAt}
	}
	if st.Text != "" {
		return HermesKeyDecision{Action: HermesActionClearDraft}
	}
	return HermesKeyDecision{Action: HermesActionExit, ExitProcess: true}
}

func isOnFirstHermesRow(st HermesInputState) bool {
	return st.CursorRow <= 0
}

func isOnLastHermesRow(st HermesInputState) bool {
	last := st.LineCount - 1
	if last < 0 {
		last = 0
	}
	return st.CursorRow >= last
}
