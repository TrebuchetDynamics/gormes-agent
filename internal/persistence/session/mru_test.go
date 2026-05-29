package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestSessionDirectoryMRUResolvesMostRecentlyActiveBeyondNewestStartedRows(t *testing.T) {
	db := openSessionDirectoryTestDB(t)
	for i := 0; i < 25; i++ {
		id := fmt.Sprintf("new-start-%02d", i)
		seedSessionDirectoryTurn(t, db, id, "user", "newer started", int64(10_000+i), "")
	}
	seedSessionDirectoryTurn(t, db, "old-start-recently-active", "user", "old start", 100, "")
	seedSessionDirectoryTurn(t, db, "old-start-recently-active", "assistant", "recent activity", 20_000, "")

	got, err := ResolveMostRecentSession(context.Background(), db, "cli")
	if err != nil {
		t.Fatalf("ResolveMostRecentSession: %v", err)
	}
	if got != "old-start-recently-active" {
		t.Fatalf("most recent session = %q, want old-start-recently-active", got)
	}
}

func TestSessionDirectoryMRUFallsBackToStartedAtForLegacyRows(t *testing.T) {
	db := openSessionDirectoryTestDB(t)
	seedSessionDirectoryTurn(t, db, "older", "user", "older", 100, "")
	seedSessionDirectoryTurn(t, db, "newer", "user", "newer", 200, "")

	got, err := ResolveMostRecentSession(context.Background(), db, "cli")
	if err != nil {
		t.Fatalf("ResolveMostRecentSession: %v", err)
	}
	if got != "newer" {
		t.Fatalf("most recent session = %q, want newer", got)
	}
}

func TestSessionDirectoryResolvePrefix(t *testing.T) {
	db := openSessionDirectoryTestDB(t)
	seedSessionDirectoryTurn(t, db, "20260315_092437_c9a6ff", "user", "target", 100, "")
	seedSessionDirectoryTurn(t, db, "20260315_092500_other", "user", "other", 200, "")
	seedSessionDirectoryTurn(t, db, "20260315_092437_c9ffff", "user", "ambiguous", 300, "")

	got, err := ResolveSessionIDPrefix(context.Background(), db, "20260315_092437_c9a6")
	if err != nil {
		t.Fatalf("ResolveSessionIDPrefix unique: %v", err)
	}
	if got != "20260315_092437_c9a6ff" {
		t.Fatalf("resolved id = %q, want full target id", got)
	}

	if _, err := ResolveSessionIDPrefix(context.Background(), db, "missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("missing prefix err = %v, want ErrSessionNotFound", err)
	}
	if _, err := ResolveSessionIDPrefix(context.Background(), db, "20260315_092437_c9"); !errors.Is(err, ErrSessionPrefixAmbiguous) {
		t.Fatalf("ambiguous prefix err = %v, want ErrSessionPrefixAmbiguous", err)
	}
}

func openSessionDirectoryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE turns (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		ts_unix INTEGER NOT NULL,
		chat_id TEXT NOT NULL DEFAULT '',
		meta_json TEXT
	)`); err != nil {
		t.Fatalf("create turns: %v", err)
	}
	return db
}

func seedSessionDirectoryTurn(t *testing.T, db *sql.DB, sessionID, role, content string, ts int64, chatID string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES (?, ?, ?, ?, ?)`, sessionID, role, content, ts, chatID); err != nil {
		t.Fatalf("seed turn %s: %v", sessionID, err)
	}
}
