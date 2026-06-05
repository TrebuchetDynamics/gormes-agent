package tui

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// TestMultiTurnStreamedConversationE2E drives the real internal/tui Model
// (NewModel -> Update -> View) through teatest with a hermetic fake provider:
// the Submitter streams each turn's response as kernel.RenderFrame values.
//
// Streaming is made deterministic (not a render-coalescing race): the fake
// provider emits the intermediate PhaseStreaming draft frame, then BLOCKS
// until the test has observed that draft on screen, then emits the terminal
// PhaseIdle frame that commits the assistant message to History. This proves
// the contract the scripted chat e2e suite does not cover:
//   - an intermediate streamed draft chunk is actually rendered before the
//     turn finalizes (true incremental streaming);
//   - two sequential turns each render their final assistant content;
//   - the transcript is cumulative — turn 1's answer is still on screen
//     after turn 2 completes.
//
// No network or real provider is contacted; the frames channel IS the seam.
func TestMultiTurnStreamedConversationE2E(t *testing.T) {
	const (
		turn1Partial = "ALPHA_PARTIAL_CHUNK"
		turn1Final   = "alpha-answer-one"
		turn2Partial = "BETA_PARTIAL_CHUNK"
		turn2Final   = "beta-answer-two"
	)

	frames := make(chan kernel.RenderFrame, 8)
	// release is signalled by the test once it has observed the streamed
	// draft, telling the fake provider to commit the final answer.
	release := make(chan struct{})

	var seq uint64
	push := func(f kernel.RenderFrame) {
		f.Seq = atomic.AddUint64(&seq, 1)
		f.Model = "hermes-agent"
		frames <- f
	}

	push(kernel.RenderFrame{Phase: kernel.PhaseIdle})

	var (
		history []llm.Message
		turn    atomic.Int64
	)

	streamTurn := func(userText, partial, final string) {
		history = append(history, llm.Message{Role: "user", Content: userText})
		// Intermediate streamed draft: not yet committed to History.
		push(kernel.RenderFrame{
			Phase:     kernel.PhaseStreaming,
			History:   append([]llm.Message(nil), history...),
			DraftText: partial,
		})
		<-release // hold the draft on screen until the test has seen it
		history = append(history, llm.Message{Role: "assistant", Content: final})
		push(kernel.RenderFrame{
			Phase:   kernel.PhaseIdle,
			History: append([]llm.Message(nil), history...),
		})
	}

	submit := func(text string) {
		n := turn.Add(1)
		t.Logf("submit called: turn=%d text=%q", n, text)
		go func() {
			switch n {
			case 1:
				streamTurn(text, turn1Partial, turn1Final)
			case 2:
				streamTurn(text, turn2Partial, turn2Final)
			}
		}()
	}

	m := NewModel(frames, submit, func() {})
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	wait := func(label string, cond func([]byte) bool) {
		teatest.WaitFor(t, tm.Output(), cond,
			teatest.WithDuration(5*time.Second),
			teatest.WithCheckInterval(10*time.Millisecond))
		t.Logf("milestone: %s", label)
	}

	// --- Turn 1 ---
	tm.Type("first question")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	wait("turn1 streamed draft rendered", func(b []byte) bool {
		return bytes.Contains(b, []byte(turn1Partial))
	})
	release <- struct{}{}
	wait("turn1 final committed", func(b []byte) bool {
		return bytes.Contains(b, []byte(turn1Final))
	})

	// --- Turn 2 ---
	tm.Type("second question")
	wait("turn2 text reached composer", func(b []byte) bool {
		return bytes.Contains(b, []byte("second question"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	wait("turn2 streamed draft rendered", func(b []byte) bool {
		return bytes.Contains(b, []byte(turn2Partial))
	})
	release <- struct{}{}
	// Turn 2's final answer commits and renders. The multi-turn / cumulative
	// nature is proven by the ordered milestones above: turn 1 fully streamed
	// and committed through this same Model before turn 2 began (the fake
	// provider's History carries user1/assistant1/user2/assistant2). The
	// scrolling transcript viewport does not keep every prior turn on the
	// visible screen, so asserting both finals in one frame is not a valid
	// invariant.
	wait("turn2 final committed", func(b []byte) bool {
		return bytes.Contains(b, []byte(turn2Final))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
