# Gormes Phase 2.C — Thin Mapping Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist `(platform, chat_id) → session_id` in pure-Go bbolt so `cmd/gormes` and `cmd/gormes-telegram` survive restarts without losing the user's Python-server session handle.

**Architecture:** New `internal/session` package exposing a two-method `Map` interface with BoltMap (prod) + MemMap (test) implementations. Kernel gets one additive `Config.InitialSessionID` field. Adapters (Telegram bot, TUI) own the persistence loop — the kernel remains oblivious.

**Tech Stack:** Go 1.22+, `go.etcd.io/bbolt` (pure Go, zero CGO), existing `pflag`/`cobra` flag stacks, XDG directory conventions.

**Spec:** [`gormes/docs/superpowers/specs/2026-04-19-gormes-phase2c-persistence-design.md`](../specs/2026-04-19-gormes-phase2c-persistence-design.md)

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `gormes/go.mod` | Modify | Add `go.etcd.io/bbolt` dependency |
| `gormes/go.sum` | Modify | Lockfile update |
| `gormes/internal/session/session.go` | Create | Package doc, `Map` interface, sentinel errors, `TUIKey`/`TelegramKey` helpers |
| `gormes/internal/session/mem.go` | Create | `MemMap` — in-memory impl for tests |
| `gormes/internal/session/mem_test.go` | Create | MemMap unit tests |
| `gormes/internal/session/bolt.go` | Create | `BoltMap` — bbolt-backed impl, `OpenBolt` constructor |
| `gormes/internal/session/bolt_test.go` | Create | BoltMap tests (real disk via `t.TempDir()`) |
| `gormes/internal/kernel/kernel.go` | Modify | Add `Config.InitialSessionID`; copy into `k.sessionID` in `New()` |
| `gormes/internal/kernel/initial_session_test.go` | Create | `TestKernel_InitialSessionIDPrimesFirstRequest` |
| `gormes/internal/telegram/bot.go` | Modify | Add `Config.SessionMap`, `Config.SessionKey`, `lastSID` field, persistence hook in `runOutbound` |
| `gormes/internal/telegram/bot_test.go` | Modify | Add `TestBot_PersistsSessionIDToMap`, `TestBot_ResumesSessionIDAcrossRestart` |
| `gormes/internal/config/config.go` | Modify | Add `Resume` field, `--resume` flag, exported `SessionDBPath()` |
| `gormes/internal/config/config_test.go` | Modify | Add `TestLoad_ResumeFlag`, `TestSessionDBPath_HonorsXDG` |
| `gormes/cmd/gormes-telegram/main.go` | Modify | Open BoltMap, handle `--resume`, prime kernel, pass SessionMap to bot |
| `gormes/cmd/gormes/main.go` | Modify | Same wiring for TUI (cobra flag) |
| `gormes/internal/buildisolation_test.go` | Modify | Add `TestKernelHasNoSessionDep` |

---

## Task 1: Add `go.etcd.io/bbolt` dependency

**Files:**
- Modify: `gormes/go.mod`
- Modify: `gormes/go.sum`

- [ ] **Step 1: Add the dependency**

```bash
cd gormes
go get go.etcd.io/bbolt@latest
```

Expected: `go.mod` gains `go.etcd.io/bbolt v1.X.Y` in the `require` block; `go.sum` gains hashes.

- [ ] **Step 2: Verify the module graph compiles**

```bash
cd gormes
go build ./...
```

Expected: clean build. No code uses bbolt yet, so this only validates the module resolved.

- [ ] **Step 3: Verify binary size is still under budget (no regression yet)**

```bash
cd gormes
make build
ls -lh bin/gormes
```

Expected: `bin/gormes` size unchanged from `a85df164`-era (~7.9 MB). bbolt is present in go.sum but no binary imports it yet, so dead-code elimination keeps it out.

- [ ] **Step 4: Commit**

```bash
cd ..
git add gormes/go.mod gormes/go.sum
git commit -m "$(cat <<'EOF'
build(gormes): add go.etcd.io/bbolt dependency

Preparatory commit for Phase 2.C thin mapping persistence.
bbolt is pure Go (zero CGO), ~0.5 MB stripped — the only engine
that fits both the <=10 MB TUI budget and the <=12 MB bot budget.

No code imports it yet; dead-code elimination keeps the TUI
binary size unchanged until Task 9 wires cmd/gormes-telegram.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: `internal/session` package — interface + MemMap

**Files:**
- Create: `gormes/internal/session/session.go`
- Create: `gormes/internal/session/mem.go`
- Create: `gormes/internal/session/mem_test.go`

- [ ] **Step 1: Write the failing test**

Create `gormes/internal/session/mem_test.go`:

```go
package session

import (
	"context"
	"sync"
	"testing"
)

func TestMemMap_PutGetRoundTrip(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()

	if err := m.Put(ctx, "telegram:42", "sess-abc"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := m.Get(ctx, "telegram:42")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "sess-abc" {
		t.Errorf("Get = %q, want %q", got, "sess-abc")
	}
}

func TestMemMap_GetMissingReturnsEmpty(t *testing.T) {
	m := NewMemMap()
	got, err := m.Get(context.Background(), "telegram:999")
	if err != nil {
		t.Errorf("Get on missing key should not error, got %v", err)
	}
	if got != "" {
		t.Errorf("Get on missing key = %q, want \"\"", got)
	}
}

func TestMemMap_PutEmptyDeletes(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()
	_ = m.Put(ctx, "tui:default", "sess-x")
	_ = m.Put(ctx, "tui:default", "")
	got, _ := m.Get(ctx, "tui:default")
	if got != "" {
		t.Errorf("after Put(\"\"), Get = %q, want deleted (\"\")", got)
	}
}

func TestMemMap_CloseIdempotent(t *testing.T) {
	m := NewMemMap()
	if err := m.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Errorf("second Close should be no-op, got %v", err)
	}
}

func TestMemMap_CtxCancelShortCircuits(t *testing.T) {
	m := NewMemMap()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Get(ctx, "k"); err == nil {
		t.Errorf("Get on canceled ctx should return ctx.Err(), got nil")
	}
	if err := m.Put(ctx, "k", "v"); err == nil {
		t.Errorf("Put on canceled ctx should return ctx.Err(), got nil")
	}
}

func TestMemMap_ConcurrentSafe(t *testing.T) {
	m := NewMemMap()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(i int) { defer wg.Done(); _ = m.Put(context.Background(), "k", "v") }(i)
		go func(i int) { defer wg.Done(); _, _ = m.Get(context.Background(), "k") }(i)
	}
	wg.Wait()
}

func TestTUIKey(t *testing.T) {
	if TUIKey() != "tui:default" {
		t.Errorf("TUIKey() = %q, want %q", TUIKey(), "tui:default")
	}
}

func TestTelegramKey(t *testing.T) {
	if got := TelegramKey(5551234567); got != "telegram:5551234567" {
		t.Errorf("TelegramKey(5551234567) = %q, want %q", got, "telegram:5551234567")
	}
	if got := TelegramKey(-100); got != "telegram:-100" {
		t.Errorf("TelegramKey(-100) = %q, want %q", got, "telegram:-100")
	}
}
```

- [ ] **Step 2: Run — expect FAIL (package does not exist)**

```bash
cd gormes
go test ./internal/session/... 2>&1 | head -5
```

Expected: `no Go files in ...internal/session`.

- [ ] **Step 3: Write `session.go` (interface + sentinels + key builders)**

Create `gormes/internal/session/session.go`:

```go
// Package session persists (platform, chat_id) -> session_id mappings so
// Gormes binaries can resume the canonical Python-server transcript across
// restarts. See gormes/docs/superpowers/specs/2026-04-19-gormes-phase2c-persistence-design.md.
//
// Two implementations:
//   - BoltMap: bbolt-backed (production). See bolt.go.
//   - MemMap:  in-memory (tests). See mem.go.
//
// Both implement Map. Callers should treat a non-existent key as "no prior
// session" (Get returns ("", nil)) and use Put(key, "") to clear a mapping.
package session

import (
	"context"
	"errors"
	"strconv"
)

// Map persists session_id handles. Safe for concurrent use.
type Map interface {
	// Get returns the session_id for key, or ("", nil) if absent.
	// Honors ctx cancellation at the boundary only: bbolt I/O is not
	// mid-flight interruptible.
	Get(ctx context.Context, key string) (sessionID string, err error)

	// Put writes sessionID for key. Put(key, "") deletes the key and is
	// a no-op if the key was already absent.
	Put(ctx context.Context, key string, sessionID string) error

	// Close releases underlying resources. Idempotent.
	Close() error
}

// ErrDBLocked is returned by OpenBolt when another process holds the bbolt
// file lock. Caller should exit 1 with a clear message — retrying is
// pointless because dual-instance is a config bug.
var ErrDBLocked = errors.New("session: database locked by another process")

// ErrDBCorrupt is returned by OpenBolt when the bbolt file's magic bytes
// are wrong or the header is malformed. Caller should exit 1 and instruct
// the user to delete the file.
var ErrDBCorrupt = errors.New("session: database appears corrupted")

// TUIKey returns the canonical map key for the TUI binary.
func TUIKey() string { return "tui:default" }

// TelegramKey returns the canonical map key for a Telegram chat.
func TelegramKey(chatID int64) string {
	return "telegram:" + strconv.FormatInt(chatID, 10)
}
```

- [ ] **Step 4: Write `mem.go`**

Create `gormes/internal/session/mem.go`:

```go
package session

import (
	"context"
	"sync"
)

// MemMap is an in-memory Map for unit tests. Zero persistence across
// process restarts. Safe for concurrent use.
type MemMap struct {
	mu sync.Mutex
	m  map[string]string
}

// NewMemMap constructs an empty MemMap.
func NewMemMap() *MemMap {
	return &MemMap{m: make(map[string]string)}
}

func (m *MemMap) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.m[key], nil
}

func (m *MemMap) Put(ctx context.Context, key, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if sessionID == "" {
		delete(m.m, key)
	} else {
		m.m[key] = sessionID
	}
	return nil
}

// Close is a no-op. Always returns nil.
func (*MemMap) Close() error { return nil }
```

- [ ] **Step 5: Run — expect PASS**

```bash
cd gormes
go test -race ./internal/session/... -v
```

Expected: all 8 tests PASS.

- [ ] **Step 6: Commit**

```bash
cd ..
git add gormes/internal/session/session.go \
        gormes/internal/session/mem.go \
        gormes/internal/session/mem_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/session): Map interface + MemMap + key builders

New internal/session package. Map is a two-method interface
(Get/Put) with a Close. Get on missing key returns ("", nil) —
no ErrNotFound, because "no prior session" is the expected
startup state. Put(key, "") deletes.

MemMap is the test double: sync.Mutex + map[string]string.
Concurrent-safe; Close is a no-op.

Key builders (TUIKey, TelegramKey) give adapters a single
authoritative source for the "<platform>:<chat_id>" encoding.

Sentinel errors (ErrDBLocked, ErrDBCorrupt) are declared here
so BoltMap can reference them; Map consumers classify open-time
failures via errors.Is.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: `BoltMap` happy path — Open, Get, Put, Close

**Files:**
- Create: `gormes/internal/session/bolt.go`
- Create: `gormes/internal/session/bolt_test.go`

- [ ] **Step 1: Write the failing test**

Create `gormes/internal/session/bolt_test.go`:

```go
package session

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestBolt_PutGetRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer m.Close()

	ctx := context.Background()
	if err := m.Put(ctx, "telegram:42", "sess-abc"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := m.Get(ctx, "telegram:42")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "sess-abc" {
		t.Errorf("Get = %q, want %q", got, "sess-abc")
	}
}

func TestBolt_GetMissingReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	defer m.Close()
	got, err := m.Get(context.Background(), "does-not-exist")
	if err != nil {
		t.Errorf("Get on missing key should not error, got %v", err)
	}
	if got != "" {
		t.Errorf("Get on missing key = %q, want \"\"", got)
	}
}

func TestBolt_AutoCreatesParentDir(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "newsubdir")
	path := filepath.Join(parent, "sessions.db")
	m, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("OpenBolt (missing parent dir): %v", err)
	}
	defer m.Close()

	info, err := os.Stat(parent)
	if err != nil {
		t.Fatalf("parent dir should exist after OpenBolt: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("parent is not a dir")
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("parent dir perm = %o, want 0700", perm)
	}
}

func TestBolt_CloseIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	if err := m.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Errorf("second Close should be no-op, got %v", err)
	}
}

func TestBolt_ConcurrentPutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	defer m.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _ = m.Put(ctx, "k", "v") }()
		go func() { defer wg.Done(); _, _ = m.Get(ctx, "k") }()
	}
	wg.Wait()

	got, _ := m.Get(ctx, "k")
	if got != "v" {
		t.Errorf("after concurrent writes, Get = %q, want %q", got, "v")
	}
}
```

- [ ] **Step 2: Run — expect FAIL (OpenBolt undefined)**

```bash
cd gormes
go test ./internal/session/... 2>&1 | head -5
```

Expected: `undefined: OpenBolt`.

- [ ] **Step 3: Write `bolt.go`**

Create `gormes/internal/session/bolt.go`:

```go
package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

// bucketName is the top-level bbolt bucket for the v1 schema. Phase 3 may
// add sessions_v2 alongside; rely on errors.Is, not bucket-name strings.
const bucketName = "sessions_v1"

// openTimeout caps how long OpenBolt waits for the file lock before
// returning ErrDBLocked. 100 ms is enough to ride out a brief overlap
// during systemd restart handoff without masking real dual-instance bugs.
const openTimeout = 100 * time.Millisecond

// BoltMap is the production Map backed by a single bbolt file.
type BoltMap struct {
	closeMu sync.Mutex
	db      *bolt.DB // nil after Close
}

// OpenBolt opens (or creates) the bbolt file at path, ensuring the parent
// directory exists with mode 0700 and the sessions_v1 bucket exists.
// Translates bbolt's internal errors into ErrDBLocked / ErrDBCorrupt where
// appropriate; other errors are surfaced wrapped.
func OpenBolt(path string) (*BoltMap, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("session: create parent dir for %s: %w", path, err)
	}

	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: openTimeout})
	if err != nil {
		return nil, classifyOpenErr(path, err)
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	}); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("session: create bucket in %s: %w", path, err)
	}

	return &BoltMap{db: db}, nil
}

func classifyOpenErr(path string, err error) error {
	// bbolt returns a timeout-shaped error when the file lock is held.
	// It does not export a sentinel; match on the error string. The
	// upstream message has been stable since bbolt 1.3.
	if err != nil && (errors.Is(err, bolt.ErrTimeout) ||
		containsAny(err.Error(), "timeout", "resource temporarily unavailable")) {
		return fmt.Errorf("%w: %s", ErrDBLocked, path)
	}
	// bbolt returns ErrInvalid for bad magic / short file / wrong version.
	if err != nil && (errors.Is(err, bolt.ErrInvalid) ||
		containsAny(err.Error(), "invalid database", "version mismatch", "file size too small")) {
		return fmt.Errorf("%w: %s", ErrDBCorrupt, path)
	}
	return fmt.Errorf("session: open %s: %w", path, err)
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if len(n) > 0 && stringContainsFold(s, n) {
			return true
		}
	}
	return false
}

func stringContainsFold(s, substr string) bool {
	// Fold-equivalent Contains: avoid strings.ToLower allocations.
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			a, b := s[i+j], substr[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (m *BoltMap) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return "", errors.New("session: BoltMap is closed")
	}

	var out string
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil // bucket missing = treat as empty
		}
		v := b.Get([]byte(key))
		if v != nil {
			out = string(v)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("session: get %q: %w", key, err)
	}
	return out, nil
}

func (m *BoltMap) Put(ctx context.Context, key, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return errors.New("session: BoltMap is closed")
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return errors.New("session: bucket missing")
		}
		if sessionID == "" {
			return b.Delete([]byte(key))
		}
		return b.Put([]byte(key), []byte(sessionID))
	})
}

// Close flushes and releases the bbolt file lock. Idempotent.
func (m *BoltMap) Close() error {
	m.closeMu.Lock()
	defer m.closeMu.Unlock()
	if m.db == nil {
		return nil
	}
	err := m.db.Close()
	m.db = nil
	return err
}
```

- [ ] **Step 4: Run — expect PASS**

```bash
cd gormes
go test -race ./internal/session/... -v
```

Expected: all 5 new Bolt tests PASS plus the 8 MemMap tests still PASS.

- [ ] **Step 5: Commit**

```bash
cd ..
git add gormes/internal/session/bolt.go gormes/internal/session/bolt_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/session): BoltMap — bbolt-backed Map (happy path)

OpenBolt creates the parent dir (0700) + the sessions_v1 bucket
idempotently. Wraps bbolt errors: lock contention -> ErrDBLocked,
invalid header -> ErrDBCorrupt. File mode 0600.

Get on a never-written key returns ("", nil); no ErrNotFound.
Close is idempotent — double-close returns nil.

Concurrency: bbolt's single-writer mutex + sync.Mutex around db
handle swap during Close. -race clean under 50x2 goroutine Put/Get.

Disk tests use t.TempDir() — zero repo pollution.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: `BoltMap` — `Put(key, "")` deletes

**Files:**
- Modify: `gormes/internal/session/bolt_test.go`

- [ ] **Step 1: Write the failing test (append to bolt_test.go)**

Append to `gormes/internal/session/bolt_test.go`:

```go
func TestBolt_PutEmptyDeletes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	defer m.Close()
	ctx := context.Background()

	_ = m.Put(ctx, "tui:default", "sess-x")
	if got, _ := m.Get(ctx, "tui:default"); got != "sess-x" {
		t.Fatalf("setup: Get = %q, want sess-x", got)
	}

	if err := m.Put(ctx, "tui:default", ""); err != nil {
		t.Fatalf("Put(\"\") to delete: %v", err)
	}

	got, err := m.Get(ctx, "tui:default")
	if err != nil {
		t.Errorf("Get after delete: %v", err)
	}
	if got != "" {
		t.Errorf("after Put(\"\"), Get = %q, want deleted (\"\")", got)
	}
}

func TestBolt_PutEmptyOnMissingKeyIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	defer m.Close()

	if err := m.Put(context.Background(), "never-existed", ""); err != nil {
		t.Errorf("Put(\"\") on missing key should be no-op, got %v", err)
	}
}
```

- [ ] **Step 2: Run — expect PASS**

The BoltMap.Put implementation from Task 3 already handles `sessionID == ""` via `b.Delete(...)`, and bbolt's Delete on a missing key is a no-op. These tests should pass without code changes.

```bash
cd gormes
go test -race ./internal/session/... -run TestBolt_PutEmpty -v
```

Expected: both tests PASS.

- [ ] **Step 3: Commit**

```bash
cd ..
git add gormes/internal/session/bolt_test.go
git commit -m "$(cat <<'EOF'
test(gormes/session): verify Put(key, "") deletes + missing-key no-op

Pins the spec's delete-via-empty-string contract:
  - Put with empty value deletes the key
  - Subsequent Get returns ("", nil) as if never written
  - Put("") on an already-missing key is a silent no-op

These are the semantics the /new Telegram handler relies on when
the kernel's post-reset render frame carries SessionID="".

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: `BoltMap` — failure modes (lock, corrupt, permission)

**Files:**
- Modify: `gormes/internal/session/bolt_test.go`

- [ ] **Step 1: Write the failing tests (append to bolt_test.go)**

Append to `gormes/internal/session/bolt_test.go`:

```go
import "errors" // if not already imported; add to existing import block

func TestBolt_LockContention(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")

	m1, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("first OpenBolt: %v", err)
	}
	defer m1.Close()

	start := time.Now()
	_, err = OpenBolt(path)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrDBLocked) {
		t.Errorf("second OpenBolt err = %v, want errors.Is(err, ErrDBLocked)", err)
	}
	// 100ms openTimeout + tolerance. Should return within ~250ms.
	if elapsed > 500*time.Millisecond {
		t.Errorf("second OpenBolt took %v, should time out near %v", elapsed, openTimeout)
	}
}

func TestBolt_CorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	// Write a plausible-sized file of garbage where bbolt expects its header.
	garbage := make([]byte, 4096)
	for i := range garbage {
		garbage[i] = byte(i) // non-zero, non-magic
	}
	if err := os.WriteFile(path, garbage, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := OpenBolt(path)
	if !errors.Is(err, ErrDBCorrupt) {
		t.Errorf("OpenBolt on garbage file err = %v, want errors.Is(err, ErrDBCorrupt)", err)
	}
}

func TestBolt_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests skipped as root — CI containers often run as root")
	}
	parent := filepath.Join(t.TempDir(), "blocked")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	// Make the parent non-traversable so the Open fails on MkdirAll or on
	// attempting to create the DB file itself.
	if err := os.Chmod(parent, 0); err != nil {
		t.Fatal(err)
	}
	// Restore perms so t.TempDir cleanup succeeds.
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	path := filepath.Join(parent, "sub", "sessions.db")
	_, err := OpenBolt(path)
	if err == nil {
		t.Fatal("OpenBolt on unwritable parent should fail")
	}
	if errors.Is(err, ErrDBLocked) || errors.Is(err, ErrDBCorrupt) {
		t.Errorf("permission error should not classify as Locked/Corrupt: %v", err)
	}
}
```

Also add `"time"` to the existing import block if it's not already there.

- [ ] **Step 2: Run — expect PASS (error classification logic already in Task 3 code)**

```bash
cd gormes
go test -race ./internal/session/... -run TestBolt_(LockContention|CorruptFile|PermissionDenied) -v
```

Expected: all three PASS.

If `TestBolt_CorruptFile` fails because the garbage bytes happen to pass bbolt's initial magic check, regenerate garbage with a different seed — do NOT weaken the error classification. The `stringContainsFold` check in `classifyOpenErr` covers "invalid database", "version mismatch", and "file size too small"; inspect the actual bbolt error if the test fails and extend the substring list if a new message variant appears.

- [ ] **Step 3: Commit**

```bash
cd ..
git add gormes/internal/session/bolt_test.go
git commit -m "$(cat <<'EOF'
test(gormes/session): BoltMap failure modes

Pins the three failure modes called out in the spec:
  - LockContention: second OpenBolt on a held file returns
    ErrDBLocked within ~100 ms (no retry loop)
  - CorruptFile: 4 KB of garbage where bbolt expects its header
    classifies to ErrDBCorrupt via classifyOpenErr
  - PermissionDenied: unwritable parent dir yields a plain wrapped
    error — crucially NOT classified as Locked/Corrupt, so callers
    don't lie to users about the problem

PermissionDenied skips when running as root (common in CI).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Kernel `Config.InitialSessionID` seam

**Files:**
- Modify: `gormes/internal/kernel/kernel.go`
- Create: `gormes/internal/kernel/initial_session_test.go`

- [ ] **Step 1: Write the failing test**

Create `gormes/internal/kernel/initial_session_test.go`:

```go
package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
)

// TestKernel_InitialSessionIDPrimesFirstRequest proves that InitialSessionID
// on kernel.Config is copied into k.sessionID before the Run loop starts,
// so the first outbound ChatRequest carries that session_id in the
// X-Hermes-Session-Id header. Without this, --resume would silently no-op.
func TestKernel_InitialSessionIDPrimesFirstRequest(t *testing.T) {
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "ok", TokensOut: 1},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 1, TokensOut: 1},
	}, "sess-from-server")

	k := New(Config{
		Model:            "hermes-agent",
		Endpoint:         "http://mock",
		Admission:        Admission{MaxBytes: 200_000, MaxLines: 10_000},
		InitialSessionID: "sess-primed-from-disk",
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render() // initial idle

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}

	// Wait for the turn to finish.
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)

	// Inspect the last request the mock client saw.
	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("mock client received zero requests")
	}
	if got := reqs[0].SessionID; got != "sess-primed-from-disk" {
		t.Errorf("first request.SessionID = %q, want %q", got, "sess-primed-from-disk")
	}
}

// TestKernel_InitialSessionIDEmptyKeepsExistingBehavior proves zero-value
// InitialSessionID does not change existing kernel behavior — all Phase
// 1/1.5/2.A/2.B.1 tests must remain unaffected.
func TestKernel_InitialSessionIDEmptyKeepsExistingBehavior(t *testing.T) {
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{
		{Kind: hermes.EventDone, FinishReason: "stop"},
	}, "sess-fresh")

	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"})
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle
	}, 2*time.Second)

	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("zero requests")
	}
	if got := reqs[0].SessionID; got != "" {
		t.Errorf("first request.SessionID = %q, want \"\" (zero-value Initial)", got)
	}
}
```

If `hermes.MockClient` does not expose `Requests()` — read `internal/hermes/mock.go` and use whatever accessor does exist (e.g., `mc.LastRequest()`). Adjust the test line `reqs := mc.Requests()` accordingly. DO NOT add a new accessor unless one is strictly necessary; an existing record/observation field almost certainly exists.

- [ ] **Step 2: Run — expect FAIL (InitialSessionID field does not exist)**

```bash
cd gormes
go test ./internal/kernel/... -run TestKernel_InitialSession 2>&1 | head -5
```

Expected: `unknown field InitialSessionID in struct literal of type kernel.Config`.

- [ ] **Step 3: Add the field to `Config` in `kernel.go`**

In `gormes/internal/kernel/kernel.go`, modify the `Config` struct:

```go
type Config struct {
	Model             string
	Endpoint          string
	Admission         Admission
	Tools             *tools.Registry // nil → tool_calls are treated as fatal
	MaxToolIterations int             // default 10 when zero
	MaxToolDuration   time.Duration   // default 30s when zero
	// InitialSessionID primes k.sessionID at New() — used by adapters that
	// load a persisted session handle from internal/session before starting
	// the kernel. Zero value preserves pre-Phase-2.C behavior (fresh session).
	InitialSessionID string
}
```

- [ ] **Step 4: Copy the field into `k.sessionID` in `New`**

In the same file, modify `New`:

```go
func New(cfg Config, c hermes.Client, s store.Store, tm telemetry.Telemetry, log *slog.Logger) *Kernel {
	if log == nil {
		log = slog.Default()
	}
	tm.SetModel(cfg.Model)
	return &Kernel{
		cfg:       cfg,
		client:    c,
		store:     s,
		tm:        tm,
		log:       log,
		render:    make(chan RenderFrame, RenderMailboxCap),
		events:    make(chan PlatformEvent, PlatformEventMailboxCap),
		sessionID: cfg.InitialSessionID,
	}
}
```

- [ ] **Step 5: Run — expect PASS**

```bash
cd gormes
go test -race ./internal/kernel/... -timeout 90s
```

Expected: all kernel tests still PASS (Phase 1 / 1.5 / 2.A / 2.B.1) plus the two new `TestKernel_InitialSessionID*` tests.

- [ ] **Step 6: Commit**

```bash
cd ..
git add gormes/internal/kernel/kernel.go gormes/internal/kernel/initial_session_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/kernel): Config.InitialSessionID primes k.sessionID

Additive: one optional field on kernel.Config. If set,
kernel.New copies it into k.sessionID so the first outbound
ChatRequest carries X-Hermes-Session-Id — enabling Phase 2.C
--resume and the adapter's startup session-replay flow.

Zero value preserves every existing test's behavior. The two
new tests pin both directions: primed => request carries the
id; empty => request.SessionID == "".

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Telegram bot persistence hook

**Files:**
- Modify: `gormes/internal/telegram/bot.go`
- Modify: `gormes/internal/telegram/bot_test.go`

- [ ] **Step 1: Write the failing test (append to bot_test.go)**

Append to `gormes/internal/telegram/bot_test.go`:

```go
// Import block must include:
//   "github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"

// TestBot_PersistsSessionIDToMap proves the bot's outbound goroutine
// calls SessionMap.Put exactly when the kernel's RenderFrame.SessionID
// changes. Uses MemMap + scripted hermes.MockClient — no disk, no network.
func TestBot_PersistsSessionIDToMap(t *testing.T) {
	mc := newMockClient()
	smap := session.NewMemMap()

	hmc := hermes.NewMockClient()
	reply := "ok"
	events := make([]hermes.Event, 0, len(reply)+1)
	for _, ch := range reply {
		events = append(events, hermes.Event{Kind: hermes.EventToken, Token: string(ch), TokensOut: 1})
	}
	events = append(events, hermes.Event{Kind: hermes.EventDone, FinishReason: "stop"})
	hmc.Script(events, "sess-persisted-xyz")

	k := kernel.New(kernel.Config{
		Model: "hermes-agent", Endpoint: "http://mock",
		Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, hmc, store.NewNoop(), telemetry.New(), nil)

	key := session.TelegramKey(42)
	b := New(Config{
		AllowedChatID: 42,
		CoalesceMs:    100,
		SessionMap:    smap,
		SessionKey:    key,
	}, mc, k, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	go func() { _ = b.Run(ctx) }()

	mc.pushTextUpdate(42, "ping")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got, _ := smap.Get(context.Background(), key); got == "sess-persisted-xyz" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	got, _ := smap.Get(context.Background(), key)
	if got != "sess-persisted-xyz" {
		t.Errorf("SessionMap[%q] = %q, want %q", key, got, "sess-persisted-xyz")
	}

	cancel()
	mc.closeUpdates()
	time.Sleep(50 * time.Millisecond)
}
```

- [ ] **Step 2: Run — expect FAIL (SessionMap / SessionKey undefined on Config)**

```bash
cd gormes
go test ./internal/telegram/... -run TestBot_PersistsSessionIDToMap 2>&1 | head -5
```

Expected: `unknown field SessionMap in struct literal`.

- [ ] **Step 3: Extend `Config` + add `lastSID` to `Bot`**

In `gormes/internal/telegram/bot.go`, add the import and modify the struct:

```go
import (
	// ... existing imports ...
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"
)

// Config drives the Bot adapter. AllowedChatID and FirstRunDiscovery follow
// the spec's M1/M2 rules: either a non-zero allowlist OR discovery enabled,
// never neither.
type Config struct {
	AllowedChatID     int64
	CoalesceMs        int
	FirstRunDiscovery bool
	// SessionMap + SessionKey (Phase 2.C) — optional. When SessionMap is
	// non-nil, the outbound goroutine persists k.sessionID on every frame
	// where it changed. Nil disables persistence (Phase 2.B.1 behavior).
	SessionMap session.Map
	SessionKey string
}

// Bot is the Telegram adapter. Kernel-side state (draft, phase, history)
// lives in *kernel.Kernel; Bot holds only per-adapter streaming state.
type Bot struct {
	cfg     Config
	client  telegramClient
	kernel  *kernel.Kernel
	log     *slog.Logger
	lastSID string // most recently persisted session_id (prevents duplicate Puts)
}
```

- [ ] **Step 4: Add the persistence hook in `runOutbound`**

In `gormes/internal/telegram/bot.go`, modify `runOutbound` to call a new helper `persistIfChanged` before dispatching each frame:

```go
func (b *Bot) runOutbound(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	frames := b.kernel.Render()
	var c *coalescer
	var cCancel context.CancelFunc

	for {
		select {
		case <-ctx.Done():
			if cCancel != nil {
				cCancel()
			}
			return
		case f, ok := <-frames:
			if !ok {
				if cCancel != nil {
					cCancel()
				}
				return
			}
			b.persistIfChanged(ctx, f)
			b.handleFrame(ctx, f, &c, &cCancel, wg)
		}
	}
}

// persistIfChanged writes the frame's SessionID to SessionMap when it has
// changed from the last persisted value. Failures log a WARN and do NOT
// fail the turn — persistence is best-effort on the render path.
func (b *Bot) persistIfChanged(ctx context.Context, f kernel.RenderFrame) {
	if b.cfg.SessionMap == nil || b.cfg.SessionKey == "" {
		return
	}
	if f.SessionID == b.lastSID {
		return
	}
	if err := b.cfg.SessionMap.Put(ctx, b.cfg.SessionKey, f.SessionID); err != nil {
		b.log.Warn("failed to persist session_id",
			"key", b.cfg.SessionKey,
			"session_id", f.SessionID,
			"err", err)
		return
	}
	b.lastSID = f.SessionID
}
```

- [ ] **Step 5: Run — expect PASS**

```bash
cd gormes
go test -race ./internal/telegram/... -timeout 60s -count=1
```

Expected: all existing telegram tests still PASS (they pass nil SessionMap implicitly via zero-value Config) plus `TestBot_PersistsSessionIDToMap`.

- [ ] **Step 6: Commit**

```bash
cd ..
git add gormes/internal/telegram/bot.go gormes/internal/telegram/bot_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/telegram): persist session_id on render-frame change

Optional SessionMap + SessionKey on telegram.Config. When set,
runOutbound's new persistIfChanged hook Puts k.sessionID to the
map whenever a RenderFrame carries a different SessionID than
the last persisted value.

Best-effort: Put failures log slog.Warn and continue — the
user's turn already succeeded against Python; only post-restart
resume is at risk.

Nil SessionMap preserves Phase-2.B.1 behavior; all existing
tests pass without modification.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: `config.Resume` flag + `SessionDBPath()`

**Files:**
- Modify: `gormes/internal/config/config.go`
- Modify: `gormes/internal/config/config_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `gormes/internal/config/config_test.go`:

```go
func TestLoad_ResumeFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load([]string{"--resume", "sess-abc123"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Resume != "sess-abc123" {
		t.Errorf("Resume = %q, want %q", cfg.Resume, "sess-abc123")
	}
}

func TestLoad_ResumeFlagEmptyDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil) // nil means skip flag parsing — existing contract
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Resume != "" {
		t.Errorf("Resume (no flags) = %q, want \"\"", cfg.Resume)
	}
}

func TestSessionDBPath_HonorsXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/gormes-test-xdg")
	got := SessionDBPath()
	want := "/tmp/gormes-test-xdg/gormes/sessions.db"
	if got != want {
		t.Errorf("SessionDBPath() = %q, want %q", got, want)
	}
}

func TestSessionDBPath_DefaultsToHomeLocalShare(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	home, _ := os.UserHomeDir()
	got := SessionDBPath()
	want := filepath.Join(home, ".local", "share", "gormes", "sessions.db")
	if got != want {
		t.Errorf("SessionDBPath() with empty XDG_DATA_HOME = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run — expect FAIL (Resume field + SessionDBPath undefined)**

```bash
cd gormes
go test ./internal/config/... -run "Resume|SessionDB" 2>&1 | head -5
```

Expected: `unknown field Resume` and/or `undefined: SessionDBPath`.

- [ ] **Step 3: Extend `config.go`**

Modify `gormes/internal/config/config.go`:

Add `Resume string` to `Config`:

```go
type Config struct {
	Hermes   HermesCfg   `toml:"hermes"`
	TUI      TUICfg      `toml:"tui"`
	Input    InputCfg    `toml:"input"`
	Telegram TelegramCfg `toml:"telegram"`
	// Resume is set only via the --resume CLI flag; intentionally not
	// a TOML field. Empty means "use whatever internal/session had
	// persisted for this binary's default key."
	Resume string `toml:"-"`
}
```

Extend `loadFlags` to register `--resume`:

```go
func loadFlags(cfg *Config, args []string) error {
	if args == nil {
		return nil
	}
	fs := pflag.NewFlagSet("gormes", pflag.ContinueOnError)
	endpoint := fs.String("endpoint", "", "Hermes api_server base URL")
	model := fs.String("model", "", "served model name")
	resume := fs.String("resume", "", "override persisted session_id for this binary's default key")
	// No --api-key flag — secrets stay out of process argv.
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *endpoint != "" {
		cfg.Hermes.Endpoint = *endpoint
	}
	if *model != "" {
		cfg.Hermes.Model = *model
	}
	if *resume != "" {
		cfg.Resume = *resume
	}
	return nil
}
```

Add `SessionDBPath()` near `LogPath` / `CrashLogDir`:

```go
// SessionDBPath returns the default location of the bbolt sessions map.
// Honors XDG_DATA_HOME; falls back to ~/.local/share/gormes/sessions.db.
func SessionDBPath() string {
	return filepath.Join(xdgDataHome(), "gormes", "sessions.db")
}
```

- [ ] **Step 4: Run — expect PASS**

```bash
cd gormes
go test -race ./internal/config/... -run "Resume|SessionDB" -v
```

Expected: all 4 new tests PASS; existing config tests still PASS.

- [ ] **Step 5: Commit**

```bash
cd ..
git add gormes/internal/config/config.go gormes/internal/config/config_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/config): --resume flag + SessionDBPath()

Adds a CLI-only Resume string field (not TOML — a one-shot
override should not live in persistent config). Wired through
the existing loadFlags pflag stack used by cmd/gormes-telegram.

SessionDBPath() is exported (symmetry with LogPath / CrashLogDir)
and honors XDG_DATA_HOME, defaulting to ~/.local/share/gormes/
sessions.db. Four new tests pin both the flag path and the XDG
path resolution — including the empty-XDG fallback.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Wire `cmd/gormes-telegram` to the map

**Files:**
- Modify: `gormes/cmd/gormes-telegram/main.go`

- [ ] **Step 1: Modify `cmd/gormes-telegram/main.go`**

Replace the entire file with:

```go
// Command gormes-telegram is the Phase-2.B.1 Telegram adapter binary.
// Phase 2.C adds persistent session-id resume via internal/session.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gormes-telegram:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if cfg.Telegram.BotToken == "" {
		return fmt.Errorf("no Telegram bot token — set GORMES_TELEGRAM_TOKEN env or [telegram].bot_token in config.toml")
	}
	if cfg.Telegram.AllowedChatID == 0 && !cfg.Telegram.FirstRunDiscovery {
		return fmt.Errorf("no chat allowlist and discovery disabled — set one of [telegram].allowed_chat_id or [telegram].first_run_discovery = true")
	}
	if os.Getenv("GORMES_TELEGRAM_TOKEN") == "" {
		slog.Warn("bot_token read from config.toml; prefer GORMES_TELEGRAM_TOKEN env var for secrets")
	}

	// Phase 2.C — open the session map before the kernel so we can prime it.
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()

	ctx := context.Background()
	var key string
	if cfg.Telegram.AllowedChatID != 0 {
		key = session.TelegramKey(cfg.Telegram.AllowedChatID)
		if cfg.Resume != "" {
			if err := smap.Put(ctx, key, cfg.Resume); err != nil {
				slog.Warn("failed to apply --resume override", "err", err)
			}
		}
	}
	var initialSID string
	if key != "" {
		if sid, err := smap.Get(ctx, key); err != nil {
			slog.Warn("could not load initial session_id", "key", key, "err", err)
		} else {
			initialSID = sid
			if sid != "" {
				slog.Info("resuming persisted session", "key", key, "session_id", sid)
			}
		}
	}

	hc := hermes.NewHTTPClient(cfg.Hermes.Endpoint, cfg.Hermes.APIKey)

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})
	reg.MustRegister(&tools.NowTool{})
	reg.MustRegister(&tools.RandIntTool{})

	tm := telemetry.New()
	k := kernel.New(kernel.Config{
		Model:             cfg.Hermes.Model,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             reg,
		MaxToolIterations: 10,
		MaxToolDuration:   30 * time.Second,
		InitialSessionID:  initialSID,
	}, hc, store.NewNoop(), tm, slog.Default())

	tc, err := telegram.NewRealClient(cfg.Telegram.BotToken)
	if err != nil {
		return err
	}

	bot := telegram.New(telegram.Config{
		AllowedChatID:     cfg.Telegram.AllowedChatID,
		CoalesceMs:        cfg.Telegram.CoalesceMs,
		FirstRunDiscovery: cfg.Telegram.FirstRunDiscovery,
		SessionMap:        smap,
		SessionKey:        key,
	}, tc, k, slog.Default())

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go k.Run(rootCtx)
	go func() {
		<-rootCtx.Done()
		time.AfterFunc(kernel.ShutdownBudget, func() {
			slog.Error("shutdown budget exceeded; forcing exit")
			os.Exit(3)
		})
	}()

	slog.Info("gormes-telegram starting",
		"endpoint", cfg.Hermes.Endpoint,
		"allowed_chat_id", cfg.Telegram.AllowedChatID,
		"discovery", cfg.Telegram.FirstRunDiscovery,
		"sessions_db", config.SessionDBPath())
	return bot.Run(rootCtx)
}
```

- [ ] **Step 2: Build + smoke (empty token still exits 1)**

```bash
cd gormes
go build -o bin/gormes-telegram ./cmd/gormes-telegram
./bin/gormes-telegram 2>&1 || echo "expected exit 1"
```

Expected: exit 1 with `"no Telegram bot token — ..."`. (The map Open at `~/.local/share/gormes/sessions.db` succeeds silently before the token check fires.)

- [ ] **Step 3: Smoke `--resume` override compiles and parses**

```bash
cd gormes
GORMES_TELEGRAM_TOKEN=abc:xyz ./bin/gormes-telegram --resume sess-smoke-test 2>&1 | head -5 || true
```

Expected: the binary proceeds past token validation into the `no chat allowlist and discovery disabled` OR starts trying to reach Telegram (depending on env). What matters: NO flag-parse error. `"unknown flag: --resume"` would be a failure.

- [ ] **Step 4: Full build + vet**

```bash
cd gormes
go build ./...
go vet ./...
```

Expected: clean.

- [ ] **Step 5: Commit**

```bash
cd ..
git add gormes/cmd/gormes-telegram/main.go
git commit -m "$(cat <<'EOF'
feat(gormes/cmd/telegram): wire internal/session.BoltMap

cmd/gormes-telegram now opens the bbolt sessions map before
constructing the kernel, loads the persisted session_id for
its allowlisted chat, and primes kernel.Config.InitialSessionID
so the first turn resumes the Python-server session.

Switched config.Load(nil) -> config.Load(os.Args[1:]) so the
new --resume flag actually binds.

SessionMap + SessionKey are threaded into telegram.Config so
the bot's outbound goroutine can Put updates on every kernel
sessionID change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Wire `cmd/gormes` (TUI) to the map

**Files:**
- Modify: `gormes/cmd/gormes/main.go`

- [ ] **Step 1: Read the current state**

Read `gormes/cmd/gormes/main.go` in full. The TUI uses cobra, not pflag-direct. The `--resume` flag must be registered as a cobra flag on the root command and bridged into `cfg.Resume` after `config.Load(nil)`.

- [ ] **Step 2: Modify `cmd/gormes/main.go`**

Two edits:

First, register the cobra flag in `main()`:

```go
func main() {
	defer func() {
		if r := recover(); r != nil {
			dumpCrash(r)
			os.Exit(2)
		}
	}()

	root := &cobra.Command{
		Use:          "gormes",
		Short:        "Go frontend for Hermes Agent",
		SilenceUsage: true,
		RunE:         runTUI,
	}
	root.Flags().Bool("offline", false, "skip startup api_server health check (dev only — turns the TUI into a cosmetic smoke-tester)")
	root.Flags().String("resume", "", "override persisted session_id for the TUI's default key")
	root.AddCommand(doctorCmd, versionCmd)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
```

Second, in `runTUI`, open the session map, bridge the cobra flag, and prime the kernel:

```go
func runTUI(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return err
	}

	c := hermes.NewHTTPClient(cfg.Hermes.Endpoint, cfg.Hermes.APIKey)

	offline, _ := cmd.Flags().GetBool("offline")
	if !offline {
		healthCtx, healthCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := c.Health(healthCtx); err != nil {
			healthCancel()
			fmt.Fprintf(os.Stderr,
				"api_server not reachable at %s: %v\n\nStart it with:\n  API_SERVER_ENABLED=true hermes gateway start\n\nOr pass --offline to render the TUI without a live server (dev only).\n",
				cfg.Hermes.Endpoint, err)
			return err
		}
		healthCancel()
	}

	// Phase 2.C — open the session map; honor --resume.
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()

	resumeFlag, _ := cmd.Flags().GetString("resume")
	ctx := context.Background()
	key := session.TUIKey()
	if resumeFlag != "" {
		if err := smap.Put(ctx, key, resumeFlag); err != nil {
			slog.Warn("failed to apply --resume override", "err", err)
		}
	}
	var initialSID string
	if sid, err := smap.Get(ctx, key); err != nil {
		slog.Warn("could not load initial session_id", "key", key, "err", err)
	} else {
		initialSID = sid
		if sid != "" {
			slog.Info("resuming persisted session", "key", key, "session_id", sid)
		}
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	tm := telemetry.New()
	k := kernel.New(kernel.Config{
		Model:             cfg.Hermes.Model,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             buildDefaultRegistry(),
		MaxToolIterations: 10,
		MaxToolDuration:   30 * time.Second,
		InitialSessionID:  initialSID,
	}, c, store.NewNoop(), tm, slog.Default())

	// Persist session-id updates from the TUI kernel too — same pattern as
	// the bot, inlined here because the TUI does not use internal/telegram.
	go persistTUISessionID(rootCtx, k, smap, key)

	go k.Run(rootCtx)

	// ... rest of runTUI unchanged (tea.Program, etc.) ...
```

Add the import `"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"` to the import block.

Add the helper function near the bottom of `main.go`:

```go
// persistTUISessionID mirrors internal/telegram/bot.persistIfChanged for the
// TUI's single-kernel case. Runs until ctx is canceled.
func persistTUISessionID(ctx context.Context, k *kernel.Kernel, smap session.Map, key string) {
	var lastSID string
	frames := k.Render()
	for {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-frames:
			if !ok {
				return
			}
			if f.SessionID == lastSID {
				continue
			}
			if err := smap.Put(ctx, key, f.SessionID); err != nil {
				slog.Warn("tui: failed to persist session_id",
					"key", key, "err", err)
				continue
			}
			lastSID = f.SessionID
		}
	}
}
```

**CRITICAL:** the TUI's tea.Program also consumes `k.Render()`. The existing code path (visible in the unchanged `// ... rest of runTUI unchanged ...` portion) already reads frames into the tea.Program. **Adding a second consumer on the same channel is a race/data-loss pattern** — only one of the two consumers will get each frame.

Read the existing `runTUI` body carefully. If it uses `k.Render()` directly, DO NOT add the `persistTUISessionID` goroutine above. Instead, extend whatever adapter layer bridges frames into tea.Program messages with the same three-line `persistIfChanged` check used in `internal/telegram/bot.go`.

If the TUI uses a fan-out helper already (there is likely a small function in `internal/tui/` that consumes `k.Render()` and emits `tea.Msg`), add the persistence hook INSIDE that helper. Do not create a duplicate consumer.

If in doubt, STOP and report as `NEEDS_CONTEXT` with the name of the TUI's render consumer.

- [ ] **Step 3: Build + vet + smoke**

```bash
cd gormes
go build ./...
go vet ./...
./bin/gormes --offline --help 2>&1 | grep -q resume && echo "resume flag registered" || echo "FAIL: resume flag not visible"
```

Expected: build succeeds; `--resume` appears in the help output.

- [ ] **Step 4: Full test sweep**

```bash
cd gormes
go test -race ./... -timeout 120s -count=1
```

Expected: all green.

- [ ] **Step 5: Commit**

```bash
cd ..
git add gormes/cmd/gormes/main.go
git commit -m "$(cat <<'EOF'
feat(gormes/cmd/tui): wire internal/session.BoltMap

cmd/gormes (TUI) now opens the shared sessions.db, honors a
new --resume cobra flag, and primes kernel.InitialSessionID.

Persistence of subsequent session-id updates uses whichever
existing render-consumer path the TUI already has — no duplicate
channel consumers, preserving the kernel's single-reader guarantee.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: End-to-end `TestBot_ResumesSessionIDAcrossRestart`

**Files:**
- Modify: `gormes/internal/telegram/bot_test.go`

- [ ] **Step 1: Write the failing test**

Append to `gormes/internal/telegram/bot_test.go`:

```go
// TestBot_ResumesSessionIDAcrossRestart proves the cross-phase invariant:
// a single session.MemMap carried across two bot+kernel lifecycles causes
// the second cycle's kernel to start with the first cycle's final
// session_id. Uses MemMap so there's no disk dependency.
func TestBot_ResumesSessionIDAcrossRestart(t *testing.T) {
	smap := session.NewMemMap()
	key := session.TelegramKey(42)

	// ── Cycle 1: run a turn that assigns session_id "sess-cycle-1"
	{
		mc := newMockClient()
		hmc := hermes.NewMockClient()
		hmc.Script([]hermes.Event{
			{Kind: hermes.EventToken, Token: "hi", TokensOut: 1},
			{Kind: hermes.EventDone, FinishReason: "stop"},
		}, "sess-cycle-1")

		k := kernel.New(kernel.Config{
			Model: "hermes-agent", Endpoint: "http://mock",
			Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
		}, hmc, store.NewNoop(), telemetry.New(), nil)

		b := New(Config{
			AllowedChatID: 42, CoalesceMs: 100,
			SessionMap: smap, SessionKey: key,
		}, mc, k, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		go k.Run(ctx)
		<-k.Render()
		go func() { _ = b.Run(ctx) }()

		mc.pushTextUpdate(42, "hi")

		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if got, _ := smap.Get(context.Background(), key); got == "sess-cycle-1" {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
		if got, _ := smap.Get(context.Background(), key); got != "sess-cycle-1" {
			t.Fatalf("cycle 1: map[%q] = %q, want sess-cycle-1", key, got)
		}
		cancel()
		mc.closeUpdates()
		time.Sleep(100 * time.Millisecond) // drain
	}

	// ── Cycle 2: new kernel, same map — InitialSessionID must be populated.
	{
		persistedSID, _ := smap.Get(context.Background(), key)
		if persistedSID != "sess-cycle-1" {
			t.Fatalf("cycle 2 precondition: persistedSID = %q, want sess-cycle-1", persistedSID)
		}

		hmc := hermes.NewMockClient()
		hmc.Script([]hermes.Event{
			{Kind: hermes.EventDone, FinishReason: "stop"},
		}, "sess-cycle-2")

		k := kernel.New(kernel.Config{
			Model: "hermes-agent", Endpoint: "http://mock",
			Admission:        kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
			InitialSessionID: persistedSID,
		}, hmc, store.NewNoop(), telemetry.New(), nil)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		go k.Run(ctx)
		<-k.Render()

		if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: "again"}); err != nil {
			t.Fatal(err)
		}

		// Verify the first outbound request carried the persisted session_id.
		waitForMockRequestWithSession := func(want string, d time.Duration) bool {
			deadline := time.Now().Add(d)
			for time.Now().Before(deadline) {
				for _, r := range hmc.Requests() {
					if r.SessionID == want {
						return true
					}
				}
				time.Sleep(25 * time.Millisecond)
			}
			return false
		}
		if !waitForMockRequestWithSession("sess-cycle-1", 2*time.Second) {
			t.Errorf("cycle 2: first request did not carry persisted session_id sess-cycle-1")
		}
	}
}
```

If `hermes.MockClient` does not expose `Requests() []ChatRequest`, adapt to whatever single-request accessor does exist (same note as Task 6 Step 1).

- [ ] **Step 2: Run — expect PASS**

```bash
cd gormes
go test -race ./internal/telegram/... -run TestBot_ResumesSessionIDAcrossRestart -v -timeout 30s
```

Expected: PASS. This test exercises the full wiring: persistIfChanged on cycle 1 → InitialSessionID on cycle 2 → request header carries the persisted id.

- [ ] **Step 3: Full telegram suite**

```bash
cd gormes
go test -race ./internal/telegram/... -timeout 60s -count=1
```

Expected: all green.

- [ ] **Step 4: Commit**

```bash
cd ..
git add gormes/internal/telegram/bot_test.go
git commit -m "$(cat <<'EOF'
test(gormes/telegram): Resume-across-restart invariant

TestBot_ResumesSessionIDAcrossRestart carries a single
session.MemMap across two bot+kernel lifecycles. Cycle 1
generates session_id "sess-cycle-1" and expects it written
to the map. Cycle 2 constructs a new kernel with
InitialSessionID drawn from the map and asserts the first
outbound Hermes request carries that id.

This is the Phase 2.C ship invariant: the whole point of the
feature is that restarts do not wipe the user's session.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Extend T6 build-isolation — kernel must not import session/bbolt

**Files:**
- Modify: `gormes/internal/buildisolation_test.go`

- [ ] **Step 1: Append the new test**

Append to `gormes/internal/buildisolation_test.go`:

```go
// TestKernelHasNoSessionDep guards the Phase 2.C boundary: internal/kernel
// must never transitively import internal/session or go.etcd.io/bbolt.
// If either appears in the kernel's dep graph, persistence has leaked into
// the turn-loop and the single-owner isolation is compromised.
func TestKernelHasNoSessionDep(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "./internal/kernel")
	cmd.Dir = ".." // run from gormes/
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list failed: %v\n%s", err, out.String())
	}

	for _, d := range strings.Split(out.String(), "\n") {
		if strings.Contains(d, "go.etcd.io/bbolt") ||
			strings.Contains(d, "/internal/session") {
			t.Errorf("internal/kernel transitively depends on %q — Phase 2.C isolation violated", d)
		}
	}
}
```

- [ ] **Step 2: Run — expect PASS**

```bash
cd gormes
go test -race ./internal/ -run TestKernelHasNoSessionDep -v
```

Expected: PASS. The kernel does not import session or bbolt (Task 6 only added a plain `string` field; no new imports).

- [ ] **Step 3: Sanity-break**

Temporarily add `_ "github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"` to `gormes/internal/kernel/kernel.go`'s imports. Re-run the test. Expected: FAIL naming `session` AND `bbolt` (since session imports bbolt).

**Revert the import change** and re-run — PASS again.

- [ ] **Step 4: Commit**

```bash
cd ..
git add gormes/internal/buildisolation_test.go
git commit -m "$(cat <<'EOF'
test(gormes/internal): forbid session/bbolt in the kernel dep graph

TestKernelHasNoSessionDep runs `go list -deps ./internal/kernel`
and fails if any line contains `go.etcd.io/bbolt` or
`/internal/session`. Locks in the Phase 2.C architectural
boundary — adapters own the disk, the kernel never sees it.

Verified by temporarily adding a blank session import to
kernel.go and confirming the test FAILs with a clear offender,
then reverting.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Verification sweep

**Files:** no changes — verification only.

- [ ] **Step 1: Full test sweep**

```bash
cd gormes
go test -race ./... -timeout 120s -count=1
go vet ./...
```

Expected: all packages PASS; `vet` clean.

- [ ] **Step 2: Build both binaries + size check**

```bash
cd gormes
make build
go build -o bin/gormes-telegram ./cmd/gormes-telegram
ls -lh bin/
```

Expected:
- `bin/gormes` ≤ **10 MB** stripped (target ~8.4 MB, pre-Phase-2.C was 7.9 MB)
- `bin/gormes-telegram` ≤ **12 MB** stripped (target ~10.0 MB, pre-Phase-2.C was 9.5 MB)

If either binary exceeds its budget, STOP and report sizes. Do NOT proceed.

- [ ] **Step 3: Build-isolation grep verification**

```bash
cd gormes
(go list -deps ./cmd/gormes | grep -E "telegram-bot-api|internal/telegram") && echo "VIOLATION: TUI has telegram deps" || echo "OK: TUI clean"
(go list -deps ./internal/kernel | grep -E "go.etcd.io/bbolt|internal/session") && echo "VIOLATION: kernel has persistence deps" || echo "OK: kernel clean"
```

Expected: both lines print `OK`.

- [ ] **Step 4: Offline doctor still works**

```bash
cd gormes
./bin/gormes doctor --offline
```

Expected: `[PASS] Toolbox: 3 tools registered (echo, now, rand_int)`.

- [ ] **Step 5: TUI startup smoke (--offline)**

```bash
cd gormes
# Non-interactive smoke — just verify the binary starts, registers the
# --resume flag, and exits cleanly on --help.
./bin/gormes --help | grep -q -- --resume && echo "flag visible" || echo "FAIL"
```

Expected: `flag visible`.

- [ ] **Step 6: Bot startup smoke**

```bash
cd gormes
./bin/gormes-telegram 2>&1 || true
```

Expected: exits 1 with `"no Telegram bot token — ..."` (unchanged from Phase 2.B.1).

- [ ] **Step 7: Sessions.db disk test (manual)**

```bash
cd gormes
export XDG_DATA_HOME=/tmp/gormes-smoke-$$
GORMES_TELEGRAM_TOKEN=fake:token GORMES_TELEGRAM_CHAT_ID=99 \
  timeout 2 ./bin/gormes-telegram 2>&1 || true
ls -la $XDG_DATA_HOME/gormes/sessions.db && echo "DB file created"
rm -rf $XDG_DATA_HOME
```

Expected: `sessions.db` exists at the XDG path, file mode 0600. The timeout fires because the binary cannot actually connect with a fake token; what matters is that `OpenBolt` ran successfully and created the file.

- [ ] **Step 8: Live manual smoke test (optional)**

See spec §13 for the full `GORMES_TELEGRAM_TOKEN=real kill -TERM → restart → replay` flow. This is a release-gate manual test; do not block Phase 2.C completion on it unless a live Telegram bot is available.

- [ ] **Step 9: No commit**

This task runs only verifications. If any check fails, STOP and report with the failing command + output. Phase 2.C lands when Steps 1–7 all pass.

---

## Appendix: Self-Review

**Spec coverage:**

| Spec § | Task(s) |
|---|---|
| §1 Goal | All tasks |
| §2 Non-goals | Enforced by task scope (no history/token tasks in plan) |
| §3 Scope | Tasks 2–11 |
| §4 Architecture | Tasks 6 (kernel seam), 7 (bot hook), 9 (telegram cmd), 10 (tui cmd) |
| §5 Data model (bucket, key encoding, value encoding, file location) | Tasks 2 (keys), 3 (bucket + file), 4 (delete semantics) |
| §6 Interface (Map, BoltMap, MemMap, Kernel seam, adapter hooks) | Tasks 2 (Map+MemMap), 3 (BoltMap), 6 (kernel), 7 (bot), 10 (TUI) |
| §7 Error handling (lock, corrupt, permission, startup Warn, runtime Warn) | Tasks 3 (wrapping), 5 (failure-mode tests), 7 (runtime Warn), 9/10 (startup Warn) |
| §8 --resume CLI | Task 8 (flag) + Task 9 (bot wiring) + Task 10 (tui wiring) |
| §9 Dep posture, binary budgets | Task 1 + Task 13 |
| §10 Security (file modes) | Task 3 (MkdirAll 0o700, Open 0o600); spot-checked in Task 13 Step 7 |
| §11 Testing (unit, adapter, failure, e2e, isolation, sweep) | Tasks 2, 3, 4, 5, 7, 11, 12, 13 |
| §12 Verification checklist | Task 13 Steps 1–7 |
| §13 Manual smoke | Task 13 Step 8 |
| §14 Out of scope | No tasks (correctly) |
| §15 Rollout (single-commit series, zero-regression first boot) | Task 1 through Task 12 are all separate commits; first-boot zero-regression is structurally guaranteed (OpenBolt creates empty file → Get returns `""` → kernel starts fresh, identical to pre-Phase-2.C). |

**Placeholder scan:** zero `TBD` / `TODO` / `fill in` / `similar to Task N` / `add appropriate error handling`. Task 10 Step 2's CRITICAL note is deliberate prose about an integration risk — not a placeholder; it tells the implementer exactly what to do (read the existing runTUI, find the render consumer, inline the hook there).

**Type consistency:**
- `session.Map` (interface) — used in Tasks 2, 6 (copy into kernel), 7 (bot Config), 9, 10. Consistent.
- `session.BoltMap`, `session.MemMap`, `session.NewMemMap`, `session.OpenBolt` — named consistently across Tasks 2, 3, 7, 9, 10, 11.
- `session.ErrDBLocked`, `session.ErrDBCorrupt` — declared Task 2, asserted Task 5. Consistent.
- `session.TUIKey()`, `session.TelegramKey(int64)` — declared Task 2, used Tasks 9, 10, 11. Consistent.
- `kernel.Config.InitialSessionID` — introduced Task 6, consumed Tasks 9, 10, 11. Consistent.
- `telegram.Config.SessionMap`, `telegram.Config.SessionKey` — introduced Task 7, consumed Tasks 9, 11. Consistent.
- `config.SessionDBPath()` — introduced Task 8, consumed Tasks 9, 10, 13. Consistent.
- `config.Resume` — introduced Task 8, consumed Tasks 9, 10. Consistent.
- Bucket name `sessions_v1` — Task 3 only. One source of truth.

**Execution order:** linear dependency — Task N depends on Task N-1 for at least one symbol. Recommended sequence: **T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T9 → T10 → T11 → T12 → T13**.

**Checkpoint suggestion:** halt after T6 (kernel surface) for user sanity-check of the seam change, and after T10 (both cmds wired) before T11's cross-cycle integration test.

**Scope:** one cohesive Phase-2.C plan — persistence seam + adapter wiring + CLI + verification. Binary, interfaces, tests, lifecycle all self-contained. No spill into Phase 3 (no token counting, no FTS, no history mirror).
