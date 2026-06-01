package session

import (
	"database/sql"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

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
