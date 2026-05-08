package session

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

// TestListDirectorySessions_ReturnsEmptyOnMissingTurnsTable guards a
// regression observed on every fresh install: the goncho memory.db is
// created lazily on the first turn write, so on a clean checkout the
// `turns` table doesn't exist yet. Before this fix, `gormes session
// list` surfaced a SQL error (`no such table: turns`) instead of the
// friendly "No sessions found." UX it ships when the table is empty.
//
// Contract: a missing turns table is the empty-state path — return an
// empty slice with nil error so callers can render their existing
// empty-state placeholder.
func TestListDirectorySessions_ReturnsEmptyOnMissingTurnsTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "memory.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	// Force the file to exist without running migrations: the
	// `turns` table is intentionally absent.
	if _, err := db.Exec(`CREATE TABLE other (id INTEGER)`); err != nil {
		t.Fatalf("create placeholder table: %v", err)
	}

	got, err := ListDirectorySessions(context.Background(), db, DirectoryFilter{})
	if err != nil {
		t.Fatalf("ListDirectorySessions on DB without `turns` table must return nil err; got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice; got %d entries", len(got))
	}
}

// TestListDirectorySessions_RealOtherErrorsStillSurface keeps the
// regression fence: a missing-table is a soft empty state, but any
// OTHER SQL error (locked db, corrupt file, etc.) must still bubble
// up so operators see real failures.
func TestListDirectorySessions_RealOtherErrorsStillSurface(t *testing.T) {
	// nil db → callable surface returns a typed error, not the
	// missing-table soft path.
	_, err := ListDirectorySessions(context.Background(), nil, DirectoryFilter{})
	if err == nil {
		t.Fatal("nil db must error, not silently succeed")
	}
}
