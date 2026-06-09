package gateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

type stubGoalJudge struct {
	mu        sync.Mutex
	responses []string
	err       error
	calls     int
}

func (j *stubGoalJudge) JudgeGoal(_ context.Context, _, _ string) (string, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.calls++
	if j.err != nil {
		return "", j.err
	}
	if len(j.responses) == 0 {
		return `{"done": false, "reason": "keep going"}`, nil
	}
	resp := j.responses[0]
	j.responses = j.responses[1:]
	return resp, nil
}

func (j *stubGoalJudge) callCount() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.calls
}

// loadGoal loads the current goal state from the manager.
func TestGoalJudgeFailOpen(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		judge GoalJudge
		want  string
	}{
		{name: "missing auxiliary client", judge: nil, want: GoalJudgeVerdictContinue},
		{name: "empty judge response", judge: GoalJudgeFunc(func(context.Context, string, string) (string, error) { return "", nil }), want: GoalJudgeVerdictContinue},
		{name: "malformed judge response", judge: GoalJudgeFunc(func(context.Context, string, string) (string, error) { return "not json", nil }), want: GoalJudgeVerdictContinue},
		{name: "judge error", judge: GoalJudgeFunc(func(context.Context, string, string) (string, error) { return "", errors.New("boom") }), want: GoalJudgeVerdictContinue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateGoalJudge(ctx, tc.judge, "ship it", "made progress")
			if got.Verdict != tc.want {
				t.Fatalf("EvaluateGoalJudge verdict = %q, want %q (reason=%q)", got.Verdict, tc.want, got.Reason)
			}
			if strings.TrimSpace(got.Reason) == "" {
				t.Fatal("EvaluateGoalJudge reason is empty, want fail-open evidence")
			}
		})
	}

	done := EvaluateGoalJudge(ctx, GoalJudgeFunc(func(context.Context, string, string) (string, error) {
		return "```json\n{\"done\": \"yes\", \"reason\": \"delivered\"}\n```", nil
	}), "ship it", "delivered")
	if done.Verdict != GoalJudgeVerdictDone || done.Reason != "delivered" {
		t.Fatalf("EvaluateGoalJudge done = %+v, want done delivered", done)
	}
}

func TestGatewayGoalCommandQueuesInitialTurn(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	smap := session.NewMemMap()

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		GoalMaxTurns: 4,
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	defer stopManagerTestRun(t, cancel, done)

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "u1",
		MsgID:    "m1",
		Kind:     EventSubmit,
		Text:     "/goal ship it",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1 && sentTextContains(tg, "Goal set")
	})

	got := fk.submitsSnapshot()[0]
	if got.Kind != kernel.PlatformEventSubmit || got.Text != "ship it" {
		t.Fatalf("kernel submit = %+v, want submit text ship it", got)
	}
	if strings.TrimSpace(got.SessionID) == "" {
		t.Fatalf("kernel submit session_id empty: %+v", got)
	}

	goal, ok, err := session.LoadGoal(context.Background(), smap, got.SessionID)
	if err != nil {
		t.Fatalf("LoadGoal: %v", err)
	}
	if !ok {
		t.Fatal("LoadGoal ok=false, want /goal to persist state")
	}
	if goal.Goal != "ship it" || goal.Status != session.GoalStatusActive || goal.MaxTurns != 4 {
		t.Fatalf("goal = %+v, want active ship it with budget 4", goal)
	}
}

func TestGatewayGoalCommandSetSanitizesMetadataErrors(t *testing.T) {
	ctx := context.Background()
	base := session.NewMemMap()
	if err := base.Put(ctx, "telegram:42", "sess-goal-fail"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	smap := failingTitleSessionMap{MemMap: base, err: errors.New("metadata failed\n**Injected:** bearer plain-secret")}
	tg := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		GoalMaxTurns: 4,
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u1", MsgID: "m1", Kind: EventSubmit, Text: "/goal ship it"}); err != nil {
		t.Fatalf("handleInbound: %v", err)
	}

	got := tg.sentSnapshot()[0].Text
	for _, forbidden := range []string{"plain-secret", "**Injected:**", "\n**"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("goal metadata error leaked unsafe text %q in:\n%s", forbidden, got)
		}
	}
	if got != "goal_metadata_unavailable: [redacted]" {
		t.Fatalf("goal metadata error reply = %q, want redacted", got)
	}
}

func TestGatewayGoalCommandSetSanitizesConfirmation(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	smap := session.NewMemMap()

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		GoalMaxTurns: 4,
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	defer stopManagerTestRun(t, cancel, done)

	unsafeGoal := "ship it\n**Injected:** token=plain-secret"
	tg.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u1", MsgID: "m1", Kind: EventSubmit, Text: "/goal " + unsafeGoal})

	waitFor(t, 200*time.Millisecond, func() bool { return sentTextContains(tg, "Goal set") })
	got := tg.sentSnapshot()[0].Text
	for _, forbidden := range []string{"plain-secret", "**Injected:**", "\n**"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("goal confirmation leaked unsafe text %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "⊙ Goal set (4-turn budget): [redacted]") {
		t.Fatalf("goal confirmation missing sanitized marker:\n%s", got)
	}
	if submitted := fk.submitsSnapshot()[0].Text; submitted != unsafeGoal {
		t.Fatalf("submitted goal text = %q, want raw goal preserved", submitted)
	}
}

func TestGatewayGoalPostTurnContinuationBudget(t *testing.T) {
	t.Run("continue queues one bounded continuation", func(t *testing.T) {
		tg, fk, render, smap, _, cleanup := newGoalLoopHarness(t, &stubGoalJudge{
			responses: []string{`{"done": false, "reason": "next step remains"}`},
		}, 3)
		defer cleanup()

		tg.pushInbound(goalEvent("/goal ship it", "m1"))
		waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })
		first := fk.submitsSnapshot()[0]

		render <- goalIdleFrame(first.SessionID, "made progress")

		waitFor(t, 200*time.Millisecond, func() bool {
			submits := fk.submitsSnapshot()
			return len(submits) == 2 && strings.Contains(submits[1].Text, "[Continuing toward your standing goal]")
		})
		if got := fk.submitsSnapshot()[1].Text; !strings.Contains(got, "Goal: ship it") {
			t.Fatalf("continuation prompt = %q, want goal text", got)
		}
		goal, ok, err := session.LoadGoal(context.Background(), smap, first.SessionID)
		if err != nil || !ok {
			t.Fatalf("LoadGoal = %+v ok=%v err=%v", goal, ok, err)
		}
		if goal.TurnsUsed != 1 || goal.Status != session.GoalStatusActive {
			t.Fatalf("goal after continue = %+v, want one turn used active", goal)
		}
		if !sentTextContains(tg, "Continuing toward goal") {
			t.Fatalf("sent messages = %+v, want continuation verdict", tg.sentSnapshot())
		}
	})

	t.Run("done stops without continuation", func(t *testing.T) {
		tg, fk, render, smap, _, cleanup := newGoalLoopHarness(t, &stubGoalJudge{
			responses: []string{`{"done": true, "reason": "finished"}`},
		}, 3)
		defer cleanup()

		tg.pushInbound(goalEvent("/goal ship it", "m1"))
		waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })
		first := fk.submitsSnapshot()[0]

		render <- goalIdleFrame(first.SessionID, "finished")

		waitFor(t, 200*time.Millisecond, func() bool { return sentTextContains(tg, "Goal achieved") })
		if got := len(fk.submitsSnapshot()); got != 1 {
			t.Fatalf("submit count after done = %d, want no continuation", got)
		}
		goal, ok, err := session.LoadGoal(context.Background(), smap, first.SessionID)
		if err != nil || !ok {
			t.Fatalf("LoadGoal = %+v ok=%v err=%v", goal, ok, err)
		}
		if goal.Status != session.GoalStatusDone || goal.LastReason != "finished" {
			t.Fatalf("goal after done = %+v, want done finished", goal)
		}
	})

	t.Run("budget exhaustion pauses", func(t *testing.T) {
		tg, fk, render, smap, _, cleanup := newGoalLoopHarness(t, &stubGoalJudge{
			responses: []string{`{"done": false, "reason": "more work"}`},
		}, 1)
		defer cleanup()

		tg.pushInbound(goalEvent("/goal ship it", "m1"))
		waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })
		first := fk.submitsSnapshot()[0]

		render <- goalIdleFrame(first.SessionID, "not done")

		waitFor(t, 200*time.Millisecond, func() bool { return sentTextContains(tg, "Goal paused") })
		if got := len(fk.submitsSnapshot()); got != 1 {
			t.Fatalf("submit count after budget exhaustion = %d, want no continuation", got)
		}
		goal, ok, err := session.LoadGoal(context.Background(), smap, first.SessionID)
		if err != nil || !ok {
			t.Fatalf("LoadGoal = %+v ok=%v err=%v", goal, ok, err)
		}
		if goal.Status != session.GoalStatusPaused || !strings.Contains(goal.PausedReason, "turn budget exhausted") {
			t.Fatalf("goal after budget exhaustion = %+v, want paused budget evidence", goal)
		}
	})

	t.Run("queued user message preempts automatic continuation", func(t *testing.T) {
		judge := &stubGoalJudge{
			responses: []string{`{"done": false, "reason": "would continue"}`},
		}
		tg, fk, render, _, manager, cleanup := newGoalLoopHarness(t, judge, 3)
		defer cleanup()

		tg.pushInbound(goalEvent("/goal ship it", "m1"))
		waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })
		first := fk.submitsSnapshot()[0]

		tg.pushInbound(goalEvent("user has new instructions", "m2"))
		waitFor(t, 200*time.Millisecond, func() bool { return manager.hasQueuedFollowUp() })
		render <- goalIdleFrame(first.SessionID, "made progress")

		waitFor(t, 200*time.Millisecond, func() bool {
			submits := fk.submitsSnapshot()
			return len(submits) == 2 && submits[1].Text == "user has new instructions"
		})
		if judge.callCount() != 0 {
			t.Fatalf("judge calls = %d, want user follow-up to pause continuation for this turn", judge.callCount())
		}
	})
}

func TestGatewayGoalInterruptAutoPauses(t *testing.T) {
	judge := &stubGoalJudge{
		responses: []string{`{"done": false, "reason": "should not run"}`},
	}
	tg, fk, render, smap, _, cleanup := newGoalLoopHarness(t, judge, 3)
	defer cleanup()

	tg.pushInbound(goalEvent("/goal ship it", "m1"))
	waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })
	first := fk.submitsSnapshot()[0]

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "u1",
		MsgID:    "m-stop",
		Kind:     EventCancel,
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		submits := fk.submitsSnapshot()
		return len(submits) == 2 && submits[1].Kind == kernel.PlatformEventCancel
	})

	render <- kernel.RenderFrame{Phase: kernel.PhaseCancelling, SessionID: first.SessionID}

	waitFor(t, 200*time.Millisecond, func() bool { return sentTextContains(tg, "cancelled") })
	goal, ok, err := session.LoadGoal(context.Background(), smap, first.SessionID)
	if err != nil || !ok {
		t.Fatalf("LoadGoal = %+v ok=%v err=%v", goal, ok, err)
	}
	if goal.Status != session.GoalStatusPaused || !strings.Contains(strings.ToLower(goal.PausedReason), "interrupt") {
		t.Fatalf("goal after interrupt = %+v, want paused with interrupt evidence", goal)
	}
	if judge.callCount() != 0 {
		t.Fatalf("judge calls = %d, want interrupt to skip goal judge", judge.callCount())
	}
	for _, submit := range fk.submitsSnapshot() {
		if strings.Contains(submit.Text, "[Continuing toward your standing goal]") {
			t.Fatalf("unexpected continuation submit after interrupt: %+v", submit)
		}
	}
	if !sentTextContains(tg, "Goal paused") {
		t.Fatalf("sent messages = %+v, want operator-visible goal pause notice", tg.sentSnapshot())
	}
}

func TestGatewayGoalActiveTurnPolicy(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	smap := session.NewMemMap()

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		GoalMaxTurns: 3,
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	defer stopManagerTestRun(t, cancel, done)

	tg.pushInbound(goalEvent("/goal ship it", "m1"))
	waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })

	tg.pushInbound(goalEvent("/goal status", "m-status"))
	waitFor(t, 200*time.Millisecond, func() bool { return sentTextContains(tg, "Goal active") })
	if got := len(fk.submitsSnapshot()); got != 1 {
		t.Fatalf("/goal status submitted to kernel, submit count=%d want 1", got)
	}

	tg.pushInbound(goalEvent("/goal pause", "m-pause"))
	waitFor(t, 200*time.Millisecond, func() bool { return sentTextContains(tg, "Goal paused") })
	tg.pushInbound(goalEvent("/goal resume", "m-resume"))
	waitFor(t, 200*time.Millisecond, func() bool { return sentTextContains(tg, "Goal resumed") })
	tg.pushInbound(goalEvent("/goal clear", "m-clear"))
	waitFor(t, 200*time.Millisecond, func() bool { return sentTextContains(tg, "Goal cleared") })

	tg.pushInbound(goalEvent("/goal replace it", "m-replace"))
	waitFor(t, 200*time.Millisecond, func() bool { return sentTextContains(tg, "Agent is running") })
	if got := len(fk.submitsSnapshot()); got != 1 {
		t.Fatalf("mid-run /goal replace submitted to kernel, submit count=%d want 1", got)
	}
}

func newGoalLoopHarness(t *testing.T, judge GoalJudge, maxTurns int) (*fakeChannel, *fakeKernel, chan kernel.RenderFrame, *session.MemMap, *Manager, func()) {
	t.Helper()
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	smap := session.NewMemMap()
	render := make(chan kernel.RenderFrame, 8)

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		GoalJudge:    judge,
		GoalMaxTurns: maxTurns,
	}, fk, slog.Default())
	m.setRenderChan(render)
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	return tg, fk, render, smap, m, func() { stopManagerTestRun(t, cancel, done) }
}

func goalEvent(text, msgID string) InboundEvent {
	kind, body := ParseInboundText(text)
	return InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "u1",
		MsgID:    msgID,
		Kind:     kind,
		Text:     body,
	}
}

func goalIdleFrame(sessionID, assistantText string) kernel.RenderFrame {
	return kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		SessionID: sessionID,
		History: []llm.Message{{
			Role:    "assistant",
			Content: assistantText,
		}},
	}
}

func sentTextContains(ch *fakeChannel, needle string) bool {
	for _, sent := range ch.sentSnapshot() {
		if strings.Contains(sent.Text, needle) {
			return true
		}
	}
	return false
}
