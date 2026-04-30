package gateway

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestManager_TypingActionStartsDuringStreamingAndStopsOnFinal(t *testing.T) {
	ch := newFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   5,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", MsgID: "msg-1", Kind: EventSubmit, Text: "hello"})
	waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })

	frames <- kernel.RenderFrame{Phase: kernel.PhaseConnecting, StatusText: "connecting"}
	waitFor(t, 200*time.Millisecond, func() bool {
		ch.mu.Lock()
		defer ch.mu.Unlock()
		return len(ch.typingChats) == 1 && ch.typingChats[0] == "42"
	})

	frames <- kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "partial"}
	time.Sleep(25 * time.Millisecond)
	ch.mu.Lock()
	starts := len(ch.typingChats)
	ch.mu.Unlock()
	if starts != 1 {
		t.Fatalf("typing starts after repeated streaming frames = %d, want 1", starts)
	}

	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, History: []hermes.Message{{Role: "assistant", Content: "done"}}}
	waitFor(t, 300*time.Millisecond, func() bool {
		ch.mu.Lock()
		defer ch.mu.Unlock()
		return ch.typingStops == 1
	})
}

func TestManager_TypingActionSkipsChannelsWithoutCapability(t *testing.T) {
	ch := newChannelOnlyFake("plainchat")
	frames := make(chan kernel.RenderFrame, 4)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"plainchat": "thread-1"}}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "plainchat", ChatID: "thread-1", MsgID: "msg-1", Kind: EventSubmit, Text: "hello"})
	waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })

	frames <- kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "partial"}
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, History: []hermes.Message{{Role: "assistant", Content: "done"}}}
	waitFor(t, 300*time.Millisecond, func() bool { return len(ch.sentSnapshot()) > 0 })
}
