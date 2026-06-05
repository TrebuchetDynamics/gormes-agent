// Package memory provides the Phase 3.D.5 Memory Mirror.
//
// The Mirror is an async background goroutine that exports entities and
// relationships from SQLite to a human-readable Markdown file (USER.md).
// SQLite remains the source of truth; the mirror is a read-only sync target
// for operator auditability.
package memory

import "github.com/TrebuchetDynamics/gormes-agent/internal/memory/mirror"

// MirrorConfig controls the background sync behavior.
type MirrorConfig = mirror.MirrorConfig

// DefaultMirrorConfig returns production defaults.
func DefaultMirrorConfig() MirrorConfig {
	return mirror.DefaultMirrorConfig()
}

// Mirror manages the background USER.md sync.
type Mirror = mirror.Mirror

// StartMirror spawns the background sync goroutine. If cfg.Enabled is false,
// this is a no-op and returns nil. The caller must hold a concrete *SqliteStore.
func StartMirror(store *SqliteStore, cfg MirrorConfig) *Mirror {
	return mirror.StartMirror(store.db, cfg)
}
