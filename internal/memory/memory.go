// Package memory is the SQLite-backed Phase-3.A Lattice Foundation.
// It implements store.Store with fire-and-forget semantics: Exec returns
// an Ack in microseconds after enqueueing a Command on a bounded channel;
// a single-owner background worker performs all SQL I/O. On queue-full:
// log + drop. See docs/superpowers/specs/2026-04-20-gormes-phase3a-memory-design.md.
package memory

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
)

// defaultQueueCap is used when OpenSqlite receives queueCap <= 0.
const defaultQueueCap = 1024

// Stats exposes counters for tests and future telemetry. Not part of
// store.Store — consumers must hold a concrete *SqliteStore to call Stats.
type Stats struct {
	QueueLen int
	QueueCap int
	Drops    uint64
	Accepted uint64
}

// SqliteStore is a fire-and-forget store.Store backed by SQLite + FTS5.
type SqliteStore struct {
	db    *sql.DB
	queue chan store.Command
	done  chan struct{}
	log   *slog.Logger

	drops    atomic.Uint64
	accepted atomic.Uint64

	closeOnce sync.Once
	mirror    *Mirror // Phase 3.D.5: optional background USER.md sync
}

// Compile-time interface check.
var _ store.Store = (*SqliteStore)(nil)

// OpenSqlite opens/creates the SQLite file at path, applies the schema,
// and starts the background worker goroutine. queueCap <= 0 falls back
// to defaultQueueCap. log == nil falls back to slog.Default().
func OpenSqlite(path string, queueCap int, log *slog.Logger) (*SqliteStore, error) {
	if log == nil {
		log = slog.Default()
	}
	if queueCap <= 0 {
		queueCap = defaultQueueCap
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("memory: create parent dir for %s: %w", path, err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("memory: open %s: %w", path, err)
	}
	// Single writer connection (ncruces/go-sqlite3 WASM: each connection owns
	// its own WASM memory; one connection keeps the footprint minimal and
	// matches our single-owner worker goroutine anyway).
	db.SetMaxOpenConns(1)

	if err := applyPragmas(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("memory: pragmas: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	s := &SqliteStore{
		db:    db,
		queue: make(chan store.Command, queueCap),
		done:  make(chan struct{}, 1),
		log:   log,
	}
	go s.run()
	return s, nil
}

func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 2000",
		"PRAGMA foreign_keys = ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}
	return nil
}

// Exec enqueues cmd on the worker queue. Returns an Ack in microseconds.
// On queue full: increments Drops counter, logs a WARN, and returns
// Ack{} — the caller cannot tell the difference. This is the deliberate
// Zero-Leak design: a dropped turn is acceptable degradation; a blocked
// kernel is not.
func (s *SqliteStore) Exec(ctx context.Context, cmd store.Command) (store.Ack, error) {
	if err := ctx.Err(); err != nil {
		return store.Ack{}, err
	}
	select {
	case s.queue <- cmd:
		s.accepted.Add(1)
	default:
		s.drops.Add(1)
		s.log.Warn("memory: queue full, dropping command",
			"kind", cmd.Kind.String(),
			"queue_cap", cap(s.queue),
			"drops_total", s.drops.Load())
	}
	return store.Ack{}, nil
}

// Stats returns a snapshot of worker counters.
func (s *SqliteStore) Stats() Stats {
	return Stats{
		QueueLen: len(s.queue),
		QueueCap: cap(s.queue),
		Drops:    s.drops.Load(),
		Accepted: s.accepted.Load(),
	}
}

// DB returns the underlying *sql.DB handle. Exposed for read-only test
// verification; production callers should not depend on this.
func (s *SqliteStore) DB() *sql.DB { return s.db }

// StartMirror enables the Phase 3.D.5 background USER.md sync. Safe to call
// multiple times (idempotent). If cfg.Enabled is false, this is a no-op.
func (s *SqliteStore) StartMirror(cfg MirrorConfig) {
	if s.mirror != nil {
		s.mirror.Stop() // restart with new config
	}
	s.mirror = StartMirror(s, cfg)
}

// StopMirror halts the background USER.md sync. Safe to call multiple times.
func (s *SqliteStore) StopMirror() {
	if s.mirror != nil {
		s.mirror.Stop()
		s.mirror = nil
	}
}

// Close signals the worker to drain, waits up to ctx deadline for drain,
// then closes the underlying *sql.DB (which flushes WAL). Idempotent —
// subsequent calls return nil.
func (s *SqliteStore) Close(ctx context.Context) error {
	var closeErr error
	s.closeOnce.Do(func() {
		s.StopMirror() // Phase 3.D.5: stop background sync before DB close
		close(s.queue) // signal worker to exit after draining
		select {
		case <-s.done:
			// drained cleanly
		case <-ctx.Done():
			s.log.Warn("memory: shutdown deadline exceeded; in-flight writes may be lost",
				"queue_len", len(s.queue))
		}
		closeErr = s.db.Close()
	})
	return closeErr
}
