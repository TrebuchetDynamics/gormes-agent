package kernel

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestKernel_StreamDropDiagnosticsForOpenStreamRetry(t *testing.T) {
	var logs bytes.Buffer
	client := &streamDropDiagnosticsClient{
		status: hermes.ProviderStatus{Provider: "openrouter", Runtime: "chat_completions"},
		openErrs: []error{&hermes.HTTPError{
			Status:     http.StatusServiceUnavailable,
			Body:       "provider stream connection lost",
			RetryAfter: 50 * time.Millisecond,
		}},
		streams: []hermes.Stream{&streamDropDiagnosticsStream{events: []hermes.Event{
			{Kind: hermes.EventToken, Token: "recovered"},
			{Kind: hermes.EventDone, FinishReason: "stop"},
		}}},
	}
	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "https://openrouter.ai/api/v1",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, client, store.NewNoop(), telemetry.New(), slog.New(slog.NewTextHandler(&logs, nil)))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)

	initial := <-k.Render()
	if initial.Phase != PhaseIdle {
		t.Fatalf("initial phase = %v, want Idle", initial.Phase)
	}
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "recover"}); err != nil {
		t.Fatal(err)
	}

	reconnecting := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseReconnecting
	}, time.Second)
	for _, want := range []string{"openrouter", "stream drop", "HTTPError", "retry 1/5"} {
		if !strings.Contains(reconnecting.StatusText, want) {
			t.Fatalf("reconnecting status = %q, want %q", reconnecting.StatusText, want)
		}
	}

	_, final := drainUntilIdle(t, k.Render(), initial.Seq, time.Second)
	if len(final.History) == 0 || final.History[len(final.History)-1].Content != "recovered" {
		t.Fatalf("final history = %#v, want recovered assistant message", final.History)
	}
	if strings.Contains(strings.ToLower(reconnecting.StatusText), "reconnected") {
		t.Fatalf("stream drop status invented a separate reconnected message: %q", reconnecting.StatusText)
	}
	logText := logs.String()
	for _, want := range []string{
		"kernel stream drop retry",
		"provider=openrouter",
		"endpoint=https://openrouter.ai/api/v1",
		"attempt=1",
		"max_attempts=5",
		"error_type=HTTPError",
		"error_class=retryable",
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("stream-drop log missing %q in:\n%s", want, logText)
		}
	}
}

func TestKernel_StreamDropDiagnosticsForMidStreamRetry(t *testing.T) {
	var logs bytes.Buffer
	client := &streamDropDiagnosticsClient{
		status: hermes.ProviderStatus{Provider: "anthropic", Runtime: "messages"},
		streams: []hermes.Stream{
			&streamDropDiagnosticsStream{
				events: []hermes.Event{{Kind: hermes.EventToken, Token: "partial"}},
				err: &hermes.HTTPError{
					Status:     http.StatusServiceUnavailable,
					Body:       "stream reset",
					RetryAfter: 50 * time.Millisecond,
				},
			},
			&streamDropDiagnosticsStream{events: []hermes.Event{
				{Kind: hermes.EventToken, Token: "fresh answer"},
				{Kind: hermes.EventDone, FinishReason: "stop"},
			}},
		},
	}
	k := New(Config{
		Model:     "claude-fixture",
		Endpoint:  "https://api.anthropic.com",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, client, store.NewNoop(), telemetry.New(), slog.New(slog.NewTextHandler(&logs, nil)))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)

	initial := <-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "recover midstream"}); err != nil {
		t.Fatal(err)
	}

	reconnecting := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseReconnecting
	}, time.Second)
	for _, want := range []string{"anthropic", "stream drop", "HTTPError", "retry 1/5"} {
		if !strings.Contains(reconnecting.StatusText, want) {
			t.Fatalf("mid-stream reconnecting status = %q, want %q", reconnecting.StatusText, want)
		}
	}
	logText := logs.String()
	if !strings.Contains(logText, "mid_stream=true") || !strings.Contains(logText, "provider=anthropic") {
		t.Fatalf("mid-stream log missing structured provider/mid_stream fields:\n%s", logText)
	}

	_, final := drainUntilIdle(t, k.Render(), initial.Seq, time.Second)
	if len(final.History) == 0 || final.History[len(final.History)-1].Content != "fresh answer" {
		t.Fatalf("final history = %#v, want fresh retry answer", final.History)
	}
}

type streamDropDiagnosticsClient struct {
	mu       sync.Mutex
	status   hermes.ProviderStatus
	openErrs []error
	streams  []hermes.Stream
	calls    int
}

func (c *streamDropDiagnosticsClient) OpenStream(context.Context, hermes.ChatRequest) (hermes.Stream, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if len(c.openErrs) > 0 {
		err := c.openErrs[0]
		c.openErrs = c.openErrs[1:]
		return nil, err
	}
	if len(c.streams) == 0 {
		return &streamDropDiagnosticsStream{}, nil
	}
	stream := c.streams[0]
	c.streams = c.streams[1:]
	return stream, nil
}

func (c *streamDropDiagnosticsClient) OpenRunEvents(context.Context, string) (hermes.RunEventStream, error) {
	return nil, hermes.ErrRunEventsNotSupported
}

func (c *streamDropDiagnosticsClient) Health(context.Context) error { return nil }

func (c *streamDropDiagnosticsClient) ProviderStatus() hermes.ProviderStatus {
	return c.status
}

type streamDropDiagnosticsStream struct {
	events []hermes.Event
	err    error
	pos    int
}

func (s *streamDropDiagnosticsStream) Recv(context.Context) (hermes.Event, error) {
	if s.pos < len(s.events) {
		ev := s.events[s.pos]
		s.pos++
		return ev, nil
	}
	if s.err != nil {
		err := s.err
		s.err = nil
		return hermes.Event{}, err
	}
	return hermes.Event{}, io.EOF
}

func (s *streamDropDiagnosticsStream) SessionID() string { return "" }
func (s *streamDropDiagnosticsStream) Close() error      { return nil }
