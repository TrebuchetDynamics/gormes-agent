package mirror

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

// sync must not rewrite USER.md when the underlying graph is unchanged. The
// change-detection hash must therefore exclude the volatile "Last synced"
// timestamp, otherwise every tick produces a new hash and a redundant write.
func TestMirrorSyncSkipsRewriteWhenGraphUnchanged(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "memory.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	for _, stmt := range []string{
		`CREATE TABLE entities (id INTEGER PRIMARY KEY, name TEXT, type TEXT, description TEXT)`,
		`CREATE TABLE relationships (source_id INTEGER, target_id INTEGER, predicate TEXT, weight REAL)`,
		`INSERT INTO entities(name, type, description) VALUES ('A', 'PERSON', 'desc')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	userMD := filepath.Join(dir, "USER.md")
	// Distinct timestamps per sync, well over a second apart, so a timestamp in
	// the change-detection hash would force a redundant rewrite.
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := &Mirror{
		cfg:   MirrorConfig{Path: userMD},
		log:   slog.Default(),
		store: db,
		now:   func() time.Time { return clock },
	}

	m.sync()
	if _, err := os.Stat(userMD); err != nil {
		t.Fatalf("first sync did not write USER.md: %v", err)
	}

	// Remove the file; a second sync over the SAME graph must detect no change
	// and skip the write, so the file stays absent.
	if err := os.Remove(userMD); err != nil {
		t.Fatalf("remove USER.md: %v", err)
	}
	clock = clock.Add(time.Hour) // advance the sync clock past a second boundary
	m.sync()
	if _, err := os.Stat(userMD); !os.IsNotExist(err) {
		t.Fatalf("second sync rewrote USER.md despite unchanged graph (stat err=%v)", err)
	}
}
