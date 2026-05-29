package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestNoopStore_AckFast(t *testing.T) {
	s := NewNoop()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ack, err := s.Exec(ctx, Command{Kind: AppendUserTurn, Payload: json.RawMessage(`{"text":"hi"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if ack.TurnID != 0 {
		t.Errorf("TurnID = %d, want 0 from NoopStore", ack.TurnID)
	}
}

func TestNoopStore_RespectsCancelledCtx(t *testing.T) {
	s := NewNoop()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	_, err := s.Exec(ctx, Command{Kind: AppendUserTurn})
	if err != context.Canceled {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestSlowStore_HitsDeadline(t *testing.T) {
	s := NewSlow(500 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := s.Exec(ctx, Command{Kind: AppendUserTurn})
	if err != context.DeadlineExceeded {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Errorf("Exec took %v; should abort near the 100ms deadline, not wait for the full 500ms delay", d)
	}
}

func TestSlowStore_AcksAfterDelay(t *testing.T) {
	s := NewSlow(20 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	ack, err := s.Exec(ctx, Command{Kind: AppendUserTurn})
	if err != nil {
		t.Fatal(err)
	}
	if ack.TurnID != 1 {
		t.Errorf("TurnID = %d, want 1 from SlowStore", ack.TurnID)
	}
}
