package gateway

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type typingActionFakeChannel struct {
	*fakeChannel

	err     error
	actions []typingActionCall
}

type typingActionCall struct {
	ChatID string
	Action string
}

func newTypingActionFakeChannel(name string) *typingActionFakeChannel {
	return &typingActionFakeChannel{fakeChannel: newFakeChannel(name)}
}

func (f *typingActionFakeChannel) SendChatAction(_ context.Context, chatID, action string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.actions = append(f.actions, typingActionCall{ChatID: chatID, Action: action})
	return f.err
}

func (f *typingActionFakeChannel) actionSnapshot() []typingActionCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]typingActionCall, len(f.actions))
	copy(out, f.actions)
	return out
}

func TestTypingAction_FiresOnFirstStreamingFrame(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	ch := newTypingActionFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		Now: func() time.Time { return now },
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer func() {
		if coCancel != nil {
			coCancel()
		}
	}()

	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		DraftText: "working",
	}, &co, &coCancel)

	got := ch.actionSnapshot()
	if len(got) != 1 {
		t.Fatalf("typing actions = %+v, want one action", got)
	}
	if got[0].ChatID != "42" || got[0].Action != "typing" {
		t.Fatalf("typing action = %+v, want chat 42 typing", got[0])
	}
}

func TestTypingActionCapableChannelDoesNotSendStaticHourglassOnlyBubble(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	ch := newTypingActionFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		Now: func() time.Time { return now },
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer func() {
		if coCancel != nil {
			coCancel()
		}
	}()

	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase: kernel.PhaseConnecting,
	}, &co, &coCancel)

	if got := len(ch.actionSnapshot()); got != 1 {
		t.Fatalf("typing actions = %d, want 1", got)
	}
	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("sent messages = %+v, want no standalone hourglass bubble before text", sent)
	}
}

func TestTypingActionCapableChannelSendsFirstTextInsteadOfHourglass(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	ch := newTypingActionFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		Now:        func() time.Time { return now },
		CoalesceMs: 10,
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer func() {
		if coCancel != nil {
			coCancel()
		}
	}()

	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		DraftText: "working",
	}, &co, &coCancel)

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(ch.sentSnapshot()) > 0
	})
	sent := ch.sentSnapshot()
	if sent[0].Text == "⏳" {
		t.Fatalf("first sent text = %q, want assistant text instead of hourglass placeholder", sent[0].Text)
	}
	if !strings.Contains(sent[0].Text, "working") {
		t.Fatalf("first sent text = %q, want assistant draft containing %q", sent[0].Text, "working")
	}
}

func TestTypingAction_ThrottledToFourSecondWindow(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	ch := newTypingActionFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		Now: func() time.Time { return now },
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer func() {
		if coCancel != nil {
			coCancel()
		}
	}()

	for i := 0; i < 3; i++ {
		m.dispatchFrame(context.Background(), kernel.RenderFrame{
			Phase:     kernel.PhaseStreaming,
			DraftText: "working",
		}, &co, &coCancel)
	}
	if got := len(ch.actionSnapshot()); got != 1 {
		t.Fatalf("typing actions inside window = %d, want 1", got)
	}

	now = now.Add(4*time.Second + time.Millisecond)
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		DraftText: "still working",
	}, &co, &coCancel)
	if got := len(ch.actionSnapshot()); got != 2 {
		t.Fatalf("typing actions after next window = %d, want 2", got)
	}
}

func TestTypingAction_StopsOnPhaseIdle(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	ch := newTypingActionFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		Now: func() time.Time { return now },
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer func() {
		if coCancel != nil {
			coCancel()
		}
	}()

	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		DraftText: "working",
	}, &co, &coCancel)
	now = now.Add(5 * time.Second)
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "assistant", Content: "done"},
		},
	}, &co, &coCancel)

	if got := len(ch.actionSnapshot()); got != 1 {
		t.Fatalf("typing actions after idle = %d, want still 1", got)
	}
}

func TestTypingAction_NonTypingChannelStillDeliversStreamFrame(t *testing.T) {
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		CoalesceMs: 10,
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer func() {
		if coCancel != nil {
			coCancel()
		}
	}()

	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		DraftText: "working",
	}, &co, &coCancel)

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(ch.sentSnapshot()) > 0
	})
	sent := ch.sentSnapshot()
	if sent[0].Text == "⏳" {
		t.Fatalf("non-typing channel sent static placeholder %q instead of stream content", sent[0].Text)
	}
	if !strings.Contains(sent[0].Text, "working") {
		t.Fatalf("non-typing channel sent %q, want stream content", sent[0].Text)
	}
}

func TestTypingAction_FailureRecordsEvidenceNoRetry(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	ch := newTypingActionFakeChannel("telegram")
	ch.err = errors.New("telegram: 429 with token nope")
	var evidence []TypingActionEvidence
	m := NewManagerWithSubmitter(ManagerConfig{
		Now: func() time.Time { return now },
		TypingActionEvidenceSink: func(ev TypingActionEvidence) {
			evidence = append(evidence, ev)
		},
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "msg-1")

	var co *coalescer
	var coCancel context.CancelFunc
	defer func() {
		if coCancel != nil {
			coCancel()
		}
	}()

	for i := 0; i < 2; i++ {
		m.dispatchFrame(context.Background(), kernel.RenderFrame{
			Phase:     kernel.PhaseStreaming,
			DraftText: "working",
		}, &co, &coCancel)
	}

	if got := len(ch.actionSnapshot()); got != 1 {
		t.Fatalf("typing action retries inside failure window = %d, want 1", got)
	}
	if len(evidence) != 1 {
		t.Fatalf("typing evidence = %+v, want one redacted evidence item", evidence)
	}
	if evidence[0].Code != "typing_action_failed" || evidence[0].Message != "typing action failed" {
		t.Fatalf("typing evidence = %+v, want redacted failure code/message", evidence[0])
	}
}

func TestTypingAction_EvidenceSinkPanicLogged(t *testing.T) {
	var logs bytes.Buffer
	m := NewManagerWithSubmitter(ManagerConfig{
		TypingActionEvidenceSink: func(TypingActionEvidence) {
			panic("sink boom")
		},
	}, nil, slog.New(slog.NewTextHandler(&logs, nil)))

	m.recordTypingActionFailure("telegram")

	got := logs.String()
	if !strings.Contains(got, "typing_action_evidence_sink_panic") {
		t.Fatalf("typing sink panic log = %q, want typed panic evidence", got)
	}
}
