// Package memory provides the Phase 3.D.5 Memory Mirror.
//
// The Mirror is an async background goroutine that exports entities and
// relationships from SQLite to a human-readable Markdown file (USER.md).
// SQLite remains the source of truth; the mirror is a read-only sync target
// for operator auditability.
//
// Design constraints:
//   - Never touches the 250ms kernel latency budget (fire-and-forget)
//   - Failures log warnings but do not crash the bot
//   - Configurable path (respects memory.mirror_path)
//   - Default sync interval: 30s
//
// See gormes/docs/superpowers/specs/2026-04-20-gormes-phase3d5-mirror-design.md.
package memory

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// MirrorConfig controls the background sync behavior.
type MirrorConfig struct {
	// Enabled turns the mirror on/off. Default: true.
	Enabled bool
	// Path is the destination Markdown file. Default: "~/.hermes/memory/USER.md"
	Path string
	// Interval is the sync period. Default: 30s.
	Interval time.Duration
	// Logger receives mirror lifecycle events. If nil, slog.Default() is used.
	Logger *slog.Logger
}

// DefaultMirrorConfig returns production defaults.
func DefaultMirrorConfig() MirrorConfig {
	home, _ := os.UserHomeDir()
	return MirrorConfig{
		Enabled:  true,
		Path:     filepath.Join(home, ".hermes", "memory", "USER.md"),
		Interval: 30 * time.Second,
		Logger:   nil,
	}
}

// Mirror manages the background USER.md sync.
type Mirror struct {
	cfg       MirrorConfig
	log       *slog.Logger
	store     *SqliteStore
	ticker    *time.Ticker
	stop      chan struct{}
	stopOnce  sync.Once
	wg        sync.WaitGroup
	lastHash  string // content hash to avoid redundant writes
	lastHashMu sync.Mutex
}

// StartMirror spawns the background sync goroutine. If cfg.Enabled is false,
// this is a no-op and returns nil. The caller must hold a concrete *SqliteStore.
func StartMirror(store *SqliteStore, cfg MirrorConfig) *Mirror {
	if !cfg.Enabled {
		return nil
	}
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o700); err != nil {
		log.Warn("mirror: failed to create parent directory",
			"path", cfg.Path,
			"error", err)
		// Continue anyway — the first sync will fail and warn
	}

	m := &Mirror{
		cfg:    cfg,
		log:    log,
		store:  store,
		ticker: time.NewTicker(cfg.Interval),
		stop:   make(chan struct{}),
	}

	m.wg.Add(1)
	go m.loop()
	m.log.Info("mirror: started", "path", cfg.Path, "interval", cfg.Interval)
	return m
}

// Stop signals the mirror to exit and waits for the goroutine to finish.
// Idempotent — safe to call multiple times.
func (m *Mirror) Stop() {
	if m == nil {
		return
	}
	m.stopOnce.Do(func() {
		close(m.stop)
		m.ticker.Stop()
		m.wg.Wait()
		m.log.Info("mirror: stopped")
	})
}

// loop runs until Stop() is called. It performs an immediate first sync,
// then waits on the ticker.
func (m *Mirror) loop() {
	defer m.wg.Done()

	// Immediate first sync
	m.sync()

	for {
		select {
		case <-m.ticker.C:
			m.sync()
		case <-m.stop:
			return
		}
	}
}

// sync performs one round-trip: query SQLite, render Markdown, write file.
// Errors are logged as warnings; the bot never crashes.
func (m *Mirror) sync() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch entities
	entities, err := m.queryEntities(ctx)
	if err != nil {
		m.log.Warn("mirror: failed to query entities", "error", err)
		return
	}

	// Fetch relationships
	rels, err := m.queryRelationships(ctx)
	if err != nil {
		m.log.Warn("mirror: failed to query relationships", "error", err)
		return
	}

	// Render Markdown
	content := m.renderMarkdown(entities, rels)

	// Compute hash to avoid redundant writes
	hash := hashContent(content)
	m.lastHashMu.Lock()
	if hash == m.lastHash {
		m.lastHashMu.Unlock()
		return // no change
	}
	m.lastHash = hash
	m.lastHashMu.Unlock()

	// Write atomically (tmp + rename)
	if err := m.writeAtomic(content); err != nil {
		m.log.Warn("mirror: failed to write USER.md", "path", m.cfg.Path, "error", err)
		return
	}

	m.log.Debug("mirror: synced",
		"entities", len(entities),
		"relationships", len(rels),
		"path", m.cfg.Path)
}

type entity struct {
	ID          int64
	Name        string
	Type        string
	Description string
}

type relationship struct {
	SourceName string
	Predicate  string
	TargetName string
	Weight     float64
}

func (m *Mirror) queryEntities(ctx context.Context) ([]entity, error) {
	rows, err := m.store.db.QueryContext(ctx, `
		SELECT id, name, type, description
		FROM entities
		ORDER BY type, name
	`)
	if err != nil {
		return nil, fmt.Errorf("query entities: %w", err)
	}
	defer rows.Close()

	var entities []entity
	for rows.Next() {
		var e entity
		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.Description); err != nil {
			return nil, fmt.Errorf("scan entity: %w", err)
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (m *Mirror) queryRelationships(ctx context.Context) ([]relationship, error) {
	rows, err := m.store.db.QueryContext(ctx, `
		SELECT e1.name, r.predicate, e2.name, r.weight
		FROM relationships r
		JOIN entities e1 ON r.source_id = e1.id
		JOIN entities e2 ON r.target_id = e2.id
		ORDER BY e1.name, r.predicate, e2.name
	`)
	if err != nil {
		return nil, fmt.Errorf("query relationships: %w", err)
	}
	defer rows.Close()

	var rels []relationship
	for rows.Next() {
		var r relationship
		if err := rows.Scan(&r.SourceName, &r.Predicate, &r.TargetName, &r.Weight); err != nil {
			return nil, fmt.Errorf("scan relationship: %w", err)
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

// renderMarkdown produces the Hermes-compatible USER.md format.
// Groups entities by type, lists relationships with weights.
func (m *Mirror) renderMarkdown(entities []entity, rels []relationship) string {
	var b strings.Builder

	// Header
	b.WriteString("# Memory Export (Gormes)\n\n")
	b.WriteString(fmt.Sprintf("Last synced: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString("## Overview\n\n")
	b.WriteString(fmt.Sprintf("- **Total entities:** %d\n", len(entities)))
	b.WriteString(fmt.Sprintf("- **Total relationships:** %d\n\n", len(rels)))

	// Entities by type
	if len(entities) > 0 {
		b.WriteString("## Entities\n\n")

		// Group by type
		byType := make(map[string][]entity)
		for _, e := range entities {
			byType[e.Type] = append(byType[e.Type], e)
		}

		// Sort types for determinism
		types := make([]string, 0, len(byType))
		for t := range byType {
			types = append(types, t)
		}
		sort.Strings(types)

		for _, typ := range types {
			b.WriteString(fmt.Sprintf("### %s (%d)\n\n", typ, len(byType[typ])))
			for _, e := range byType[typ] {
				b.WriteString(fmt.Sprintf("- **%s**", e.Name))
				if e.Description != "" {
					b.WriteString(fmt.Sprintf(": %s", e.Description))
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	// Relationships
	if len(rels) > 0 {
		b.WriteString("## Relationships\n\n")
		b.WriteString(fmt.Sprintf("(%d total)\n\n", len(rels)))

		for _, r := range rels {
			b.WriteString(fmt.Sprintf("- **%s** → [%s] → **%s**", r.SourceName, r.Predicate, r.TargetName))
			if r.Weight != 1.0 {
				b.WriteString(fmt.Sprintf(" (weight: %.2f)", r.Weight))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("---\n\n")
	b.WriteString("*This file is auto-generated from the Gormes memory store. " +
		"Edits here will be overwritten on next sync. SQLite is the source of truth.*\n")

	return b.String()
}

// writeAtomic writes content to a temp file, then renames it to the target.
// This provides atomic updates (readers never see a partial file).
func (m *Mirror) writeAtomic(content string) error {
	dir := filepath.Dir(m.cfg.Path)
	base := filepath.Base(m.cfg.Path)
	tmp := filepath.Join(dir, "."+base+".tmp")

	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}

	if err := os.Rename(tmp, m.cfg.Path); err != nil {
		os.Remove(tmp) // best-effort cleanup
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// hashContent produces a simple hash for change detection.
// Using a simple checksum approach — not cryptographic, just for deduplication.
func hashContent(s string) string {
	// FNV-1a inspired simple hash
	var h uint64 = 0xcbf29ce484222325 // FNV offset basis
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 0x100000001b3 // FNV prime
	}
	return fmt.Sprintf("%016x", h)
}
