package memory

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
)

func insertTurn(t *testing.T, s *SqliteStore, sid, content string) {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{
		"session_id": sid, "content": content, "ts_unix": time.Now().Unix(),
	})
	_, _ = s.Exec(context.Background(), store.Command{
		Kind: store.AppendUserTurn, Payload: payload,
	})
}

func waitForAccepted(t *testing.T, s *SqliteStore, want int, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if int(s.Stats().Accepted) >= want {
			// Also wait for the worker to flush all rows to the DB.
			var n int
			_ = s.db.QueryRow("SELECT COUNT(*) FROM turns").Scan(&n)
			if n >= want {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	var n int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM turns").Scan(&n)
	t.Fatalf("timeout waiting for turns >= %d; Accepted=%d turns=%d",
		want, s.Stats().Accepted, n)
}

func TestFTS5_MatchBasic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	insertTurn(t, s, "s1", "I remember the word asparagus")
	insertTurn(t, s, "s1", "and the capital of portugal is Lisbon")
	insertTurn(t, s, "s2", "banana")
	waitForAccepted(t, s, 3, 2*time.Second)

	rows, err := s.db.Query(`SELECT rowid FROM turns_fts WHERE turns_fts MATCH ?`, "asparagus")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	if len(ids) != 1 {
		t.Errorf("MATCH asparagus returned %d rows, want 1", len(ids))
	}
}

func TestFTS5_MatchPhrase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	insertTurn(t, s, "s1", "gormes telegram binary is awesome")
	insertTurn(t, s, "s1", "telegram works; gormes works")
	insertTurn(t, s, "s1", "unrelated message about coffee")
	waitForAccepted(t, s, 3, 2*time.Second)

	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM turns_fts WHERE turns_fts MATCH ?`,
		`"gormes telegram"`,
	).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf(`MATCH "gormes telegram" returned %d rows, want 1`, n)
	}
}

func TestFTS5_DeleteTriggerUpdatesIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	insertTurn(t, s, "s1", "ephemeral memory candidate")
	waitForAccepted(t, s, 1, 2*time.Second)

	var n int
	_ = s.db.QueryRow(
		`SELECT COUNT(*) FROM turns_fts WHERE turns_fts MATCH ?`, "ephemeral",
	).Scan(&n)
	if n != 1 {
		t.Fatalf("precondition: expected 1 FTS hit, got %d", n)
	}

	if _, err := s.db.Exec("DELETE FROM turns"); err != nil {
		t.Fatal(err)
	}

	_ = s.db.QueryRow(
		`SELECT COUNT(*) FROM turns_fts WHERE turns_fts MATCH ?`, "ephemeral",
	).Scan(&n)
	if n != 0 {
		t.Errorf("after DELETE, FTS hit count = %d, want 0", n)
	}
}
