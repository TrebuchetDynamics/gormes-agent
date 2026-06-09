package coalescing

import (
	"context"
	"testing"
	"time"
)

type contextIgnoringSender struct {
	placeholderCalls int
	editCalls        int
}

func (s *contextIgnoringSender) SendPlaceholder(context.Context, string) (string, error) {
	s.placeholderCalls++
	return "msg-1", nil
}

func (s *contextIgnoringSender) EditMessage(context.Context, string, string, string) error {
	s.editCalls++
	return nil
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
