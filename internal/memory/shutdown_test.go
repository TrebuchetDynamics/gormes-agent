package memory

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
)

func TestClose_DrainsQueue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 4096, nil)

	const N = 100
	for i := 0; i < N; i++ {
		payload, _ := json.Marshal(map[string]any{
			"session_id": "s", "content": "msg", "ts_unix": int64(i),
		})
		_, _ = s.Exec(context.Background(), store.Command{
			Kind: store.AppendUserTurn, Payload: payload,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// DB is closed inside Close; reopen to verify.
	s2, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close(context.Background())

	var n int
	if err := s2.db.QueryRow("SELECT COUNT(*) FROM turns").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != N {
		t.Errorf("persisted turns = %d, want %d", n, N)
	}
}

func TestClose_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)

	ctx := context.Background()
	if err := s.Close(ctx); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := s.Close(ctx); err != nil {
		t.Errorf("second Close should be no-op, got %v", err)
	}
}

func TestClose_HonorsDeadline(t *testing.T) {
	// With an already-expired context, Close should return promptly even
	// if the worker has not finished draining. The DB still gets closed
	// so WAL checkpoint runs; no panic, no goroutine leak.
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 4096, nil)

	// Pre-fill the queue.
	for i := 0; i < 500; i++ {
		payload, _ := json.Marshal(map[string]any{
			"session_id": "s", "content": "x", "ts_unix": int64(i),
		})
		_, _ = s.Exec(context.Background(), store.Command{
			Kind: store.AppendUserTurn, Payload: payload,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.Close(ctx) }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close on tiny deadline returned err = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close hung despite ctx deadline")
	}
}
