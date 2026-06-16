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

func TestCoalescer_FinalWaitsForInFlightInitialDelivery(t *testing.T) {
	ch := &blockingPlaceholderChannel{
		fakeChannel:      newFakeChannel("test"),
		entered:          make(chan struct{}),
		release:          make(chan struct{}),
		extraPlaceholder: make(chan struct{}, 1),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newCoalescer(ch, time.Millisecond, "chat1")
	go c.run(ctx)

	c.setPending("preview")
	select {
	case <-ch.entered:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocked initial placeholder")
	}

	done := make(chan struct{})
	go func() {
		c.flushImmediateFinal(ctx, "final", true)
		close(done)
	}()

	select {
	case <-ch.extraPlaceholder:
		t.Fatalf("final delivery sent a second placeholder while initial delivery was still in flight; sent=%#v", ch.sentSnapshot())
	case <-time.After(50 * time.Millisecond):
	}

	close(ch.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for final delivery")
	}

	if got := len(ch.placeholdersSnapshot()); got != 1 {
		t.Fatalf("placeholder sends = %d, want 1; sent=%#v edits=%#v", got, ch.sentSnapshot(), ch.editsSnapshot())
	}
	edits := ch.editsSnapshot()
	if len(edits) < 2 || edits[len(edits)-1].Text != "final" {
		t.Fatalf("final edit missing after in-flight initial delivery; edits=%#v", edits)
	}
}

type blockingEditChannel struct {
	*fakeChannel
	blockText string
	entered   chan struct{}
	release   chan struct{}
}

type blockingPlaceholderChannel struct {
	*fakeChannel
	entered          chan struct{}
	release          chan struct{}
	extraPlaceholder chan struct{}
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

func (b *blockingPlaceholderChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if b.entered == nil {
		b.entered = make(chan struct{})
	}
	if b.release == nil {
		b.release = make(chan struct{})
	}
	if b.extraPlaceholder == nil {
		b.extraPlaceholder = make(chan struct{}, 1)
	}

	b.mu.Lock()
	call := len(b.placeholders) + 1
	b.mu.Unlock()
	if call == 1 {
		select {
		case <-b.entered:
		default:
			close(b.entered)
		}
		select {
		case <-b.release:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	} else {
		select {
		case b.extraPlaceholder <- struct{}{}:
		default:
		}
	}
	return b.fakeChannel.SendPlaceholder(ctx, chatID)
}

func (b *blockingPlaceholderChannel) placeholdersSnapshot() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneSlice(b.placeholders)
}
