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
)

type reactionLifecycleChannel struct {
	*channelOnlyFake

	mu          sync.Mutex
	starts      []fakeReaction
	completions []reactionCompletion
	startErr    error
	completeErr error
}

type reactionCompletion struct {
	ChatID  string
	MsgID   string
	Outcome ProcessingOutcome
}

func newReactionLifecycleChannel(name string) *reactionLifecycleChannel {
	return &reactionLifecycleChannel{channelOnlyFake: newChannelOnlyFake(name)}
}

func (r *reactionLifecycleChannel) OnProcessingStart(_ context.Context, chatID, msgID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.starts = append(r.starts, fakeReaction{ChatID: chatID, MsgID: msgID})
	return r.startErr
}

func (r *reactionLifecycleChannel) OnProcessingComplete(_ context.Context, chatID, msgID string, outcome ProcessingOutcome) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completions = append(r.completions, reactionCompletion{ChatID: chatID, MsgID: msgID, Outcome: outcome})
	return r.completeErr
}

func (r *reactionLifecycleChannel) reactionSnapshots() ([]fakeReaction, []reactionCompletion) {
	r.mu.Lock()
	defer r.mu.Unlock()
	starts := append([]fakeReaction(nil), r.starts...)
	completions := append([]reactionCompletion(nil), r.completions...)
	return starts, completions
}

func TestReactionLifecycle_ManagerStartAndSuccess(t *testing.T) {
	ch := newReactionLifecycleChannel("telegram")
	frames := make(chan kernel.RenderFrame, 4)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	defer stopManagerTestRun(t, cancel, done)

	ch.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m1",
		Kind: EventSubmit, Text: "hello",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		starts, _ := ch.reactionSnapshots()
		return len(fk.submitsSnapshot()) == 1 && len(starts) == 1
	})
	starts, _ := ch.reactionSnapshots()
	if starts[0] != (fakeReaction{ChatID: "42", MsgID: "m1"}) {
		t.Fatalf("processing starts = %+v, want chat/message ids", starts)
	}

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	}

	waitFor(t, 200*time.Millisecond, func() bool {
		_, completions := ch.reactionSnapshots()
		return len(completions) == 1
	})
	_, completions := ch.reactionSnapshots()
	if got := completions[0]; got.ChatID != "42" || got.MsgID != "m1" || got.Outcome != ProcessingOutcomeSuccess {
		t.Fatalf("processing completion = %+v, want success for original message", got)
	}
}

func TestReactionLifecycle_CancelledSuppressesSuccessFailure(t *testing.T) {
	ch := newReactionLifecycleChannel("telegram")
	frames := make(chan kernel.RenderFrame, 4)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	defer stopManagerTestRun(t, cancel, done)

	ch.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m2",
		Kind: EventSubmit, Text: "stop me",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		starts, _ := ch.reactionSnapshots()
		return len(fk.submitsSnapshot()) == 1 && len(starts) == 1
	})

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m-stop", Kind: EventCancel})
	waitFor(t, 200*time.Millisecond, func() bool {
		submits := fk.submitsSnapshot()
		return len(submits) == 2 && submits[1].Kind == kernel.PlatformEventCancel
	})
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle}

	waitFor(t, 200*time.Millisecond, func() bool {
		_, completions := ch.reactionSnapshots()
		return len(completions) == 1
	})
	_, completions := ch.reactionSnapshots()
	if got := completions[0].Outcome; got != ProcessingOutcomeCancelled {
		t.Fatalf("completion outcome = %v, want cancelled", got)
	}
}

func TestReactionLifecycle_ReactionErrorsDoNotBlockFinalDelivery(t *testing.T) {
	ch := newReactionLifecycleChannel("telegram")
	ch.startErr = errors.New("reaction_unavailable: no permission")
	ch.completeErr = errors.New("reaction_unavailable: terminal denied")
	frames := make(chan kernel.RenderFrame, 4)
	fk := &fakeKernel{}

	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, logger)
	m.setRenderChan(frames)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	defer stopManagerTestRun(t, cancel, done)

	ch.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m3",
		Kind: EventSubmit, Text: "hello",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "final answer still sends"},
		},
	}

	waitFor(t, 200*time.Millisecond, func() bool {
		sent := ch.sentSnapshot()
		return len(sent) == 1 && strings.Contains(sent[0].Text, "final answer still sends")
	})
	stopManagerTestRun(t, cancel, done)
	if !strings.Contains(logs.String(), "reaction_unavailable") {
		t.Fatalf("logs = %q, want redacted reaction_unavailable evidence", logs.String())
	}
}
