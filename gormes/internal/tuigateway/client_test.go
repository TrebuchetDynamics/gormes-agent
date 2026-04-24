package tuigateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/kernel"
)

// TestRemoteClient_FramesStreamsFromHandler proves a RemoteClient composed
// around NewSSEHandler yields decoded kernel.RenderFrames over its Frames
// channel. This is the client-side seat that cmd/gormes --remote will fill.
func TestRemoteClient_FramesStreamsFromHandler(t *testing.T) {
	t.Parallel()
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Seq: 7, Phase: kernel.PhaseStreaming, DraftText: "hi"}
	close(frames)

	srv := httptest.NewServer(NewSSEHandler(frames))
	defer srv.Close()

	client := &RemoteClient{FramesURL: srv.URL, EventsURL: srv.URL}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := client.Frames(ctx)
	if err != nil {
		t.Fatalf("Frames: %v", err)
	}
	select {
	case f, ok := <-out:
		if !ok {
			t.Fatal("channel closed before frame arrived")
		}
		if f.Seq != 7 || f.DraftText != "hi" {
			t.Fatalf("frame = %+v", f)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for frame")
	}
}

// TestRemoteClient_SubmitReachesSink proves Submit serialises a
// PlatformEventSubmit with the caller-provided text and reaches the sink
// registered behind NewEventHandler.
func TestRemoteClient_SubmitReachesSink(t *testing.T) {
	t.Parallel()
	got := make(chan kernel.PlatformEvent, 1)
	sink := func(e kernel.PlatformEvent) error {
		got <- e
		return nil
	}
	srv := httptest.NewServer(NewEventHandler(sink))
	defer srv.Close()

	client := &RemoteClient{FramesURL: srv.URL, EventsURL: srv.URL}
	if err := client.Submit(context.Background(), "hello remote"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	select {
	case e := <-got:
		if e.Kind != kernel.PlatformEventSubmit || e.Text != "hello remote" {
			t.Fatalf("sink got %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("sink did not receive submit")
	}
}

// TestRemoteClient_CancelReachesSink proves Cancel forwards
// PlatformEventCancel without populating Text or any other metadata.
func TestRemoteClient_CancelReachesSink(t *testing.T) {
	t.Parallel()
	got := make(chan kernel.PlatformEvent, 1)
	sink := func(e kernel.PlatformEvent) error { got <- e; return nil }
	srv := httptest.NewServer(NewEventHandler(sink))
	defer srv.Close()

	client := &RemoteClient{FramesURL: srv.URL, EventsURL: srv.URL}
	if err := client.Cancel(context.Background()); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	select {
	case e := <-got:
		if e.Kind != kernel.PlatformEventCancel {
			t.Fatalf("Kind = %v, want PlatformEventCancel", e.Kind)
		}
		if e.Text != "" {
			t.Fatalf("Text = %q, want empty", e.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("sink did not receive cancel")
	}
}

// TestRemoteClient_ResetReachesSink proves Reset forwards
// PlatformEventResetSession so a remote "new chat" button can round-trip.
func TestRemoteClient_ResetReachesSink(t *testing.T) {
	t.Parallel()
	got := make(chan kernel.PlatformEvent, 1)
	sink := func(e kernel.PlatformEvent) error { got <- e; return nil }
	srv := httptest.NewServer(NewEventHandler(sink))
	defer srv.Close()

	client := &RemoteClient{FramesURL: srv.URL, EventsURL: srv.URL}
	if err := client.Reset(context.Background()); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	select {
	case e := <-got:
		if e.Kind != kernel.PlatformEventResetSession {
			t.Fatalf("Kind = %v, want PlatformEventResetSession", e.Kind)
		}
	case <-time.After(time.Second):
		t.Fatal("sink did not receive reset")
	}
}

// TestRemoteClient_SubmitSurfacesSinkBackpressure proves Submit returns a
// non-nil error when the server replies 502 (sink reported ErrEventMailboxFull),
// so the Bubble Tea side can retry rather than silently desync.
func TestRemoteClient_SubmitSurfacesSinkBackpressure(t *testing.T) {
	t.Parallel()
	var calls int32
	sink := func(kernel.PlatformEvent) error {
		atomic.AddInt32(&calls, 1)
		return kernel.ErrEventMailboxFull
	}
	srv := httptest.NewServer(NewEventHandler(sink))
	defer srv.Close()

	client := &RemoteClient{FramesURL: srv.URL, EventsURL: srv.URL}
	err := client.Submit(context.Background(), "nope")
	if err == nil {
		t.Fatal("Submit: want non-nil error on sink backpressure")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("sink invoked %d times; want 1", got)
	}
}

// TestRemoteClient_FramesErrorsOnBadHandshake proves a non-200 SSE handshake
// propagates as a synchronous error so the --remote startup path can bail
// loudly instead of hanging on an empty channel.
func TestRemoteClient_FramesErrorsOnBadHandshake(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := &RemoteClient{FramesURL: srv.URL, EventsURL: srv.URL}
	if _, err := client.Frames(context.Background()); err == nil {
		t.Fatal("Frames: want non-nil error on 500 handshake")
	}
}
