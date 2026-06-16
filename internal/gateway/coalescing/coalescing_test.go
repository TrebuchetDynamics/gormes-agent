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
