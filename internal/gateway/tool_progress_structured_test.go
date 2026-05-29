package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type structuredToolProgressFakeChannel struct {
	*fakeChannel

	toolProgress []ToolProgressEvent
}

func newStructuredToolProgressFakeChannel(name string) *structuredToolProgressFakeChannel {
	return &structuredToolProgressFakeChannel{fakeChannel: newFakeChannel(name)}
}

func (f *structuredToolProgressFakeChannel) SendToolProgress(_ context.Context, _ string, progress ToolProgressEvent) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.toolProgress = append(f.toolProgress, progress)
	return progress.ID, nil
}

func (f *structuredToolProgressFakeChannel) toolProgressSnapshot() []ToolProgressEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]ToolProgressEvent, len(f.toolProgress))
	copy(out, f.toolProgress)
	return out
}

func TestManager_Outbound_NavivoxToolProgressUsesStructuredEvents(t *testing.T) {
	nv := newStructuredToolProgressFakeChannel("navivox")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"navivox": "s-1"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(nv); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	nv.pushInbound(InboundEvent{
		Platform: "navivox", ChatID: "s-1", MsgID: "req-1",
		Kind: EventSubmit, Text: "open the internal dashboard",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: browser_navigate: https://secret.example/dashboard?token=plain-secret-token"},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		return len(nv.toolProgressSnapshot()) >= 1
	})
	progress := nv.toolProgressSnapshot()
	if len(progress) != 1 {
		t.Fatalf("structured tool progress = %+v, want one event", progress)
	}
	if progress[0].ID == "" || progress[0].ToolName != "browser_navigate" || progress[0].Status != ToolProgressStarted {
		t.Fatalf("structured tool progress event = %+v", progress[0])
	}
	for _, forbidden := range []string{"secret.example", "plain-secret-token"} {
		if strings.Contains(progress[0].Summary, forbidden) {
			t.Fatalf("structured tool progress leaked raw argument %q in %+v", forbidden, progress[0])
		}
	}
	if sent := nv.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("navivox received text tool progress instead of structured event: %+v", sent)
	}

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: browser_navigate: https://secret.example/dashboard?token=plain-secret-token"},
		},
	}
	waitFor(t, 500*time.Millisecond, func() bool {
		return len(nv.toolProgressSnapshot()) >= 2
	})
	progress = nv.toolProgressSnapshot()
	if progress[1].ID != progress[0].ID || progress[1].Status != ToolProgressUpdated {
		t.Fatalf("structured tool update event = %+v, first=%+v", progress[1], progress[0])
	}
	for _, forbidden := range []string{"secret.example", "plain-secret-token"} {
		if strings.Contains(progress[1].Summary, forbidden) {
			t.Fatalf("structured tool update leaked raw argument %q in %+v", forbidden, progress[1])
		}
	}

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "open the internal dashboard"},
			{Role: "assistant", Content: "The dashboard is open."},
		},
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: browser_navigate: https://secret.example/dashboard?token=plain-secret-token"},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		return len(nv.toolProgressSnapshot()) >= 2 && len(nv.sentSnapshot()) >= 1
	})
	progress = nv.toolProgressSnapshot()
	if progress[len(progress)-1].ID != progress[0].ID || progress[len(progress)-1].Status != ToolProgressFinished {
		t.Fatalf("final structured tool progress event = %+v, first=%+v", progress[len(progress)-1], progress[0])
	}
	for _, sent := range nv.sentSnapshot() {
		if strings.Contains(sent.Text, "browser_navigate") || strings.Contains(sent.Text, "plain-secret-token") {
			t.Fatalf("final navivox text leaked tool progress: %+v", nv.sentSnapshot())
		}
	}
}
