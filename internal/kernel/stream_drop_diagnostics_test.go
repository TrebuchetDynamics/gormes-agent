package kernel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestKernel_StreamDropDiagnosticsForOpenStreamRetry(t *testing.T) {
	var logs bytes.Buffer
	client := &streamDropDiagnosticsClient{
		status: llm.ProviderStatus{Provider: "openrouter", Runtime: "chat_completions"},
		openErrs: []error{&llm.HTTPError{
			Status:     http.StatusServiceUnavailable,
			Body:       "provider stream connection lost",
			RetryAfter: 50 * time.Millisecond,
			Headers: map[string]string{
				"cf-ray":                "8f1a2b3c4d5e6f7g-LAX",
				"x-openrouter-provider": "Anthropic",
				"authorization":         "Bearer should-not-leak",
			},
		}},
		streams: []llm.Stream{&streamDropDiagnosticsStream{events: []llm.Event{
			{Kind: llm.EventToken, Token: "recovered"},
			{Kind: llm.EventDone, FinishReason: "stop"},
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
		"http_status=503",
		"upstream_headers=\"cf-ray=8f1a2b3c4d5e6f7g-LAX x-openrouter-provider=Anthropic\"",
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("stream-drop log missing %q in:\n%s", want, logText)
		}
	}
	if strings.Contains(logText, "authorization") || strings.Contains(logText, "should-not-leak") {
		t.Fatalf("stream-drop log leaked non-allowlisted header:\n%s", logText)
	}
}

func TestKernel_StreamDropDiagnosticsForMidStreamRetry(t *testing.T) {
	var logs bytes.Buffer
	wrappedErr := fmt.Errorf("provider wrapper: %w", &llm.HTTPError{
		Status:     http.StatusServiceUnavailable,
		Body:       "stream reset",
		RetryAfter: 50 * time.Millisecond,
	})
	client := &streamDropDiagnosticsClient{
		status: llm.ProviderStatus{Provider: "anthropic", Runtime: "messages"},
		streams: []llm.Stream{
			&streamDropDiagnosticsStream{
				events: []llm.Event{{Kind: llm.EventToken, Token: "partial"}},
				err:    wrappedErr,
				diag: llm.StreamDiagnostics{
					HTTPStatus:      http.StatusOK,
					Headers:         map[string]string{"x-request-id": "req-midstream", "cookie": "secret-cookie"},
					Bytes:           4096,
					Chunks:          3,
					Elapsed:         8 * time.Second,
					TimeToFirstByte: 400 * time.Millisecond,
				},
			},
			&streamDropDiagnosticsStream{events: []llm.Event{
				{Kind: llm.EventToken, Token: "fresh answer"},
				{Kind: llm.EventDone, FinishReason: "stop"},
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
	for _, want := range []string{"anthropic", "stream drop", "HTTPError", "after 8.0s", "retry 1/5"} {
		if !strings.Contains(reconnecting.StatusText, want) {
			t.Fatalf("mid-stream reconnecting status = %q, want %q", reconnecting.StatusText, want)
		}
	}
	logText := logs.String()
	for _, want := range []string{
		"mid_stream=true",
		"provider=anthropic",
		"http_status=200",
		"bytes=4096",
		"chunks=3",
		"elapsed=8s",
		"ttfb=400ms",
		"upstream_headers=\"x-request-id=req-midstream\"",
		"error_chain=\"wrapError(provider wrapper: Service Unavailable: stream reset) <- HTTPError(Service Unavailable: stream reset)\"",
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("mid-stream log missing %q in:\n%s", want, logText)
		}
	}
	if strings.Contains(logText, "cookie") || strings.Contains(logText, "secret-cookie") {
		t.Fatalf("mid-stream log leaked non-allowlisted header:\n%s", logText)
	}

	_, final := drainUntilIdle(t, k.Render(), initial.Seq, time.Second)
	if len(final.History) == 0 || final.History[len(final.History)-1].Content != "fresh answer" {
		t.Fatalf("final history = %#v, want fresh retry answer", final.History)
	}
}

type streamDropDiagnosticsClient struct {
	mu       sync.Mutex
	status   llm.ProviderStatus
	openErrs []error
	streams  []llm.Stream
	calls    int
}

func (c *streamDropDiagnosticsClient) OpenStream(context.Context, llm.ChatRequest) (llm.Stream, error) {
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

func (c *streamDropDiagnosticsClient) OpenRunEvents(context.Context, string) (llm.RunEventStream, error) {
	return nil, llm.ErrRunEventsNotSupported
}

func (c *streamDropDiagnosticsClient) Health(context.Context) error { return nil }

func (c *streamDropDiagnosticsClient) ProviderStatus() llm.ProviderStatus {
	return c.status
}

type streamDropDiagnosticsStream struct {
	events []llm.Event
	err    error
	diag   llm.StreamDiagnostics
	pos    int
}

func (s *streamDropDiagnosticsStream) Recv(context.Context) (llm.Event, error) {
	if s.pos < len(s.events) {
		ev := s.events[s.pos]
		s.pos++
		return ev, nil
	}
	if s.err != nil {
		err := s.err
		s.err = nil
		return llm.Event{}, err
	}
	return llm.Event{}, io.EOF
}

func (s *streamDropDiagnosticsStream) SessionID() string { return "" }
func (s *streamDropDiagnosticsStream) Close() error      { return nil }

func (s *streamDropDiagnosticsStream) StreamDiagnostics() llm.StreamDiagnostics {
	return s.diag
}
