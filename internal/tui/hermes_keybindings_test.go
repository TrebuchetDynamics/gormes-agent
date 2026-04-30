package tui

import (
	"strings"
	"testing"
	"time"
)

// TestHermesKeybindings_AltEnterAndCtrlJInsertNewline proves multi-line input
// works without submit. Hermes binds both Alt+Enter (escape, enter) and Ctrl+J
// (c-j) as "insert newline" so multi-line drafts compose without ever
// submitting. The pure resolver MUST classify both as ActionInsertNewline,
// regardless of whether the editor draft is empty or contains text, and MUST
// NOT classify them as ActionSubmit even on a single-line draft.
func TestHermesKeybindings_AltEnterAndCtrlJInsertNewline(t *testing.T) {
	cases := []struct {
		name string
		ev   HermesKeyEvent
	}{
		{"alt-enter empty draft", HermesKeyEvent{Kind: HermesKeyEnter, Alt: true}},
		{"alt-enter with draft", HermesKeyEvent{Kind: HermesKeyEnter, Alt: true}},
		{"ctrl-j empty draft", HermesKeyEvent{Kind: HermesKeyCtrlJ}},
		{"ctrl-j with draft", HermesKeyEvent{Kind: HermesKeyCtrlJ}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := HermesInputState{Text: "hello", Phase: HermesPhaseIdle}
			if strings.HasPrefix(tc.name, "alt-enter empty") || strings.HasPrefix(tc.name, "ctrl-j empty") {
				st.Text = ""
			}
			got := ResolveHermesKey(tc.ev, st)
			if got.Action != HermesActionInsertNewline {
				t.Fatalf("ResolveHermesKey(%+v, %+v).Action = %v, want HermesActionInsertNewline", tc.ev, st, got.Action)
			}
			if got.Action == HermesActionSubmit {
				t.Fatalf("multi-line keybinding must never submit on its own draft")
			}
		})
	}
}

// TestHermesKeybindings_CtrlCClearsIdleDraftThenSecondCtrlCQuits proves idle
// draft clearing matches Hermes. While idle (no agent turn, no modal), the
// first Ctrl+C MUST clear the editor draft instead of exiting if any text is
// present. After the draft is empty, a subsequent Ctrl+C MUST exit. This
// matches handle_ctrl_c's "If there's text or images, clear them (like bash)"
// branch followed by the empty-buffer exit.
func TestHermesKeybindings_CtrlCClearsIdleDraftThenSecondCtrlCQuits(t *testing.T) {
	st := HermesInputState{Text: "draft text", Phase: HermesPhaseIdle}
	first := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyCtrlC}, st)
	if first.Action != HermesActionClearDraft {
		t.Fatalf("first Ctrl+C with draft = %v, want HermesActionClearDraft", first.Action)
	}
	if first.ExitProcess {
		t.Fatalf("first Ctrl+C with draft must not exit; got ExitProcess=true")
	}

	// Apply the clear, then send Ctrl+C again on an empty draft.
	st.Text = ""
	second := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyCtrlC}, st)
	if second.Action != HermesActionExit {
		t.Fatalf("second Ctrl+C on empty draft = %v, want HermesActionExit", second.Action)
	}
	if !second.ExitProcess {
		t.Fatalf("second Ctrl+C on empty draft must mark ExitProcess=true")
	}
}

// TestHermesKeybindings_CtrlCActiveTurnCancelsThenDoublePressForceQuit proves
// active-turn interrupt and force-exit timing behavior. While a turn is
// in-flight, the first Ctrl+C MUST request cancellation (HermesActionCancel),
// not exit. A second Ctrl+C within 2s MUST force-exit. A second Ctrl+C after
// the 2s window MUST cancel again rather than force-exit, mirroring Hermes's
// "press Ctrl+C again to force exit" two-second window.
func TestHermesKeybindings_CtrlCActiveTurnCancelsThenDoublePressForceQuit(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	st := HermesInputState{
		Phase: HermesPhaseRunning,
		Now:   t0,
	}
	first := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyCtrlC}, st)
	if first.Action != HermesActionCancel {
		t.Fatalf("first Ctrl+C in-flight = %v, want HermesActionCancel", first.Action)
	}
	if first.ExitProcess {
		t.Fatalf("first Ctrl+C in-flight must not exit")
	}
	if first.LastCtrlCAt.IsZero() {
		t.Fatalf("first Ctrl+C in-flight must record LastCtrlCAt for the next press to consult")
	}

	// Second press within 2s -> force exit.
	st.LastCtrlCAt = first.LastCtrlCAt
	st.Now = t0.Add(1500 * time.Millisecond)
	second := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyCtrlC}, st)
	if second.Action != HermesActionForceQuit {
		t.Fatalf("second Ctrl+C within 2s = %v, want HermesActionForceQuit", second.Action)
	}
	if !second.ExitProcess {
		t.Fatalf("second Ctrl+C within 2s must mark ExitProcess=true")
	}

	// Second press after 2s -> cancel again, NOT force exit.
	st.Now = t0.Add(2500 * time.Millisecond)
	again := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyCtrlC}, st)
	if again.Action != HermesActionCancel {
		t.Fatalf("Ctrl+C after the 2s window = %v, want HermesActionCancel", again.Action)
	}
	if again.ExitProcess {
		t.Fatalf("Ctrl+C after the 2s window must not exit")
	}
}

// TestHermesKeybindings_CtrlDDeletesCharOrExitsOnlyWhenEmpty proves
// readline-style EOF behavior. When the draft has any text, Ctrl+D MUST be
// classified as ActionDeleteCharUnderCursor and MUST NOT exit. When the draft
// is empty, Ctrl+D MUST be classified as ActionExit with ExitProcess=true.
// This matches handle_ctrl_d's "Only exit when the input is empty — same as
// bash/zsh" branch.
func TestHermesKeybindings_CtrlDDeletesCharOrExitsOnlyWhenEmpty(t *testing.T) {
	withText := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyCtrlD}, HermesInputState{Text: "abc"})
	if withText.Action != HermesActionDeleteCharUnderCursor {
		t.Fatalf("Ctrl+D with draft = %v, want HermesActionDeleteCharUnderCursor", withText.Action)
	}
	if withText.ExitProcess {
		t.Fatalf("Ctrl+D with draft must not exit; got ExitProcess=true")
	}

	empty := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyCtrlD}, HermesInputState{Text: ""})
	if empty.Action != HermesActionExit {
		t.Fatalf("Ctrl+D on empty draft = %v, want HermesActionExit", empty.Action)
	}
	if !empty.ExitProcess {
		t.Fatalf("Ctrl+D on empty draft must mark ExitProcess=true")
	}
}

// TestHermesKeybindings_BusyModeUnavailableDoesNotSubmitSlash proves unported
// busy slash text is consumed with evidence and never reaches kernel.Submit.
// While the agent is running, an Enter on a recognized-but-unported slash
// command (here `/queue`, declared ActiveTurnPolicyUnavailable in the
// CommandRegistry) MUST resolve as HermesActionConsumeWithEvidence — not
// HermesActionSubmit, HermesActionInterrupt, or HermesActionQueueForNextTurn —
// and MUST surface human-readable evidence so the operator sees why the slash
// command was not dispatched. This guards the contract that unported slash
// text never leaks to the model as a prompt body.
func TestHermesKeybindings_BusyModeUnavailableDoesNotSubmitSlash(t *testing.T) {
	for _, mode := range []HermesBusyInputMode{HermesBusyInterrupt, HermesBusyQueue, HermesBusySteer} {
		t.Run(string(mode), func(t *testing.T) {
			st := HermesInputState{
				Text:          "/queue do a thing",
				Phase:         HermesPhaseRunning,
				BusyInputMode: mode,
			}
			got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyEnter}, st)
			if got.Action != HermesActionConsumeWithEvidence {
				t.Fatalf("Enter on /queue while busy mode=%s -> %v, want HermesActionConsumeWithEvidence", mode, got.Action)
			}
			if got.Evidence == "" {
				t.Fatalf("ConsumeWithEvidence must set Evidence; got empty")
			}
			if !strings.Contains(strings.ToLower(got.Evidence), "queue") {
				t.Fatalf("Evidence = %q, want to mention the offending command", got.Evidence)
			}
			if got.SubmitText != "" {
				t.Fatalf("ConsumeWithEvidence must NOT carry SubmitText; got %q", got.SubmitText)
			}
		})
	}
}

// TestHermesKeybindings_EnterPlainTextHonorsBusyInputMode proves that, while a
// turn is running, plain prompt text is routed by busy_input_mode:
// "interrupt" (default) -> HermesActionInterrupt, "queue" -> HermesActionQueueForNextTurn,
// "steer" -> HermesActionSteer. Idle plain text always submits. This locks the
// busy_input_mode contract called out in the row's contract field.
func TestHermesKeybindings_EnterPlainTextHonorsBusyInputMode(t *testing.T) {
	cases := []struct {
		name   string
		state  HermesInputState
		want   HermesAction
	}{
		{"idle plain text submits", HermesInputState{Text: "hello", Phase: HermesPhaseIdle}, HermesActionSubmit},
		{"running interrupt mode", HermesInputState{Text: "hello", Phase: HermesPhaseRunning, BusyInputMode: HermesBusyInterrupt}, HermesActionInterrupt},
		{"running queue mode", HermesInputState{Text: "hello", Phase: HermesPhaseRunning, BusyInputMode: HermesBusyQueue}, HermesActionQueueForNextTurn},
		{"running steer mode", HermesInputState{Text: "hello", Phase: HermesPhaseRunning, BusyInputMode: HermesBusySteer}, HermesActionSteer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyEnter}, tc.state)
			if got.Action != tc.want {
				t.Fatalf("ResolveHermesKey(Enter, %+v).Action = %v, want %v", tc.state, got.Action, tc.want)
			}
			if got.Action == HermesActionSubmit && got.SubmitText != "hello" {
				t.Fatalf("Submit must carry the original draft text; got SubmitText=%q", got.SubmitText)
			}
		})
	}
}

// TestHermesKeybindings_CtrlLForcesRedraw proves Ctrl+L is classified as a
// pure full-redraw request that does not touch the editor draft, kernel
// state, or in-flight phase.
func TestHermesKeybindings_CtrlLForcesRedraw(t *testing.T) {
	st := HermesInputState{Text: "preserved", Phase: HermesPhaseRunning}
	got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyCtrlL}, st)
	if got.Action != HermesActionForceRedraw {
		t.Fatalf("Ctrl+L = %v, want HermesActionForceRedraw", got.Action)
	}
	if got.SubmitText != "" {
		t.Fatalf("Ctrl+L must not carry SubmitText; got %q", got.SubmitText)
	}
	if got.ExitProcess {
		t.Fatalf("Ctrl+L must not exit")
	}
}

// TestHermesKeybindings_HistoryUpDownOnlyAtLineBoundaries proves history
// browsing fires only when the cursor is on the first/last line of the draft
// (Hermes's auto_up / auto_down semantics).  When the draft is multi-line and
// the cursor sits on an interior line, Up/Down move the cursor instead.
func TestHermesKeybindings_HistoryUpDownOnlyAtLineBoundaries(t *testing.T) {
	// Single-line draft, cursor at row 0 of 1: Up == history prev, Down == history next.
	single := HermesInputState{Text: "draft", LineCount: 1, CursorRow: 0}
	if got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyUp}, single); got.Action != HermesActionHistoryPrev {
		t.Fatalf("Up on single-line first row = %v, want HermesActionHistoryPrev", got.Action)
	}
	if got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyDown}, single); got.Action != HermesActionHistoryNext {
		t.Fatalf("Down on single-line last row = %v, want HermesActionHistoryNext", got.Action)
	}

	// Multi-line draft, cursor on the middle line: Up/Down move the cursor.
	mid := HermesInputState{Text: "a\nb\nc", LineCount: 3, CursorRow: 1}
	if got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyUp}, mid); got.Action != HermesActionMoveCursorUp {
		t.Fatalf("Up on interior row = %v, want HermesActionMoveCursorUp", got.Action)
	}
	if got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyDown}, mid); got.Action != HermesActionMoveCursorDown {
		t.Fatalf("Down on interior row = %v, want HermesActionMoveCursorDown", got.Action)
	}

	// Multi-line draft, cursor on the first row: Up == history prev, Down moves cursor.
	top := HermesInputState{Text: "a\nb\nc", LineCount: 3, CursorRow: 0}
	if got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyUp}, top); got.Action != HermesActionHistoryPrev {
		t.Fatalf("Up on first row of multi-line draft = %v, want HermesActionHistoryPrev", got.Action)
	}
	if got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyDown}, top); got.Action != HermesActionMoveCursorDown {
		t.Fatalf("Down on first row of multi-line draft = %v, want HermesActionMoveCursorDown", got.Action)
	}

	// Multi-line draft, cursor on the last row: Down == history next, Up moves cursor.
	bottom := HermesInputState{Text: "a\nb\nc", LineCount: 3, CursorRow: 2}
	if got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyDown}, bottom); got.Action != HermesActionHistoryNext {
		t.Fatalf("Down on last row of multi-line draft = %v, want HermesActionHistoryNext", got.Action)
	}
	if got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyUp}, bottom); got.Action != HermesActionMoveCursorUp {
		t.Fatalf("Up on last row of multi-line draft = %v, want HermesActionMoveCursorUp", got.Action)
	}
}

// TestHermesKeybindings_HistoryRingPrevNextWraps proves the in-memory history
// ring behaves like prompt_toolkit's draft history: Append records the latest
// submitted draft, Prev walks backward until exhausted, Next walks forward and
// returns to an empty draft. Empty submissions are ignored to match Hermes's
// `if text:` guard.
func TestHermesKeybindings_HistoryRingPrevNextWraps(t *testing.T) {
	r := NewHermesHistory()
	r.Append("first")
	r.Append("second")
	r.Append("third")
	r.Append("") // empty must be ignored

	if got, ok := r.Prev(); !ok || got != "third" {
		t.Fatalf("Prev() = (%q, %v), want (\"third\", true)", got, ok)
	}
	if got, ok := r.Prev(); !ok || got != "second" {
		t.Fatalf("Prev() = (%q, %v), want (\"second\", true)", got, ok)
	}
	if got, ok := r.Prev(); !ok || got != "first" {
		t.Fatalf("Prev() = (%q, %v), want (\"first\", true)", got, ok)
	}
	// Hitting top: Prev is a no-op (still returns "first" or false; either way,
	// must not return a newer entry).
	if got, _ := r.Prev(); got != "first" {
		t.Fatalf("Prev() at top = %q, want stay at \"first\"", got)
	}
	if got, ok := r.Next(); !ok || got != "second" {
		t.Fatalf("Next() = (%q, %v), want (\"second\", true)", got, ok)
	}
	if got, ok := r.Next(); !ok || got != "third" {
		t.Fatalf("Next() = (%q, %v), want (\"third\", true)", got, ok)
	}
	// Walking past the newest entry returns an empty draft (back to "fresh").
	if got, _ := r.Next(); got != "" {
		t.Fatalf("Next() past newest = %q, want \"\" (fresh draft)", got)
	}
}

// TestHermesKeybindings_ModalCtrlCCancelsModalNotProcess proves that, when a
// modal panel (approval/clarify/sudo/secret) is active, Ctrl+C cancels the
// modal instead of cancelling the running turn or exiting. This matches the
// handle_ctrl_c priority list where modal cancel runs ahead of the agent
// interrupt branch.
func TestHermesKeybindings_ModalCtrlCCancelsModalNotProcess(t *testing.T) {
	st := HermesInputState{
		Phase:       HermesPhaseRunning,
		ModalActive: true,
	}
	got := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyCtrlC}, st)
	if got.Action != HermesActionCancelModal {
		t.Fatalf("Ctrl+C with modal active = %v, want HermesActionCancelModal", got.Action)
	}
	if got.ExitProcess {
		t.Fatalf("Ctrl+C cancelling a modal must not exit the process")
	}
}
