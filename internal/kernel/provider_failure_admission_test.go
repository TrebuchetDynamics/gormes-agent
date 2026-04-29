package kernel

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestKernel_SubmitAcceptedAfterProviderOpenStreamFailure(t *testing.T) {
	client := &providerFailureThenSuccessClient{err: errors.New("Forbidden: provider returned HTML error body")}
	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, client, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render() // initial idle

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "first"}); err != nil {
		t.Fatal(err)
	}
	failed := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return strings.Contains(f.LastError, "provider returned HTML error body")
	}, time.Second)
	if !strings.Contains(failed.LastError, "provider returned HTML error body") {
		t.Fatalf("first failure LastError = %q, want provider error", failed.LastError)
	}
	if failed.Phase != PhaseFailed {
		t.Fatalf("first failure phase = %s, want failed terminal status before next submit", failed.Phase)
	}

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "second"}); err != nil {
		t.Fatal(err)
	}
	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && len(f.History) > 0 && f.History[len(f.History)-1].Role == "assistant"
	}, 2*time.Second)
	if strings.Contains(final.LastError, "still processing") {
		t.Fatalf("LastError = %q, want second submit accepted after provider failure", final.LastError)
	}
	if got := client.OpenStreamCalls(); got != 2 {
		t.Fatalf("OpenStream calls = %d, want 2; second submit should reach provider", got)
	}
}

type providerFailureThenSuccessClient struct {
	mu    sync.Mutex
	err   error
	calls int
}

func (c *providerFailureThenSuccessClient) OpenStream(context.Context, hermes.ChatRequest) (hermes.Stream, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.calls == 1 {
		return nil, c.err
	}
	return &singleTurnStream{events: []hermes.Event{
		{Kind: hermes.EventToken, Token: "ok", TokensOut: 1},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensOut: 1},
	}}, nil
}

func (c *providerFailureThenSuccessClient) OpenStreamCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func (c *providerFailureThenSuccessClient) OpenRunEvents(context.Context, string) (hermes.RunEventStream, error) {
	return nil, hermes.ErrRunEventsNotSupported
}

func (c *providerFailureThenSuccessClient) Health(context.Context) error { return nil }

type singleTurnStream struct {
	mu     sync.Mutex
	events []hermes.Event
	pos    int
}

func (s *singleTurnStream) Recv(context.Context) (hermes.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pos >= len(s.events) {
		return hermes.Event{}, io.EOF
	}
	ev := s.events[s.pos]
	s.pos++
	return ev, nil
}

func (s *singleTurnStream) SessionID() string { return "" }
func (s *singleTurnStream) Close() error      { return nil }
