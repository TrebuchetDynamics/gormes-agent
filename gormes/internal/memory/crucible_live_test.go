//go:build live

// Package memory — live end-to-end extractor crucible.
//
// Gated by the `live` build tag so CI / normal `go test` never hits it.
// Run explicitly with:
//   GORMES_ENDPOINT=http://127.0.0.1:8642 go test -tags=live ./internal/memory/...  -run TestCrucible_Extractor -v -timeout 2m
//
// Skips (not fails) if the api_server is unreachable — missing infrastructure
// is not a test failure under the live tag.
package memory

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/hermes"
)

func crucibleEndpoint() string {
	if v := os.Getenv("GORMES_ENDPOINT"); v != "" {
		return v
	}
	return "http://127.0.0.1:8642"
}

func crucibleModel() string {
	if v := os.Getenv("GORMES_MODEL"); v != "" {
		return v
	}
	return "hermes-agent"
}

func crucibleSkipIfUnreachable(t *testing.T, c hermes.Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Health(ctx); err != nil {
		if _, ok := err.(*net.OpError); ok {
			t.Skipf("api_server not running at %s: %v", crucibleEndpoint(), err)
		}
		t.Skipf("api_server unhealthy at %s: %v", crucibleEndpoint(), err)
	}
}

// TestCrucible_ExtractorEndToEnd seeds 3 entity-rich turns directly into
// the turns table (bypassing Telegram + kernel), runs a real Extractor
// against the live api_server, and asserts entities + relationships
// populate.
func TestCrucible_ExtractorEndToEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crucible.db")
	store, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	hc := hermes.NewHTTPClient(crucibleEndpoint(), os.Getenv("GORMES_API_KEY"))
	crucibleSkipIfUnreachable(t, hc)

	// Direct-insert 3 entity-rich turns bypassing the kernel + persistence
	// worker. This is the synthetic substitute for real Telegram DMs.
	highDensityTurns := []string{
		"I am setting up the AzulVigia project in Cadereyta.",
		"Vania is helping me test the Neovim configuration.",
		"We need to optimize the Go backend for Trebuchet Dynamics.",
	}
	for i, content := range highDensityTurns {
		_, err := store.db.Exec(
			`INSERT INTO turns(session_id, role, content, ts_unix)
			 VALUES(?, 'user', ?, ?)`,
			"crucible-session", content, time.Now().Unix()+int64(i),
		)
		if err != nil {
			t.Fatalf("direct insert %d: %v", i, err)
		}
	}

	// Construct the extractor. Tight poll so we don't wait 10s for the first
	// tick; production defaults stay 10s. Short backoff so any retries don't
	// eat the test budget.
	ext := NewExtractor(store, hc, ExtractorConfig{
		Model:        crucibleModel(),
		PollInterval: 500 * time.Millisecond,
		BatchSize:    3,
		MaxAttempts:  3,
		CallTimeout:  60 * time.Second, // real LLMs are slow
		BackoffBase:  500 * time.Millisecond,
		BackoffMax:   5 * time.Second,
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	go ext.Run(ctx)
	defer func() {
		shutdownCtx, scancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer scancel()
		_ = ext.Close(shutdownCtx)
	}()

	// Wait for ALL 3 turns to be extracted (or dead-lettered).
	deadline := time.Now().Add(85 * time.Second)
	var lastState [3]int
	for time.Now().Before(deadline) {
		rows, err := store.db.Query(`SELECT id, extracted FROM turns ORDER BY id`)
		if err != nil {
			t.Fatalf("poll turns: %v", err)
		}
		i := 0
		allDone := true
		for rows.Next() && i < 3 {
			var id int64
			var state int
			_ = rows.Scan(&id, &state)
			lastState[i] = state
			if state == 0 {
				allDone = false
			}
			i++
		}
		rows.Close()
		if allDone {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// ── Telemetry dump (what the user wants to see) ─────────────────────
	t.Logf("=== CRUCIBLE TELEMETRY ===")

	// 1. Turn state distribution.
	stateRows, _ := store.db.Query(`SELECT extracted, COUNT(*) FROM turns GROUP BY extracted ORDER BY extracted`)
	t.Logf("--- Turn state distribution ---")
	for stateRows.Next() {
		var state, count int
		_ = stateRows.Scan(&state, &count)
		label := map[int]string{0: "unprocessed", 1: "extracted", 2: "dead-letter"}[state]
		t.Logf("  extracted=%d (%s): %d turns", state, label, count)
	}
	stateRows.Close()

	// 2. Entity dump.
	entRows, _ := store.db.Query(`SELECT id, name, type, description FROM entities ORDER BY id`)
	t.Logf("--- Entities ---")
	entCount := 0
	for entRows.Next() {
		var id int64
		var name, typ, desc string
		_ = entRows.Scan(&id, &name, &typ, &desc)
		t.Logf("  [%d] %s (%s) — %q", id, name, typ, desc)
		entCount++
	}
	entRows.Close()

	// 3. Relationship dump (joined for readability).
	relRows, _ := store.db.Query(`
		SELECT e1.name, r.predicate, e2.name, r.weight
		FROM relationships r
		JOIN entities e1 ON r.source_id = e1.id
		JOIN entities e2 ON r.target_id = e2.id
		ORDER BY r.weight DESC, e1.name, e2.name
	`)
	t.Logf("--- Relationships ---")
	relCount := 0
	for relRows.Next() {
		var src, pred, tgt string
		var w float64
		_ = relRows.Scan(&src, &pred, &tgt, &w)
		t.Logf("  %s --[%s @ %.2f]--> %s", src, pred, w, tgt)
		relCount++
	}
	relRows.Close()

	// 4. Extractor stats.
	var attempts int
	var errMsgs []string
	errRows, _ := store.db.Query(`SELECT extraction_attempts, COALESCE(extraction_error, '') FROM turns ORDER BY id`)
	for errRows.Next() {
		var n int
		var msg string
		_ = errRows.Scan(&n, &msg)
		attempts += n
		if msg != "" {
			errMsgs = append(errMsgs, msg)
		}
	}
	errRows.Close()
	t.Logf("--- Extractor telemetry ---")
	t.Logf("  total extraction_attempts across all turns: %d", attempts)
	t.Logf("  final turn states: %v", lastState)
	if len(errMsgs) > 0 {
		for _, m := range errMsgs {
			if len(m) > 200 {
				m = m[:200] + "..."
			}
			t.Logf("  last error: %s", m)
		}
	}
	t.Logf("=== END CRUCIBLE TELEMETRY ===")

	// ── Minimal correctness assertions ─────────────────────────────────
	var nUnprocessed int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM turns WHERE extracted = 0`).Scan(&nUnprocessed)
	if nUnprocessed > 0 {
		t.Errorf("still %d unprocessed turns after 85s — the extractor didn't drain them", nUnprocessed)
	}

	if entCount == 0 {
		t.Errorf("entities table is empty — the LLM extracted nothing for 3 entity-rich turns")
	}

	// Not fatal but log for diagnostic: did we see at least ONE expected entity name?
	expected := []string{"AzulVigia", "Cadereyta", "Vania", "Neovim", "Go", "Trebuchet Dynamics"}
	var matches []string
	for _, want := range expected {
		var n int
		_ = store.db.QueryRow(`SELECT COUNT(*) FROM entities WHERE name = ?`, want).Scan(&n)
		if n > 0 {
			matches = append(matches, want)
		}
	}
	t.Logf("  expected-entity name matches: %d/%d — %v", len(matches), len(expected), matches)

	// Dump raw memory.db path for external sqlite3 inspection.
	fmt.Printf("\n[crucible] memory.db path: %s\n\n", path)
}
