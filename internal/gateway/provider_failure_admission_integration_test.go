package gateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

// TestManager_KernelInbound_AdmissionAcceptedAfterProviderHTMLError reproduces
// the production wedge: a provider returns a 403 + HTML body, the gateway
// sanitizes the error, but a follow-up user message hits "admission: still
// processing previous turn". The kernel-only and manager-only tests miss the
// integration gap where the manager's pinned turn or the kernel's phase fails
// to clear after the failed open-stream attempt.
func TestManager_KernelInbound_AdmissionAcceptedAfterProviderHTMLError(t *testing.T) {
	provider := &htmlErrorThenSuccessProvider{
		htmlErr: errors.New("Forbidden: provider returned HTML error body"),
	}

	k := kernel.New(kernel.Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, provider, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = k.Run(ctx) }()

	m := NewManager(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, k, slog.Default())

	ch := newChannelOnlyFake("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatalf("register: %v", err)
	}

	go func() { _ = m.Run(ctx) }()

	// First inbound: should reach the provider, fail with HTML 403, send the
	// sanitized error reply, and clear the active turn.
	ch.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "1",
		Kind:     EventSubmit,
		Text:     "first",
	})

	if !waitForSentTextContaining(t, ch, "provider returned HTML error body", 2*time.Second) {
		t.Fatalf("first turn never produced sanitized provider HTML error")
	}

	// Second inbound: must be accepted, NOT rejected with
	// "admission: still processing previous turn".
	ch.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "2",
		Kind:     EventSubmit,
		Text:     "Hi",
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, msg := range ch.sentSnapshot() {
			if strings.Contains(msg.Text, "still processing previous turn") {
				t.Fatalf("second submit wedged: %q (provider open-stream calls=%d)",
					msg.Text, provider.OpenStreamCalls())
			}
		}
		if provider.OpenStreamCalls() >= 2 {
			return // success: second submit reached the provider
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("second submit never reached provider; OpenStream calls=%d, sent=%v",
		provider.OpenStreamCalls(), ch.sentSnapshot())
}

// TestManager_KernelInbound_RepeatedProviderHTMLFailureNeverWedges mirrors the
// production transcript exactly: the provider keeps returning HTML 403 on
// every call, and the user sends three follow-up messages. Each follow-up
// must reach the provider — none may be rejected with
// "admission: still processing previous turn". This guards against state that
// leaks between consecutive PhaseFailed dispatches.
func TestManager_KernelInbound_RepeatedProviderHTMLFailureNeverWedges(t *testing.T) {
	provider := &alwaysHTMLErrorProvider{
		htmlErr: errors.New("Forbidden: provider returned HTML error body"),
	}

	k := kernel.New(kernel.Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, provider, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	go func() { _ = k.Run(ctx) }()

	m := NewManager(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, k, slog.Default())

	ch := newChannelOnlyFake("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatalf("register: %v", err)
	}
	go func() { _ = m.Run(ctx) }()

	// Four submits: the failing one plus three "Hi" follow-ups from the user
	// transcript. Each must reach OpenStream; none may be rejected by the
	// admission guard.
	for i, text := range []string{"trigger", "Hi", "Hi", "Hi"} {
		ch.pushInbound(InboundEvent{
			Platform: "telegram",
			ChatID:   "42",
			MsgID:    string(rune('a' + i)),
			Kind:     EventSubmit,
			Text:     text,
		})

		want := i + 1
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) && provider.OpenStreamCalls() < want {
			for _, msg := range ch.sentSnapshot() {
				if strings.Contains(msg.Text, "still processing previous turn") {
					t.Fatalf("submit %d wedged with admission error: %q (OpenStream calls=%d)",
						i, msg.Text, provider.OpenStreamCalls())
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
		if provider.OpenStreamCalls() < want {
			t.Fatalf("submit %d (%q) never reached provider; OpenStream calls=%d, sent=%v",
				i, text, provider.OpenStreamCalls(), ch.sentSnapshot())
		}
	}
}

type alwaysHTMLErrorProvider struct {
	mu      sync.Mutex
	htmlErr error
	calls   int
}

func (p *alwaysHTMLErrorProvider) OpenStream(context.Context, hermes.ChatRequest) (hermes.Stream, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	return nil, p.htmlErr
}

func (p *alwaysHTMLErrorProvider) OpenStreamCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func (p *alwaysHTMLErrorProvider) OpenRunEvents(context.Context, string) (hermes.RunEventStream, error) {
	return nil, hermes.ErrRunEventsNotSupported
}

func (p *alwaysHTMLErrorProvider) Health(context.Context) error { return nil }

func waitForSentTextContaining(t *testing.T, ch *channelOnlyFake, needle string, d time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		for _, msg := range ch.sentSnapshot() {
			if strings.Contains(msg.Text, needle) {
				return true
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

type htmlErrorThenSuccessProvider struct {
	mu      sync.Mutex
	htmlErr error
	calls   int
}

func (p *htmlErrorThenSuccessProvider) OpenStream(context.Context, hermes.ChatRequest) (hermes.Stream, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.calls == 1 {
		return nil, p.htmlErr
	}
	return &integrationSingleTurnStream{events: []hermes.Event{
		{Kind: hermes.EventToken, Token: "ok", TokensOut: 1},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensOut: 1},
	}}, nil
}

func (p *htmlErrorThenSuccessProvider) OpenStreamCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func (p *htmlErrorThenSuccessProvider) OpenRunEvents(context.Context, string) (hermes.RunEventStream, error) {
	return nil, hermes.ErrRunEventsNotSupported
}

func (p *htmlErrorThenSuccessProvider) Health(context.Context) error { return nil }

type integrationSingleTurnStream struct {
	mu     sync.Mutex
	events []hermes.Event
	pos    int
}

func (s *integrationSingleTurnStream) Recv(context.Context) (hermes.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pos >= len(s.events) {
		return hermes.Event{}, io.EOF
	}
	ev := s.events[s.pos]
	s.pos++
	return ev, nil
}

func (s *integrationSingleTurnStream) SessionID() string { return "" }
func (s *integrationSingleTurnStream) Close() error      { return nil }
