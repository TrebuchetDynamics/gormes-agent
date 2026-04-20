package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/store"
)

func TestOpenSqlite_CreatesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer s.Close(context.Background())

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM turns").Scan(&n); err != nil {
		t.Errorf("turns table missing: %v", err)
	}
	if n != 0 {
		t.Errorf("turns count at startup = %d, want 0", n)
	}

	if err := s.db.QueryRow("SELECT COUNT(*) FROM turns_fts").Scan(&n); err != nil {
		t.Errorf("turns_fts virtual table missing: %v", err)
	}
}

func TestOpenSqlite_SchemaMetaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	var v string
	err := s.db.QueryRow("SELECT v FROM schema_meta WHERE k = 'version'").Scan(&v)
	if err != nil {
		t.Fatalf("schema_meta missing: %v", err)
	}
	if v != "3a" {
		t.Errorf("schema version = %q, want %q", v, "3a")
	}
}

func TestOpenSqlite_AutoCreatesParentDir(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "newsubdir")
	path := filepath.Join(parent, "memory.db")
	s, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite (missing parent dir): %v", err)
	}
	defer s.Close(context.Background())

	info, err := os.Stat(parent)
	if err != nil {
		t.Fatalf("parent dir should exist: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("parent dir perm = %o, want 0700", perm)
	}
}

func TestOpenSqlite_SetsWALMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	var mode string
	if err := s.db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func TestSqliteStore_ExecReturnsFast(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	start := time.Now()
	_, err := s.Exec(context.Background(), store.Command{
		Kind:    store.AppendUserTurn,
		Payload: json.RawMessage(`{"session_id":"s","content":"hi","ts_unix":1}`),
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	// 10 ms is generous — real return should be sub-ms. Under the race
	// detector this still has headroom.
	if elapsed > 10*time.Millisecond {
		t.Errorf("Exec took %v, want well under 10 ms", elapsed)
	}
}

func TestSqliteStore_ExecDropsOnFullQueue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")

	// With Task 3's silent-drain run(), commands that land in the queue
	// are drained without writes. With queueCap=2 the worker can only
	// buffer 2 at a time; firing 1000 Execs back-to-back MUST overflow
	// the queue. Drops > 0 and Drops + Accepted == 1000.

	s, _ := OpenSqlite(path, 2, nil)
	defer s.Close(context.Background())

	for i := 0; i < 1000; i++ {
		_, _ = s.Exec(context.Background(), store.Command{
			Kind:    store.AppendUserTurn,
			Payload: json.RawMessage(`{}`),
		})
	}

	st := s.Stats()
	if st.Drops == 0 {
		t.Errorf("expected some Drops after 1000 Execs into queueCap=2, got 0")
	}
	if st.Drops+st.Accepted != 1000 {
		t.Errorf("Accepted (%d) + Drops (%d) = %d, want 1000",
			st.Accepted, st.Drops, st.Drops+st.Accepted)
	}
}

func TestSqliteStore_ExecHonorsCtxCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.Exec(ctx, store.Command{Kind: store.AppendUserTurn})
	if err == nil {
		t.Error("Exec with canceled ctx should return ctx.Err()")
	}
}
