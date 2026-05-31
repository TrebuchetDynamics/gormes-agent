package gateway

import (
	"context"
	"testing"
	"time"
)

func TestCoalescer_FinalDeliveryClearsStalePendingPreview(t *testing.T) {
	ch := newFakeChannel("test")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newCoalescer(ch, 5*time.Millisecond, "chat1")
	c.flushImmediate(ctx, "preview")
	c.setPending("stale preview")
	c.flushImmediateFinal(ctx, "final", true)
	go c.run(ctx)

	time.Sleep(30 * time.Millisecond)
	edits := ch.editsSnapshot()
	for _, edit := range edits {
		if edit.Text == "stale preview" {
			t.Fatalf("stale pending preview edited after final delivery; edits=%#v", edits)
		}
	}
}

func TestCoalescer_DoesNotDropPendingTextSetDuringEdit(t *testing.T) {
	ch := &blockingEditChannel{fakeChannel: newFakeChannel("test")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newCoalescer(ch, time.Millisecond, "chat1")
	c.flushImmediate(ctx, "initial")
	ch.blockText = "old"
	go c.run(ctx)

	c.setPending("old")
	select {
	case <-ch.entered:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocked edit")
	}

	c.setPending("new")
	close(ch.release)

	waitFor(t, time.Second, func() bool {
		edits := ch.editsSnapshot()
		return len(edits) >= 3 && edits[len(edits)-1].Text == "new"
	})
}

type blockingEditChannel struct {
	*fakeChannel
	blockText string
	entered   chan struct{}
	release   chan struct{}
}

func (b *blockingEditChannel) EditMessage(ctx context.Context, chatID, msgID, text string) error {
	if b.entered == nil {
		b.entered = make(chan struct{})
	}
	if b.release == nil {
		b.release = make(chan struct{})
	}
	if text == b.blockText {
		select {
		case <-b.entered:
		default:
			close(b.entered)
		}
		select {
		case <-b.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return b.fakeChannel.EditMessage(ctx, chatID, msgID, text)
}
