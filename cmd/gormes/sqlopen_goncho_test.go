package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

// TestSqlOpenGoncho_SetsBusyTimeout pins the busy_timeout invariant for the
// shared memory-DB opener: any reader/writer that opens the goncho memory
// store must wait (not fail-fast) when another connection holds a write
// lock. With busy_timeout=0 (Go sqlite3 driver default), concurrent
// connections see "database is locked" the moment the gateway starts a
// turn write — the exact symptom we observe in `gormes gateway logs`:
//
//	WARN goncho user turn write failed err="goncho: insert lifecycle
//	     message: sqlite3: database is locked"
//
// 5000ms gives the gateway plenty of time to finish a single write while
// keeping interactive commands responsive on the rare conflict.
func TestSqlOpenGoncho_SetsBusyTimeout(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "memory.db")
	db, err := sqlOpenGoncho(tmp)
	if err != nil {
		t.Fatalf("sqlOpenGoncho: %v", err)
	}
	defer db.Close()

	var got int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&got); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if got < 5000 {
		t.Errorf("busy_timeout = %dms, want >= 5000ms", got)
	}

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

// TestSqlOpenGoncho_ConcurrentConnectionsDoNotLockOut proves the runtime
// payoff: when one connection holds a write transaction briefly, a second
// connection opened via sqlOpenGoncho waits and succeeds rather than
// failing with "database is locked". Without busy_timeout this test would
// flake on the second exec.
func TestSqlOpenGoncho_ConcurrentConnectionsDoNotLockOut(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "memory.db")
	dbA, err := sqlOpenGoncho(tmp)
	if err != nil {
		t.Fatalf("open A: %v", err)
	}
	defer dbA.Close()
	if _, err := dbA.Exec(`CREATE TABLE t(x INTEGER)`); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Hold a writer transaction on connection A.
	tx, err := dbA.Begin()
	if err != nil {
		t.Fatalf("begin A: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO t(x) VALUES(1)`); err != nil {
		t.Fatalf("insert A: %v", err)
	}
	go func() {
		// Release the writer after a short delay so connection B can win.
		// The whole test runs in <1s well under busy_timeout.
		_ = tx.Commit()
	}()

	dbB, err := sqlOpenGoncho(tmp)
	if err != nil {
		t.Fatalf("open B: %v", err)
	}
	defer dbB.Close()
	// busy_timeout makes this wait instead of returning SQLITE_BUSY.
	if _, err := dbB.Exec(`INSERT INTO t(x) VALUES(2)`); err != nil {
		t.Fatalf("insert B should not lock out under busy_timeout=5000ms: %v", err)
	}

	var n int
	if err := dbB.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 2 {
		t.Errorf("rows = %d, want 2", n)
	}
}

// TestMemoryDBOpensRouteThroughSharedHelper is a static lint that proves
// every cmd/gormes/*.go memory-DB opener uses sqlOpenGoncho (or another
// helper that applies busy_timeout). Bare `sql.Open("sqlite3", path)` on
// the goncho memory.db leaves busy_timeout at 0 and races with the
// gateway's writer.
//
// The exception is sqlOpenGoncho itself, which is the canonical helper.
func TestMemoryDBOpensRouteThroughSharedHelper(t *testing.T) {
	pkgRoot := pkgGormesDir(t)
	// Files allowed to call sql.Open("sqlite3", ...) directly:
	// - gateway.go: defines sqlOpenGoncho itself.
	// Everything else must route through sqlOpenGoncho.
	allowed := map[string]bool{"gateway.go": true}
	var offenders []string
	entries, err := os.ReadDir(pkgRoot)
	if err != nil {
		t.Fatalf("read cmd/gormes: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if allowed[name] {
			continue
		}
		body, err := os.ReadFile(filepath.Join(pkgRoot, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(body), `sql.Open("sqlite3"`) {
			offenders = append(offenders, name)
		}
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("bare sql.Open(\"sqlite3\", ...) outside sqlOpenGoncho:\n  %v\n"+
			"route through sqlOpenGoncho so busy_timeout is set", offenders)
	}
}

// pkgGormesDir returns the absolute path of cmd/gormes via this test
// file's runtime.Caller, so the test does not depend on the caller's cwd.
func pkgGormesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(thisFile)
}
