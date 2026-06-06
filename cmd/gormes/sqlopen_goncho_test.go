package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/service"
	gonchoadapter "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"

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

func TestSqlOpenGoncho_SupportsGonchoV020ProfileScopedMemory(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "memory.db")
	db, err := sqlOpenGoncho(tmp)
	if err != nil {
		t.Fatalf("sqlOpenGoncho: %v", err)
	}
	defer db.Close()

	svc := goncho.NewService(db, goncho.Config{WorkspaceID: "gormes", ObserverPeerID: "gormes"}, nil)
	if err := svc.SetProfileInNamespace(context.Background(), goncho.MemoryNamespace{
		WorkspaceID: "gormes",
		ProfileID:   "mineru",
		PeerID:      "operator",
		Scope:       goncho.MemoryScopeProfile,
	}, []string{"Mineru prefers concise release notes."}); err != nil {
		t.Fatalf("SetProfileInNamespace: %v", err)
	}
	if _, err := svc.Conclude(context.Background(), goncho.ConcludeParams{
		ProfileID:  "mineru",
		Peer:       "operator",
		SessionKey: "release-session",
		Conclusion: "Goncho v0.2.0 is the active Gormes memory dependency.",
		Scope:      goncho.MemoryScopeProfile,
	}); err != nil {
		t.Fatalf("Conclude profile-scoped memory: %v", err)
	}
	got, err := svc.Search(context.Background(), goncho.SearchParams{
		ProfileID: "mineru",
		Peer:      "operator",
		Query:     "active memory dependency",
		Scope:     goncho.MemoryScopeProfile,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("Search profile-scoped memory: %v", err)
	}
	if got.ProfileID != "mineru" || len(got.Results) == 0 || !strings.Contains(got.Results[0].Content, "v0.2.0") {
		t.Fatalf("profile-scoped search = profile %q results %#v, want mineru result with v0.2.0", got.ProfileID, got.Results)
	}
}

func TestSqlOpenGoncho_AppliesObservationMigrations(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "memory.db")
	db, err := sqlOpenGoncho(tmp)
	if err != nil {
		t.Fatalf("sqlOpenGoncho: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"goncho_observations", "goncho_audit_events"} {
		var name string
		if err := db.QueryRowContext(context.Background(), `
			SELECT name
			FROM sqlite_master
			WHERE type = 'table' AND name = ?
		`, table).Scan(&name); err != nil {
			t.Fatalf("missing %s table after sqlOpenGoncho: %v", table, err)
		}
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

// TestGonchoGatewayTurnWriteSurvivesTransientProfileLock captures the
// profile-gateway failure mode from the audit bundle: a live turn tries to
// persist through the gateway Goncho adapter while another profile-scoped
// memory writer briefly owns SQLite's write lock. Hermes' SessionDB retries
// transient locked/busy writes at the application layer; Gormes must not lose
// the user turn after one short busy-handler timeout.
func TestGonchoGatewayTurnWriteSurvivesTransientProfileLock(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "memory.db")
	mem, err := memory.OpenSqlite(tmp, 32, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer func() {
		if err := mem.Close(context.Background()); err != nil {
			t.Fatalf("Close memory store: %v", err)
		}
	}()

	gonchoDB, err := sqlOpenGoncho(tmp)
	if err != nil {
		t.Fatalf("sqlOpenGoncho: %v", err)
	}
	defer gonchoDB.Close()
	gonchoDB.SetMaxOpenConns(1)
	gonchoDB.SetMaxIdleConns(1)
	if _, err := gonchoDB.Exec(`PRAGMA busy_timeout = 25`); err != nil {
		t.Fatalf("lower busy_timeout for deterministic lock regression: %v", err)
	}

	lockTx, err := mem.DB().BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin profile lock transaction: %v", err)
	}
	if _, err := lockTx.ExecContext(context.Background(), `
		INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		VALUES(?, ?, ?, ?, ?)
	`, "profile-session", "user", "background lifecycle writer", time.Now().Unix(), "telegram:profile"); err != nil {
		_ = lockTx.Rollback()
		t.Fatalf("hold profile write lock: %v", err)
	}

	svc := goncho.NewService(gonchoDB, goncho.Config{
		WorkspaceID:    "default",
		ObserverPeerID: "gormes",
	}, nil)
	store := gonchoadapter.NewStore(svc)
	errCh := make(chan error, 1)
	go func() {
		errCh <- store.AppendTurn(
			context.Background(),
			"telegram:profile",
			"profile-session",
			"user",
			"persist this user turn after the transient lock clears",
		)
	}()

	time.Sleep(100 * time.Millisecond)
	if err := lockTx.Commit(); err != nil {
		t.Fatalf("release profile write lock: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("gateway Goncho user turn write returned transient lock error: %v", err)
	}

	var persisted int
	if err := gonchoDB.QueryRowContext(context.Background(), `
		SELECT COUNT(*)
		FROM turns
		WHERE session_id = ?
		  AND role = ?
		  AND content = ?
		  AND memory_sync_status = 'ready'
	`, "profile-session", "user", "persist this user turn after the transient lock clears").Scan(&persisted); err != nil {
		t.Fatalf("count persisted user turn: %v", err)
	}
	if persisted != 1 {
		t.Fatalf("persisted user turn count = %d, want 1", persisted)
	}
}

func TestSqlOpenGoncho_SelfHealsCorruptDatabase(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "memory.db")
	if err := os.WriteFile(tmp, []byte("not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}

	db, err := sqlOpenGoncho(tmp)
	if err != nil {
		t.Fatalf("sqlOpenGoncho must quarantine and recreate corrupt memory.db: %v", err)
	}
	defer db.Close()

	var version string
	if err := db.QueryRowContext(context.Background(), `SELECT v FROM schema_meta WHERE k = 'version'`).Scan(&version); err != nil {
		t.Fatalf("self-healed memory.db must have goncho schema: %v", err)
	}
	if version == "" {
		t.Fatal("self-healed memory.db schema version is empty")
	}

	backups, err := filepath.Glob(tmp + ".corrupt-*")
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("corrupt DB must be preserved as one quarantine backup, got %v", backups)
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
	// - main.go: defines sqlOpenGoncho and sqlOpenGonchoUnmigrated.
	// Everything else must route through sqlOpenGoncho.
	allowed := map[string]bool{"main.go": true}
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
