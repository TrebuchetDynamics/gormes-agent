package session

import (
	"path/filepath"
	"testing"
)

func TestSQLOpenGonchoPinsSingleConnectionForBusyTimeout(t *testing.T) {
	// PRAGMA busy_timeout is per-connection. The pool must be pinned to one
	// connection so the 5s busy_timeout set at open time is actually honored;
	// otherwise database/sql may open extra connections with busy_timeout=0 and
	// concurrent writes fail immediately with SQLITE_BUSY instead of waiting.
	path := filepath.Join(t.TempDir(), "sessions.db")
	db, err := sqlOpenGoncho(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if max := db.Stats().MaxOpenConnections; max != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1 so per-connection busy_timeout is honored", max)
	}
}

func TestCoalesceSessionNameArgsMultiWordNames(t *testing.T) {
	got := CoalesceSessionNameArgs([]string{"-c", "my", "project", "sessions", "list"})
	want := []string{"-c", "my project", "sessions", "list"}
	if len(got) != len(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestTUISaveExportStemSanitizesEmptyAndPathSeparators(t *testing.T) {
	if got := tuiSaveExportStem("a/b\\c"); got != "a_b_c" {
		t.Fatalf("stem = %q, want a_b_c", got)
	}
	if got := tuiSaveExportStem("   "); got != "session" {
		t.Fatalf("empty stem = %q, want session", got)
	}
}
