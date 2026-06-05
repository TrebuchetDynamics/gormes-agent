package kernel

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestKernel_OpenStreamContextCanceledDoesNotRetry(t *testing.T) {
	client := &openCancelClient{}
	k := New(Config{
		Model:                "hermes-agent",
		Endpoint:             "http://mock",
		Admission:            Admission{MaxBytes: 200_000, MaxLines: 10_000},
		MaxReconnectDuration: 10 * time.Second,
	}, client, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)

	initial := <-k.Render()
	if initial.Phase != PhaseIdle {
		t.Fatalf("initial phase = %v, want idle", initial.Phase)
	}
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hello"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	failed := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseFailed
	}, 2*time.Second)
	if client.calls != 1 {
		t.Fatalf("OpenStream calls = %d, want 1 (no reconnect loop for local cancellation)", client.calls)
	}
	if strings.Contains(strings.ToLower(failed.StatusText), "reconnect") || failed.RetryStatus.AttemptsUsed != 0 {
		t.Fatalf("cancelled open stream was rendered as retry: status=%q retry=%+v", failed.StatusText, failed.RetryStatus)
	}
	if !strings.Contains(failed.LastError, "context canceled") {
		t.Fatalf("LastError = %q, want context canceled evidence", failed.LastError)
	}
}

func TestKernel_MidStreamContextCanceledDoesNotRetry(t *testing.T) {
	client := &singleStreamClient{stream: &cancelStream{err: context.Canceled}}
	k := New(Config{
		Model:                "hermes-agent",
		Endpoint:             "http://mock",
		Admission:            Admission{MaxBytes: 200_000, MaxLines: 10_000},
		MaxReconnectDuration: 10 * time.Second,
	}, client, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)

	initial := <-k.Render()
	if initial.Phase != PhaseIdle {
		t.Fatalf("initial phase = %v, want idle", initial.Phase)
	}
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hello"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	failed := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseFailed
	}, 2*time.Second)
	if client.calls != 1 {
		t.Fatalf("OpenStream calls = %d, want 1 (no reconnect loop for local cancellation)", client.calls)
	}
	if strings.Contains(strings.ToLower(failed.StatusText), "reconnect") || failed.RetryStatus.AttemptsUsed != 0 {
		t.Fatalf("cancelled stream was rendered as retry: status=%q retry=%+v", failed.StatusText, failed.RetryStatus)
	}
	if !strings.Contains(failed.LastError, "context canceled") {
		t.Fatalf("LastError = %q, want context canceled evidence", failed.LastError)
	}
}

type openCancelClient struct{ calls int }

func (c *openCancelClient) ProviderStatus() llm.ProviderStatus {
	return llm.ProviderStatus{Provider: "mock", Runtime: "test"}
}

func (c *openCancelClient) Health(context.Context) error { return nil }

func (c *openCancelClient) OpenStream(context.Context, llm.ChatRequest) (llm.Stream, error) {
	c.calls++
	return nil, context.Canceled
}

func (c *openCancelClient) OpenRunEvents(context.Context, string) (llm.RunEventStream, error) {
	return nil, llm.ErrRunEventsNotSupported
}

type singleStreamClient struct {
	stream llm.Stream
	calls  int
}

func (c *singleStreamClient) ProviderStatus() llm.ProviderStatus {
	return llm.ProviderStatus{Provider: "mock", Runtime: "test"}
}

func (c *singleStreamClient) Health(context.Context) error { return nil }

func (c *singleStreamClient) OpenStream(context.Context, llm.ChatRequest) (llm.Stream, error) {
	c.calls++
	return c.stream, nil
}

func (c *singleStreamClient) OpenRunEvents(context.Context, string) (llm.RunEventStream, error) {
	return nil, llm.ErrRunEventsNotSupported
}

type cancelStream struct{ err error }

func (s *cancelStream) Recv(context.Context) (llm.Event, error) {
	if s.err == nil {
		return llm.Event{}, io.EOF
	}
	return llm.Event{}, s.err
}

func (s *cancelStream) SessionID() string { return "" }
func (s *cancelStream) Close() error      { return nil }
