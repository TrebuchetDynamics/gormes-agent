package batching

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestEventBufferMergesFlushesAndDrains(t *testing.T) {
	buf := NewEventBuffer(MergeTextBatch)
	inbox := make(chan gateway.InboundEvent, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf.Enqueue(ctx, inbox, "chat:1", gateway.InboundEvent{Text: "first"}, time.Hour)
	buf.Enqueue(ctx, inbox, "chat:1", gateway.InboundEvent{Text: "second"}, time.Hour)
	if got := buf.Len(); got != 1 {
		t.Fatalf("Len = %d, want one pending batch", got)
	}

	buf.CancelAndDrain(ctx)
	select {
	case ev := <-inbox:
		if ev.Text != "first\nsecond" {
			t.Fatalf("drained Text = %q, want merged batch", ev.Text)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for drained batch")
	}
	if got := buf.Len(); got != 0 {
		t.Fatalf("Len after drain = %d, want empty", got)
	}
}

func TestEventBufferCancelDropsPending(t *testing.T) {
	buf := NewEventBuffer(MergePhotoBatch)
	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf.Enqueue(ctx, inbox, "album:1", gateway.InboundEvent{Text: "caption"}, time.Hour)
	buf.Cancel()
	if got := buf.Len(); got != 0 {
		t.Fatalf("Len after cancel = %d, want empty", got)
	}
	select {
	case ev := <-inbox:
		t.Fatalf("Cancel emitted pending event: %+v", ev)
	case <-time.After(20 * time.Millisecond):
	}
}
