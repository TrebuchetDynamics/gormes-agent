package kernel

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

// fixture builds a kernel wired to a fresh MockClient and NoopStore.
// The caller may swap the store via fixtureWithStore.
func fixture(t *testing.T) (*Kernel, *llm.MockClient) {
	t.Helper()
	return fixtureWithStore(t, store.NewNoop())
}

func fixtureWithStore(t *testing.T, s store.Store) (*Kernel, *llm.MockClient) {
	t.Helper()
	mc := llm.NewMockClient()
	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, mc, s, telemetry.New(), nil)
	return k, mc
}

// waitForFrameMatching drains the render channel until pred matches or the
// deadline expires. The returned frame is the matching one.
func waitForFrameMatching(t *testing.T, ch <-chan RenderFrame, pred func(RenderFrame) bool, timeout time.Duration) RenderFrame {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case f, ok := <-ch:
			if !ok {
				t.Fatal("render channel closed before predicate matched")
			}
			if pred(f) {
				return f
			}
		case <-deadline:
			t.Fatal("timeout waiting for matching render frame")
		}
	}
}

// drainUntilIdle consumes render frames until one reports PhaseIdle with
// Seq > minSeq. Returns the number of frames observed (including the final
// idle frame).
func drainUntilIdle(t *testing.T, ch <-chan RenderFrame, minSeq uint64, timeout time.Duration) (int, RenderFrame) {
	t.Helper()
	deadline := time.After(timeout)
	var count int
	var last RenderFrame
	for {
		select {
		case f, ok := <-ch:
			if !ok {
				t.Fatal("render channel closed before idle")
			}
			count++
			last = f
			if f.Phase == PhaseIdle && f.Seq > minSeq {
				return count, f
			}
		case <-deadline:
			t.Fatalf("timeout after %d frames, last phase=%v seq=%d", count, last.Phase, last.Seq)
		}
	}
}

// Test 1: 2000-token burst coalesces to < 500 render frames; final history
// contains the concatenated assistant response.
func TestKernel_ProviderOutpacesTUI_Coalesces(t *testing.T) {
	k, mc := fixture(t)

	events := make([]llm.Event, 0, 2001)
	for i := 0; i < 2000; i++ {
		events = append(events, llm.Event{Kind: llm.EventToken, Token: "x", TokensOut: i + 1})
	}
	events = append(events, llm.Event{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 10, TokensOut: 2000})
	mc.Script(events, "sess-1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go k.Run(ctx)

	// Read the initial idle frame.
	initial := <-k.Render()
	if initial.Phase != PhaseIdle {
		t.Fatalf("initial phase = %v, want Idle", initial.Phase)
	}

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}

	frames, final := drainUntilIdle(t, k.Render(), initial.Seq, 5*time.Second)

	if final.DraftText != "" {
		t.Fatalf("final idle DraftText = %q, want empty after assistant history append", final.DraftText)
	}
	// The last assistant message in history must match 2000 x's.
	if len(final.History) == 0 {
		t.Fatal("no history entries after completed turn")
	}
	last := final.History[len(final.History)-1]
	if last.Role != "assistant" {
		t.Errorf("last history role = %q, want assistant", last.Role)
	}
	if last.Content != strings.Repeat("x", 2000) {
		t.Errorf("assistant content length = %d, want 2000", len(last.Content))
	}

	// Coalescing invariant: frames < 500 for a 2000-token burst.
	if frames >= 500 {
		t.Errorf("emitted %d render frames for 2000 tokens; coalescer failed to bound output", frames)
	}
}

func TestKernel_FinalIdleClearsDraftAfterHistoryAppend(t *testing.T) {
	k, mc := fixture(t)

	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "Hi! How can I help?"},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 1, TokensOut: 5},
	}, "sess-final-clear")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)

	initial := <-k.Render()
	if initial.Phase != PhaseIdle {
		t.Fatalf("initial phase = %v, want Idle", initial.Phase)
	}

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}

	_, final := drainUntilIdle(t, k.Render(), initial.Seq, 2*time.Second)

	if final.DraftText != "" {
		t.Fatalf("final idle DraftText = %q, want empty once assistant text is in history", final.DraftText)
	}
	if got, want := len(final.History), 2; got != want {
		t.Fatalf("history entries = %d, want %d: %#v", got, want, final.History)
	}
	if got := final.History[0]; got.Role != "user" || got.Content != "hi" {
		t.Fatalf("first history entry = %#v, want user hi", got)
	}
	if got := final.History[1]; got.Role != "assistant" || got.Content != "Hi! How can I help?" {
		t.Fatalf("second history entry = %#v, want final assistant reply", got)
	}
}

// Test 2: Cancel mid-stream leaves zero goroutine leak.
func TestKernel_CancelLeakFreedom(t *testing.T) {
	// Settle the harness.
	time.Sleep(50 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	k, mc := fixture(t)

	// Script a long-running stream: 500 tokens.
	events := make([]llm.Event, 0, 501)
	for i := 0; i < 500; i++ {
		events = append(events, llm.Event{Kind: llm.EventToken, Token: "t", TokensOut: i + 1})
	}
	events = append(events, llm.Event{Kind: llm.EventDone, FinishReason: "stop"})
	mc.Script(events, "")

	runCtx, cancelRun := context.WithCancel(context.Background())
	go k.Run(runCtx)

	<-k.Render() // drain initial idle frame
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}

	// Give the stream a moment to get going.
	time.Sleep(20 * time.Millisecond)
	cancelRun()

	// Drain the render channel until it closes.
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for range k.Render() {
		}
	}()

	select {
	case <-drainDone:
	case <-time.After(2 * time.Second):
		t.Fatal("render channel did not close within 2s of cancel")
	}

	// Let any lingering goroutines unwind.
	time.Sleep(250 * time.Millisecond)

	after := runtime.NumGoroutine()
	// Tolerance of +4 covers stdlib test-harness noise.
	if after > baseline+4 {
		t.Errorf("goroutine leak: baseline=%d after=%d (delta=%d)", baseline, after, after-baseline)
	}
}

// Test 3: Admission rejects oversize input; no HTTP is opened.
func TestKernel_AdmissionRejectsOversize(t *testing.T) {
	k, mc := fixture(t)
	// Do NOT script any streams — any OpenStream call would return an empty
	// stream (io.EOF immediately). We assert below that the kernel's phase
	// stays Idle, which is the strongest statement we can make here.

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render() // initial idle

	oversize := strings.Repeat("x", 300_000)
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: oversize}); err != nil {
		t.Fatal(err)
	}

	got := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.LastError != ""
	}, time.Second)

	if got.Phase != PhaseIdle {
		t.Errorf("phase = %v, want Idle (admission must fire before any HTTP)", got.Phase)
	}
	if !strings.Contains(got.LastError, "byte limit") {
		t.Errorf("LastError = %q, want it to mention the byte limit", got.LastError)
	}

	// Silence the unused mc variable.
	_ = mc
}

// Test 4: Second submit during an active turn is rejected with a
// "still processing" LastError; the in-flight turn still completes.
func TestKernel_SecondSubmitRejected(t *testing.T) {
	k, mc := fixture(t)

	// Script a very long stream so there is a meaningful window during which
	// the kernel is mid-turn. 5000 tokens at ~µs each gives us plenty of time
	// to observe at least one rejection frame before coalescing overwrites it.
	events := make([]llm.Event, 0, 5001)
	for i := 0; i < 5000; i++ {
		events = append(events, llm.Event{Kind: llm.EventToken, Token: "t", TokensOut: i + 1})
	}
	events = append(events, llm.Event{Kind: llm.EventDone, FinishReason: "stop"})
	mc.Script(events, "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render() // initial idle

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "first"}); err != nil {
		t.Fatal(err)
	}
	// Wait for Streaming before the second submit, so the kernel is
	// definitely mid-turn (not still connecting or already done).
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseStreaming
	}, time.Second)

	// Spam multiple rejections — only one needs to survive the capacity-1
	// render mailbox coalescing to prove the kernel rejects mid-turn submits.
	// The kernel emits a rejection frame for EACH rejected submit, so the
	// more we send, the more chances the observer has to see one.
	rejected := make(chan RenderFrame, 1)
	observerDone := make(chan struct{})
	go func() {
		defer close(observerDone)
		for f := range k.Render() {
			if strings.Contains(f.LastError, "still processing") {
				select {
				case rejected <- f:
				default:
				}
				return
			}
			if f.Phase == PhaseIdle && f.Seq > 2 {
				// Turn finished without us ever seeing a rejection.
				return
			}
		}
	}()

	// Fire a burst of second submits to maximise the chance one sits in the
	// render mailbox when the observer reads it.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-observerDone:
			goto check
		default:
		}
		_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "second"})
		time.Sleep(2 * time.Millisecond)
	}
check:
	select {
	case f := <-rejected:
		if !strings.Contains(f.LastError, "still processing") {
			t.Fatalf("LastError = %q, want contains 'still processing'", f.LastError)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("did not observe rejection frame for second submit")
	}
}

// Test 5: Seq strictly monotonic across 10 turns.
func TestKernel_SeqMonotonic(t *testing.T) {
	k, mc := fixture(t)
	const turns = 10
	for i := 0; i < turns; i++ {
		mc.Script([]llm.Event{
			{Kind: llm.EventToken, Token: "t", TokensOut: 1},
			{Kind: llm.EventDone, FinishReason: "stop"},
		}, "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)

	observed := make([]uint64, 0, turns*8)
	done := make(chan struct{})

	go func() {
		defer close(done)
		completedTurns := 0
		for f := range k.Render() {
			observed = append(observed, f.Seq)
			if f.Phase == PhaseIdle {
				completedTurns++
				if completedTurns >= turns+1 { // initial idle + one per turn
					return
				}
			}
		}
	}()

	// Pace submissions so each turn finishes before the next.
	for i := 0; i < turns; i++ {
		for {
			if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "q"}); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		// Wait for this turn to complete before submitting the next.
		time.Sleep(30 * time.Millisecond)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for all turns to complete")
	}

	var prev uint64 = 0
	for i, s := range observed {
		if s <= prev {
			t.Errorf("Seq regression at index %d: prev=%d current=%d", i, prev, s)
		}
		prev = s
	}
}

// Test 6: Store ack timeout trips PhaseFailed.
func TestKernel_StoreAckTimeoutFails(t *testing.T) {
	slow := store.NewSlow(500 * time.Millisecond) // well beyond kernel's 250ms deadline
	k, _ := fixtureWithStore(t, slow)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render() // initial idle

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}

	got := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseFailed
	}, 2*time.Second)

	if !strings.Contains(got.LastError, "store ack timeout") {
		t.Errorf("LastError = %q, want contains 'store ack timeout'", got.LastError)
	}
}

// Test 7: Submit fails fast when the event mailbox is full (capacity-16).
// This confirms the bounded-mailbox invariant at the TUI→kernel seam.
func TestKernel_SubmitFailsFastOnFullMailbox(t *testing.T) {
	mc := llm.NewMockClient()
	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, mc, store.NewNoop(), telemetry.New(), nil)

	// Do NOT call Run. The events channel will fill up because nobody drains it.
	for i := 0; i < PlatformEventMailboxCap; i++ {
		if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "x"}); err != nil {
			t.Fatalf("Submit %d returned %v before full", i, err)
		}
	}
	// Next submit must fail fast, not block.
	start := time.Now()
	err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "overflow"})
	elapsed := time.Since(start)
	if err != ErrEventMailboxFull {
		t.Errorf("err = %v, want ErrEventMailboxFull", err)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("Submit took %v; must fail fast, not block", elapsed)
	}
}

// Test 8: Rapid submit during Idle (no concurrent turn) just runs the turn.
// Sanity / non-regression check alongside the concurrency tests above.
func TestKernel_SequentialTurnsCompleteCleanly(t *testing.T) {
	k, mc := fixture(t)
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "a", TokensOut: 1},
		{Kind: llm.EventDone, FinishReason: "stop"},
	}, "")
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "b", TokensOut: 1},
		{Kind: llm.EventDone, FinishReason: "stop"},
	}, "")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	for i := 0; i < 2; i++ {
		if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "q"}); err != nil {
			t.Fatal(err)
		}
		// Wait for idle before next submission.
		waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
			return f.Phase == PhaseIdle && f.Seq > 1
		}, 2*time.Second)
	}
}

// TestKernel_SetSessionModelAppliesToNextTurnAndPreservesHistory proves the
// Design-B in-session model-switch seam: SetSessionModel from Idle sets a
// resident override applied to subsequent turns' provider requests, without
// resetting conversation history.
func TestKernel_SetSessionModelAppliesToNextTurnAndPreservesHistory(t *testing.T) {
	k, mc := fixture(t)
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "one"},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 1, TokensOut: 1},
	}, "sess-sm-1")
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "two"},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 1, TokensOut: 1},
	}, "sess-sm-2")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)

	initial := <-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "first"}); err != nil {
		t.Fatal(err)
	}
	_, _ = drainUntilIdle(t, k.Render(), initial.Seq, 3*time.Second)

	if err := k.SetSessionModel("", "claude-opus-test"); err != nil {
		t.Fatalf("SetSessionModel from idle = %v, want nil", err)
	}

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "second"}); err != nil {
		t.Fatal(err)
	}
	// Wait for turn 2 to actually complete: PhaseIdle with two assistant
	// replies in history. A plain seq floor would falsely match the
	// "model switched" idle frame emitted by SetSessionModel.
	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && countRole(f.History, "assistant") >= 2
	}, 3*time.Second)

	reqs := mc.Requests()
	if len(reqs) < 2 {
		t.Fatalf("want >=2 provider requests, got %d", len(reqs))
	}
	if reqs[0].Model != "hermes-agent" {
		t.Fatalf("turn 1 model = %q, want resident default hermes-agent", reqs[0].Model)
	}
	if reqs[1].Model != "claude-opus-test" {
		t.Fatalf("turn 2 model = %q, want session override claude-opus-test", reqs[1].Model)
	}

	if users := countRole(final.History, "user"); users < 2 {
		t.Fatalf("history has %d user turns, want 2 preserved across the switch (not reset):\n%+v", users, final.History)
	}
}

// countRole counts messages with the given role in a render frame's history.
func countRole(history []llm.Message, role string) int {
	var n int
	for _, m := range history {
		if m.Role == role {
			n++
		}
	}
	return n
}

// TestKernel_SetSessionModelRejectedMidTurn proves the switch is rejected with
// a typed error during an in-flight turn, mirroring ErrResetDuringTurn, with
// no resident mutation.
func TestKernel_SetSessionModelRejectedMidTurn(t *testing.T) {
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, &blockingResetClient{stream: &blockingResetStream{release: release, sessionID: "sess-sm-busy"}}, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)

	initial := <-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "go"}); err != nil {
		t.Fatal(err)
	}
	// Wait until the turn is deterministically blocked in streaming, then
	// attempt the switch. A fast token burst is not sufficient here: on CI the
	// turn can complete between observing a frame and calling SetSessionModel.
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseStreaming && f.Seq > initial.Seq
	}, 3*time.Second)

	if err := k.SetSessionModel("", "mid-turn-model"); err != ErrSetModelDuringTurn {
		t.Fatalf("SetSessionModel mid-turn err = %v, want ErrSetModelDuringTurn", err)
	}
	close(release)
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && countRole(f.History, "assistant") >= 1
	}, 3*time.Second)
}

// TestKernel_PerEventModelWinsOverSessionOverride proves the precedence
// per-event PlatformEvent.Model > resident session override > cfg.Model.
func TestKernel_PerEventModelWinsOverSessionOverride(t *testing.T) {
	k, mc := fixture(t)
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "ok"},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 1, TokensOut: 1},
	}, "sess-sm-prec")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)

	<-k.Render() // consume the initial idle frame
	if err := k.SetSessionModel("", "session-model"); err != nil {
		t.Fatalf("SetSessionModel = %v, want nil", err)
	}
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi", Model: "per-event-model"}); err != nil {
		t.Fatal(err)
	}
	// Wait for the turn to complete (assistant reply present); a plain seq
	// floor would falsely match the "model switched" idle frame.
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && countRole(f.History, "assistant") >= 1
	}, 3*time.Second)

	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("no provider requests recorded")
	}
	if got := reqs[len(reqs)-1].Model; got != "per-event-model" {
		t.Fatalf("turn model = %q, want per-event override per-event-model to win over session override", got)
	}
}

func TestKernel_SetSessionModelCrossProviderSwapsClientViaFallbackFactory(t *testing.T) {
	resident := llm.NewMockClient()
	swapped := llm.NewMockClient()
	swapped.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "swapped"},
		{Kind: llm.EventDone, FinishReason: "stop"},
	}, "sess-cross-provider")
	var captured llm.ModelRoute
	var factoryCalls int
	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		FallbackClientFactory: func(_ context.Context, route llm.ModelRoute) (llm.Client, error) {
			factoryCalls++
			captured = route
			return swapped, nil
		},
	}, resident, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	if err := k.SetSessionModel("openrouter", "openai/gpt-4o-mini"); err != nil {
		t.Fatalf("SetSessionModel cross-provider = %v, want nil", err)
	}
	if factoryCalls != 1 {
		t.Fatalf("factoryCalls = %d, want 1", factoryCalls)
	}
	if captured.Provider != "openrouter" || captured.Model != "openai/gpt-4o-mini" {
		t.Fatalf("captured route = %+v, want openrouter/openai/gpt-4o-mini", captured)
	}
	if captured.KeyEnv == "" || captured.APIMode == "" {
		t.Fatalf("captured route missing provider metadata: %+v", captured)
	}
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && countRole(f.History, "assistant") >= 1
	}, 3*time.Second)
	if got := final.History[len(final.History)-1].Content; got != "swapped" {
		t.Fatalf("assistant reply = %q, want swapped client response", got)
	}
	if len(resident.Requests()) != 0 {
		t.Fatalf("resident client handled %d requests, want 0 after cross-provider swap", len(resident.Requests()))
	}
	if len(swapped.Requests()) != 1 {
		t.Fatalf("swapped client handled %d requests, want 1", len(swapped.Requests()))
	}
}

func TestSessionModelRouteUsesFirstConfiguredCredentialEnv(t *testing.T) {
	route, ok := sessionModelRoute("openrouter", "openai/gpt-4o-mini")
	if !ok {
		t.Fatal("sessionModelRoute ok = false, want true for implemented provider")
	}
	if route.APIKeyEnv == "" {
		t.Fatalf("APIKeyEnv = empty, want first configured credential env: %+v", route)
	}
	if strings.TrimSpace(route.APIKeyEnv) != route.APIKeyEnv {
		t.Fatalf("APIKeyEnv = %q, want trimmed", route.APIKeyEnv)
	}
}

func TestShouldSwapSessionProviderTrimsAndFallsBackToConfiguredProvider(t *testing.T) {
	k := New(Config{Provider: " openrouter "}, llm.NewMockClient(), store.NewNoop(), telemetry.New(), nil)
	if k.shouldSwapSessionProvider(" openrouter ") {
		t.Fatal("shouldSwapSessionProvider returned true for configured provider with whitespace")
	}
	if !k.shouldSwapSessionProvider(" anthropic ") {
		t.Fatal("shouldSwapSessionProvider returned false for different trimmed provider")
	}

	k.sessionProvider = " anthropic "
	if k.shouldSwapSessionProvider("anthropic") {
		t.Fatal("shouldSwapSessionProvider returned true for current session provider with whitespace")
	}
}

func TestKernel_SetSessionModelUnknownProviderDoesNotMutateResidentModel(t *testing.T) {
	k, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	err := k.SetSessionModel("missing-provider", "missing-model")
	if err == nil || !strings.Contains(err.Error(), "unknown model provider") {
		t.Fatalf("SetSessionModel unknown err = %v, want unknown provider", err)
	}
	frame := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.LastError != ""
	}, 3*time.Second)
	if frame.Model != "hermes-agent" || frame.LastError == "" {
		t.Fatalf("frame = %+v, want resident model unchanged with error evidence", frame)
	}
}

// TestKernelStreamingReasoningAccumulatedInHistory proves that reasoning events
// emitted during a stream turn are accumulated and stored in the history entry.
// This enables per-turn reasoning diagnostics and future wire-level replay.
func TestKernelStreamingReasoningAccumulatedInHistory(t *testing.T) {
	k, mc := fixture(t)

	mc.Script([]llm.Event{
		{Kind: llm.EventReasoning, Reasoning: "first thought"},
		{Kind: llm.EventReasoning, Reasoning: " second thought"},
		{Kind: llm.EventToken, Token: "Final answer."},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 5, TokensOut: 3},
	}, "ses-reasoning")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = k.Run(ctx) }()

	waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseIdle
	}, time.Second)

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "think please", SessionID: "ses-reasoning"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	frame := waitForFrameMatching(t, k.render, func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.Seq > 1
	}, 2*time.Second)

	// Inspect history — the last entry should be the assistant with accumulated reasoning.
	hist := frame.History
	if len(hist) < 2 {
		t.Fatalf("history len = %d, want at least user+assistant", len(hist))
	}
	assistant := hist[len(hist)-1]
	if assistant.Role != "assistant" || assistant.Content != "Final answer." {
		t.Fatalf("assistant = %+v, want role=assistant content='Final answer.'", assistant)
	}
	if assistant.Reasoning == nil {
		t.Fatalf("assistant.Reasoning = nil, want accumulated reasoning from stream")
	}
	if assistant.Reasoning.Text != "first thought second thought" {
		t.Fatalf("assistant.Reasoning.Text = %q, want 'first thought second thought'", assistant.Reasoning.Text)
	}
}
