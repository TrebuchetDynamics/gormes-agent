package coalescing

import (
	"context"
	"testing"
	"time"
)

type contextCheckingSender struct {
	placeholderCalls int
	editCalls        int
}

type contextIgnoringSender struct {
	placeholderCalls int
	editCalls        int
}

type cancelingInitialTextSender struct {
	placeholderCalls int
	editCalls        int
	sendCalls        int
}

type placeholderThenFailingEditSender struct {
	placeholderCalls int
	editCalls        int
	failFirstEdit    bool
}

func (s *placeholderThenFailingEditSender) SendPlaceholder(context.Context, string) (string, error) {
	s.placeholderCalls++
	return "msg-1", nil
}

func (s *placeholderThenFailingEditSender) EditMessage(context.Context, string, string, string) error {
	s.editCalls++
	if s.failFirstEdit && s.editCalls == 1 {
		return context.DeadlineExceeded
	}
	return nil
}

func (s *cancelingInitialTextSender) Send(context.Context, string, string) (string, error) {
	s.sendCalls++
	return "", context.Canceled
}

func (s *cancelingInitialTextSender) SendPlaceholder(context.Context, string) (string, error) {
	s.placeholderCalls++
	return "msg-1", nil
}

func (s *cancelingInitialTextSender) EditMessage(context.Context, string, string, string) error {
	s.editCalls++
	return nil
}

func (s *contextCheckingSender) SendPlaceholder(ctx context.Context, _ string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	s.placeholderCalls++
	return "msg-1", nil
}

func (s *contextCheckingSender) EditMessage(ctx context.Context, _, _, _ string) error {
	if ctx == nil {
		panic("nil context")
	}
	s.editCalls++
	return nil
}

func (s *contextIgnoringSender) SendPlaceholder(context.Context, string) (string, error) {
	s.placeholderCalls++
	return "msg-1", nil
}

func (s *contextIgnoringSender) EditMessage(context.Context, string, string, string) error {
	s.editCalls++
	return nil
}

func TestCoalescerFlushImmediateAllowsNilContext(t *testing.T) {
	sender := &contextCheckingSender{}
	c := New(sender, time.Second, "chat-1")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("FlushImmediate panicked with nil context: %v", r)
		}
	}()
	c.FlushImmediate(nil, "draft")

	if sender.placeholderCalls != 1 || sender.editCalls != 1 {
		t.Fatalf("delivery calls with nil context: placeholder=%d edit=%d, want 1/1", sender.placeholderCalls, sender.editCalls)
	}
}

func TestCoalescerRetainsPlaceholderAfterInitialEditFailure(t *testing.T) {
	sender := &placeholderThenFailingEditSender{failFirstEdit: true}
	c := New(sender, time.Second, "chat-1")

	c.FlushImmediate(context.Background(), "draft")
	if got := c.CurrentMessageID(); got != "msg-1" {
		t.Fatalf("CurrentMessageID after failed initial edit = %q, want retained placeholder msg-1", got)
	}

	c.FlushImmediate(context.Background(), "draft")
	if sender.placeholderCalls != 1 {
		t.Fatalf("placeholder calls = %d, want retry to edit existing placeholder", sender.placeholderCalls)
	}
	if sender.editCalls != 2 {
		t.Fatalf("edit calls = %d, want failed edit plus retry", sender.editCalls)
	}
}

func TestCoalescerInitialTextSendContextCanceledDoesNotFallbackToPlaceholder(t *testing.T) {
	sender := &cancelingInitialTextSender{}
	c := New(sender, time.Second, "chat-1", InitialTextSend())

	c.FlushImmediate(context.Background(), "draft")

	if sender.sendCalls != 1 {
		t.Fatalf("Send calls = %d, want one initial text send attempt", sender.sendCalls)
	}
	if sender.placeholderCalls != 0 || sender.editCalls != 0 {
		t.Fatalf("fallback delivery after canceled initial send: placeholder=%d edit=%d", sender.placeholderCalls, sender.editCalls)
	}
	if got := c.CurrentMessageID(); got != "" {
		t.Fatalf("CurrentMessageID = %q after canceled initial send, want empty", got)
	}
}

func TestCoalescerFinalFlushHonorsContextCanceledWhileWaitingForDeliveryLock(t *testing.T) {
	sender := &contextIgnoringSender{}
	c := New(sender, time.Second, "chat-1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.deliveryMu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.FlushImmediateFinal(ctx, "final", true)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	c.deliveryMu.Unlock()
	<-done

	if sender.placeholderCalls != 0 || sender.editCalls != 0 {
		t.Fatalf("delivery calls after context canceled while waiting: placeholder=%d edit=%d", sender.placeholderCalls, sender.editCalls)
	}
	if got := c.CurrentMessageID(); got != "" {
		t.Fatalf("CurrentMessageID = %q after canceled blocked final flush, want empty", got)
	}
}

func TestCoalescerFinalFlushHonorsCanceledContextBeforeDelivery(t *testing.T) {
	sender := &contextIgnoringSender{}
	c := New(sender, time.Second, "chat-1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c.FlushImmediateFinal(ctx, "final", true)

	if sender.placeholderCalls != 0 || sender.editCalls != 0 {
		t.Fatalf("delivery calls after canceled context: placeholder=%d edit=%d", sender.placeholderCalls, sender.editCalls)
	}
	if got := c.CurrentMessageID(); got != "" {
		t.Fatalf("CurrentMessageID = %q after canceled final flush, want empty", got)
	}
}

// TestCoalescerCursorStrippedFinalEditFailureNoSend verifies that when the
// final cursor-strip edit fails (Telegram flood control) but the last streamed
// text was "answer ▉" (the full answer plus cursor), the coalescer does NOT
// fall back to a plain Send — the full answer is already visible to the user,
// only the cursor would be stripped. Mirrors Hermes fix(gateway): suppress
// duplicate final stream sends (197337cc4).
func TestCoalescerCursorStrippedFinalEditFailureNoSend(t *testing.T) {
	editErr := context.DeadlineExceeded // simulate Telegram flood-control failure
	sender := &trackingCursorSender{editErr: editErr}
	c := New(sender, time.Second, "chat-1",
		StreamCursor(" ▉"),
	)

	// Simulate a streaming frame that was already sent with the cursor.
	c.mu.Lock()
	c.pendingMsgID = "msg-1"
	c.lastSentText = "The answer. ▉"
	c.mu.Unlock()

	// Final flush strips the cursor — text == "The answer." without cursor.
	c.FlushImmediateFinal(context.Background(), "The answer.", true)

	if sender.sendCalls > 0 {
		t.Fatalf("Send called %d times; expected 0 — answer was already visible with cursor", sender.sendCalls)
	}
}

type trackingCursorSender struct {
	editCalls int
	sendCalls int
	editErr   error
}

func (s *trackingCursorSender) SendPlaceholder(_ context.Context, _ string) (string, error) {
	return "msg-1", nil
}

func (s *trackingCursorSender) EditMessage(_ context.Context, _, _, _ string) error {
	s.editCalls++
	return s.editErr
}

func (s *trackingCursorSender) Send(_ context.Context, _ string, text string) (string, error) {
	s.sendCalls++
	return "msg-2", nil
}

func TestCoalescerNilSenderDoesNotPanicOnFinalFlush(t *testing.T) {
	var evidences []Evidence
	c := New(nil, time.Second, "chat-1", EvidenceSinkOption(func(ev Evidence) {
		evidences = append(evidences, ev)
	}))

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("FlushImmediateFinal panicked with nil sender: %v", r)
		}
		if len(evidences) != 1 || evidences[0].Code != "send_final_failed" {
			t.Fatalf("evidences = %+v, want one send_final_failed evidence", evidences)
		}
	}()

	c.FlushImmediateFinal(context.Background(), "final", true)
}
