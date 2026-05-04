# Planner Divergence Awareness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the architecture-planner-loop to handle the porting→native epistemology shift via per-row Provenance, per-subphase DriftState, reverse impl-tree research, a reactive impl-change trigger, and a drift status surface.

**Architecture:** Five layers added to `internal/architectureplanner/` and `internal/progress/`. New typed blocks `Provenance` (on Item) and `DriftState` (on Subphase) — planner-owned, autoloop-preserves-structurally (Phase B's typed round-trip handles it). New module `implscan.go` walks `cmd/+internal/`. New systemd path unit watches the impl tree.

**Tech Stack:** Go 1.25+, `os.Stat` mtime walking, systemd path units, the existing typed-struct round-trip from Phase B.

**Reference spec:** `docs/superpowers/specs/2026-04-25-planner-divergence-awareness-design.md`

**Baseline commit (spec):** `de3b0c76`

---

## File Structure

**New files:**

```text
internal/architectureplanner/implscan.go
internal/architectureplanner/implscan_test.go
internal/architectureplanner/divergence_lifecycle_test.go
```

**Modified files:**

```text
internal/progress/health.go
internal/progress/progress.go
internal/progress/health_test.go
internal/progress/preservation_test.go
internal/architectureplanner/context.go
internal/architectureplanner/config.go
internal/architectureplanner/config_test.go
internal/architectureplanner/prompt.go
internal/architectureplanner/prompt_test.go
internal/architectureplanner/run.go
internal/architectureplanner/run_test.go
internal/architectureplanner/service.go
internal/architectureplanner/service_test.go
internal/architectureplanner/ledger.go
internal/architectureplanner/status.go
internal/architectureplanner/status_test.go
```

**Responsibility map:**

- `internal/progress/health.go`: add `Provenance` and `DriftState` types alongside `RowHealth`/`PlannerVerdict`.
- `internal/progress/progress.go`: add `Item.Provenance *Provenance` (last field, after PlannerVerdict); add `Subphase.DriftState *DriftState` (last field of Subphase).
- `internal/progress/preservation_test.go`: extend symmetric preservation suite to cover all 4 typed blocks.
- `internal/architectureplanner/implscan.go`: `ScanImplementation`, `ImplInventory` type, deny-list path matching, mtime lookback.
- `internal/architectureplanner/context.go`: add `ContextBundle.ImplInventory ImplInventory`; thread through `CollectContext`.
- `internal/architectureplanner/config.go`: add `Config.GormesOriginalPaths []string` (env `PLANNER_GORMES_ORIGINAL_PATHS`); `Config.ImplLookback time.Duration` (env `PLANNER_IMPL_LOOKBACK`, default 24h); `Config.TriggerReason string` (env `PLANNER_TRIGGER_REASON`).
- `internal/architectureplanner/prompt.go`: add `PROVENANCE AWARENESS` + `DRIFT STATE` always-on soft clauses; render `## Implementation Inventory` section when bundle has ImplInventory.
- `internal/architectureplanner/run.go`: call ScanImplementation + thread into bundle; honor TriggerReason for `Trigger="impl_change"`; compute and emit DriftPromotions in LedgerEvent.
- `internal/architectureplanner/service.go`: add `RenderPlannerImplPathUnit` + extend `InstallPlannerService` to write the third .path file.
- `internal/architectureplanner/ledger.go`: add `LedgerEvent.DriftPromotions []DriftPromotion` field + `DriftPromotion` type.
- `internal/architectureplanner/status.go`: extend `RenderStatus` to bucket subphases by DriftState and show recent promotions from the ledger.

---

## Conventions Used In Every Task

- Each task is one TDD cycle ending in one commit.
- Always run failing test first; never write impl before red test.
- Run `go vet ./...` and `gofmt -l .` for the touched packages before commit.
- Run focused-then-wider test suite before each commit.
- Commit message format: `feat(progress): ...` / `feat(planner): ...` / `test(planner): ...`.
- Never modify field ordering of existing structs. New `Item.Provenance` is appended after Phase C's `PlannerVerdict`. New `Subphase.DriftState` is appended at the end of `Subphase`.
- All new env vars default to current behavior (back-compat).
- All file IO that writes `progress.json` goes through `internal/progress.SaveProgress` (Phase B/C). No new bypass path.

---

## Task 1: Add `Provenance` + `DriftState` Schema

**Files:**
- Modify: `internal/progress/health.go`
- Modify: `internal/progress/progress.go`
- Modify: `internal/progress/health_test.go`
- Modify: `internal/progress/preservation_test.go`

- [ ] **Step 1.1: Write failing tests**

Append to `internal/progress/health_test.go`:

```go
func TestProvenance_RoundTrip(t *testing.T) {
	p := &Provenance{
		OriginType:  "gormes",
		UpstreamRef: "",
		OwnedSince:  "2026-04-25T00:00:00Z",
		Note:        "autoloop is Gormes-original",
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Provenance
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.OriginType != "gormes" || got.OwnedSince != "2026-04-25T00:00:00Z" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestProvenance_OmitemptyKeepsZeroFieldsOut(t *testing.T) {
	p := &Provenance{OriginType: "upstream"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `{"origin_type":"upstream"}` {
		t.Fatalf("expected origin_type only, got %s", data)
	}
}

func TestDriftState_RoundTrip(t *testing.T) {
	d := &DriftState{
		Status:            "owned",
		LastUpstreamCheck: "2026-04-24T12:00:00Z",
		OriginDecision:    "Gormes invented L2 trigger ledger; no upstream analog",
	}
	data, _ := json.Marshal(d)
	var got DriftState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != "owned" {
		t.Fatalf("Status = %q, want owned", got.Status)
	}
}

func TestItem_ProvenanceOmitemptyByDefault(t *testing.T) {
	item := &Item{Name: "x", Status: StatusPlanned}
	data, _ := json.Marshal(item)
	if strings.Contains(string(data), "provenance") {
		t.Fatalf("Item with no provenance should not emit key, got %s", data)
	}
}

func TestSubphase_DriftStateOmitemptyByDefault(t *testing.T) {
	sub := Subphase{Name: "S"}
	data, _ := json.Marshal(sub)
	if strings.Contains(string(data), "drift_state") {
		t.Fatalf("Subphase with no drift_state should not emit key, got %s", data)
	}
}
```

- [ ] **Step 1.2: Run failing tests**

```bash
cd /home/xel/git/sages-openclaw/workspace-mineru/gormes-agent
go test ./internal/progress/ -run 'TestProvenance|TestDriftState|TestItem_ProvenanceOmitemptyByDefault|TestSubphase_DriftStateOmitemptyByDefault' -v
```
Expected: FAIL because types and fields don't exist.

- [ ] **Step 1.3: Add types in `internal/progress/health.go`**

Append (alongside RowHealth and PlannerVerdict):

```go
// Provenance is per-row source-of-truth metadata. OWNED by the planner;
// autoloop preserves it via typed-struct round-trip (Phase B). The planner
// sets origin_type="gormes" for rows that have no upstream analog.
type Provenance struct {
	OriginType  string `json:"origin_type"`            // "upstream" | "gormes" | "hybrid"
	UpstreamRef string `json:"upstream_ref,omitempty"` // e.g. "hermes:gateway/api_server.py@abc123"
	OwnedSince  string `json:"owned_since,omitempty"`  // RFC3339 when origin_type became "gormes"
	Note        string `json:"note,omitempty"`
}

// DriftState is subphase-level convergence state. OWNED by the planner.
// Status is a one-way ratchet: porting → converged → owned. Never
// auto-demoted by planner code; humans can demote by direct edit.
type DriftState struct {
	Status            string `json:"status"`                          // "porting" | "converged" | "owned"
	LastUpstreamCheck string `json:"last_upstream_check,omitempty"`   // RFC3339
	OriginDecision    string `json:"origin_decision,omitempty"`       // human-readable rationale
}
```

- [ ] **Step 1.4: Add `Item.Provenance` and `Subphase.DriftState`**

In `internal/progress/progress.go`:

```go
type Item struct {
	// ... existing fields ...
	Health         *RowHealth      `json:"health,omitempty"`
	PlannerVerdict *PlannerVerdict `json:"planner_verdict,omitempty"`
	Provenance     *Provenance     `json:"provenance,omitempty"`
}

type Subphase struct {
	// ... existing fields ...
	DriftState *DriftState `json:"drift_state,omitempty"`
}
```

- [ ] **Step 1.5: Re-run tests**

```bash
go test ./internal/progress/ -run 'TestProvenance|TestDriftState|TestItem_ProvenanceOmitemptyByDefault|TestSubphase_DriftStateOmitemptyByDefault' -v
```
Expected: PASS for all 5.

- [ ] **Step 1.6: Extend symmetric preservation tests**

Append to `internal/progress/preservation_test.go`:

```go
func TestSymmetricPreservation_FourBlocksRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	body := `{
  "version": "1",
  "phases": {
    "1": {
      "name": "P",
      "subphases": {
        "1.A": {
          "name": "S",
          "drift_state": {"status": "owned", "origin_decision": "test"},
          "items": [
            {"name": "row-1", "status": "planned", "contract": "do x",
             "health": {"attempt_count": 1},
             "planner_verdict": {"reshape_count": 2, "last_outcome": "still_failing"},
             "provenance": {"origin_type": "gormes", "owned_since": "2026-04-25T00:00:00Z"}}
          ]
        }
      }
    }
  }
}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Round-trip via SaveProgress (planner side) — all 4 blocks survive.
	prog, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := SaveProgress(path, prog); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	prog2, _ := Load(path)
	row := &prog2.Phases["1"].Subphases["1.A"].Items[0]
	if row.Health == nil || row.PlannerVerdict == nil || row.Provenance == nil {
		t.Fatal("one of the row-level typed blocks went missing")
	}
	if prog2.Phases["1"].Subphases["1.A"].DriftState == nil {
		t.Fatal("DriftState went missing on round-trip")
	}

	// Round-trip via ApplyHealthUpdates (autoloop side) — all 4 blocks survive.
	if err := ApplyHealthUpdates(path, []HealthUpdate{{
		PhaseID: "1", SubphaseID: "1.A", ItemName: "row-1",
		Mutate: func(h *RowHealth) {
			h.AttemptCount = 5
		},
	}}); err != nil {
		t.Fatalf("ApplyHealthUpdates: %v", err)
	}
	prog3, _ := Load(path)
	row3 := &prog3.Phases["1"].Subphases["1.A"].Items[0]
	if row3.PlannerVerdict == nil || row3.PlannerVerdict.ReshapeCount != 2 {
		t.Fatal("PlannerVerdict erased by autoloop write")
	}
	if row3.Provenance == nil || row3.Provenance.OriginType != "gormes" {
		t.Fatal("Provenance erased by autoloop write")
	}
	if prog3.Phases["1"].Subphases["1.A"].DriftState == nil ||
		prog3.Phases["1"].Subphases["1.A"].DriftState.Status != "owned" {
		t.Fatal("DriftState erased by autoloop write")
	}
}
```

- [ ] **Step 1.7: Run, vet, gofmt, commit**

```bash
go test ./internal/progress/...
go vet ./internal/progress/...
gofmt -l internal/progress/
git add internal/progress/health.go internal/progress/progress.go internal/progress/health_test.go internal/progress/preservation_test.go
git commit -m "feat(progress): add Provenance + DriftState schema for divergence awareness"
```

---

## Task 2: ScanImplementation + ContextBundle.ImplInventory

**Files:**
- Create: `internal/architectureplanner/implscan.go`
- Create: `internal/architectureplanner/implscan_test.go`
- Modify: `internal/architectureplanner/context.go`
- Modify: `internal/architectureplanner/config.go`
- Modify: `internal/architectureplanner/config_test.go`
- Modify: `internal/architectureplanner/run.go`

- [ ] **Step 2.1: Write failing tests**

Create `internal/architectureplanner/implscan_test.go`:

```go
package architectureplanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanImplementation_DenyListPathsAreOriginal(t *testing.T) {
	dir := t.TempDir()
	// Synth an impl tree.
	for _, p := range []string{
		"cmd/autoloop/main.go",
		"internal/architectureplanner/run.go",
		"internal/plannertriggers/triggers.go",
		"cmd/gormes/main.go",          // NOT original (not in deny list)
		"internal/gateway/server.go",  // NOT original
	} {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	denyList := []string{
		"cmd/autoloop/",
		"internal/architectureplanner/",
		"internal/plannertriggers/",
	}
	inv, err := ScanImplementation(dir, denyList, 24*time.Hour, time.Now())
	if err != nil {
		t.Fatalf("ScanImplementation: %v", err)
	}

	// All deny-listed prefixes should appear in GormesOriginalPaths inventory.
	wantGormes := map[string]bool{
		"cmd/autoloop/main.go":                       true,
		"internal/architectureplanner/run.go":        true,
		"internal/plannertriggers/triggers.go":       true,
	}
	gotGormes := map[string]bool{}
	for _, p := range inv.GormesOriginalPaths {
		gotGormes[p] = true
	}
	for w := range wantGormes {
		if !gotGormes[w] {
			t.Errorf("expected %q in GormesOriginalPaths, got %v", w, inv.GormesOriginalPaths)
		}
	}

	// Non-deny paths must NOT appear in GormesOriginalPaths.
	for _, mustNot := range []string{"cmd/gormes/main.go", "internal/gateway/server.go"} {
		if gotGormes[mustNot] {
			t.Errorf("path %q should NOT be in GormesOriginalPaths", mustNot)
		}
	}
}

func TestScanImplementation_RecentlyChangedHonorsLookback(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)

	recent := filepath.Join(dir, "cmd/autoloop/recent.go")
	old := filepath.Join(dir, "cmd/autoloop/old.go")
	for _, p := range []string{recent, old} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(recent, now.Add(-1*time.Hour), now.Add(-1*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(old, now.Add(-48*time.Hour), now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}

	inv, err := ScanImplementation(dir, []string{"cmd/autoloop/"}, 24*time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"cmd/autoloop/recent.go": true}
	for _, p := range inv.RecentlyChanged {
		if !want[p] {
			t.Errorf("RecentlyChanged includes %q (older than lookback)", p)
		}
	}
	if len(inv.RecentlyChanged) != 1 {
		t.Errorf("expected 1 recently changed, got %d: %v", len(inv.RecentlyChanged), inv.RecentlyChanged)
	}
}

func TestScanImplementation_MissingDirReturnsEmpty(t *testing.T) {
	inv, err := ScanImplementation("/nonexistent-dir-12345", nil, time.Hour, time.Now())
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if len(inv.GormesOriginalPaths) != 0 || len(inv.RecentlyChanged) != 0 {
		t.Errorf("expected empty inventory, got %+v", inv)
	}
}
```

- [ ] **Step 2.2: Run failing tests**

```bash
go test ./internal/architectureplanner/ -run TestScanImplementation -v
```
Expected: FAIL — `ScanImplementation` and `ImplInventory` don't exist.

- [ ] **Step 2.3: Implement `internal/architectureplanner/implscan.go`**

```go
package architectureplanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ImplInventory is what ScanImplementation returns to the planner prompt:
// the set of impl-tree paths that are Gormes-original (no upstream analog),
// the subset modified within the lookback window, and the candidate list of
// subphases that are entirely Gormes-original (suggested for promotion to
// DriftState.Status="owned").
type ImplInventory struct {
	GormesOriginalPaths []string `json:"gormes_original_paths,omitempty"`
	RecentlyChanged     []string `json:"recently_changed,omitempty"`
	OwnedSubphases      []string `json:"owned_subphases,omitempty"`
}

// DefaultGormesOriginalPaths is the seed deny-list of paths considered
// Gormes-original (no upstream Hermes/GBrain/Honcho analog). Tunable via
// Config.GormesOriginalPaths and the PLANNER_GORMES_ORIGINAL_PATHS env var.
var DefaultGormesOriginalPaths = []string{
	"cmd/autoloop/",
	"cmd/architecture-planner-loop/",
	"internal/autoloop/",
	"internal/architectureplanner/",
	"internal/plannertriggers/",
	"internal/progress/health.go",
	"www.gormes.ai/internal/site/installers/",
}

// ScanImplementation walks repoRoot/cmd and repoRoot/internal, identifies
// paths matching any of gormesOriginalPaths (prefix match), and reports the
// subset modified within [now-lookback, now]. Used by L3 prompt to give the
// LLM a concrete "what's here that you don't need to research upstream for"
// inventory.
//
// Missing repoRoot returns empty inventory, no error (fresh checkout case).
func ScanImplementation(repoRoot string, gormesOriginalPaths []string, lookback time.Duration, now time.Time) (ImplInventory, error) {
	if len(gormesOriginalPaths) == 0 {
		gormesOriginalPaths = DefaultGormesOriginalPaths
	}

	var inv ImplInventory
	cutoff := now.Add(-lookback)

	for _, root := range []string{"cmd", "internal"} {
		base := filepath.Join(repoRoot, root)
		if _, err := os.Stat(base); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return inv, err
		}
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // skip unreadable entries
			}
			if d.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if !matchesAnyPrefix(rel, gormesOriginalPaths) {
				return nil
			}
			inv.GormesOriginalPaths = append(inv.GormesOriginalPaths, rel)

			info, infoErr := d.Info()
			if infoErr != nil {
				return nil
			}
			mt := info.ModTime()
			if !mt.Before(cutoff) && !mt.After(now) {
				inv.RecentlyChanged = append(inv.RecentlyChanged, rel)
			}
			return nil
		})
		if err != nil {
			return inv, err
		}
	}

	sort.Strings(inv.GormesOriginalPaths)
	sort.Strings(inv.RecentlyChanged)
	return inv, nil
}

func matchesAnyPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2.4: Add ContextBundle.ImplInventory field**

In `internal/architectureplanner/context.go`:

```go
type ContextBundle struct {
	// ... existing fields ...
	ImplInventory ImplInventory `json:"impl_inventory,omitempty"`
}
```

- [ ] **Step 2.5: Add Config fields**

In `internal/architectureplanner/config.go`:

```go
type Config struct {
	// ... existing ...
	GormesOriginalPaths []string      // PLANNER_GORMES_ORIGINAL_PATHS (CSV)
	ImplLookback        time.Duration // PLANNER_IMPL_LOOKBACK; default 24h
	TriggerReason       string        // PLANNER_TRIGGER_REASON; for L4 path-unit handoff
}
```

In `ConfigFromEnv`, defaults: `GormesOriginalPaths = nil` (ScanImplementation falls back to `DefaultGormesOriginalPaths`); `ImplLookback = 24*time.Hour`; `TriggerReason = ""`. Env overrides:
- `PLANNER_GORMES_ORIGINAL_PATHS` → `splitCSV` (existing helper)
- `PLANNER_IMPL_LOOKBACK` → `time.ParseDuration`
- `PLANNER_TRIGGER_REASON` → string

- [ ] **Step 2.6: Wire ScanImplementation into RunOnce**

In `RunOnce` (after CollectContext, before BuildPrompt):

```go
inv, _ := ScanImplementation(cfg.RepoRoot, cfg.GormesOriginalPaths, cfg.ImplLookback, now)
inv.OwnedSubphases = computeOwnedSubphases(prog, cfg.GormesOriginalPaths) // see below
bundle.ImplInventory = inv
```

Add a small helper `computeOwnedSubphases(prog, paths)` that walks subphases and returns IDs whose every item's `WriteScope` is entirely under one of the deny-list prefixes. Returns `[]string` like `["5.O", "5.P"]`. If a subphase has zero items with WriteScope, treat as not-owned (can't decide).

- [ ] **Step 2.7: Run, vet, gofmt, commit**

```bash
go test ./internal/architectureplanner/...
go vet ./internal/architectureplanner/...
gofmt -l internal/architectureplanner/
git add internal/architectureplanner/implscan.go internal/architectureplanner/implscan_test.go internal/architectureplanner/context.go internal/architectureplanner/config.go internal/architectureplanner/config_test.go internal/architectureplanner/run.go
git commit -m "feat(planner): scan impl tree to surface Gormes-original inventory"
```

---

## Task 3: Drift-Aware Prompt Clauses

**Files:**
- Modify: `internal/architectureplanner/prompt.go`
- Modify: `internal/architectureplanner/prompt_test.go`

- [ ] **Step 3.1: Write failing tests**

Append to `internal/architectureplanner/prompt_test.go`:

```go
func TestBuildPrompt_ProvenanceClauseAlwaysPresent(t *testing.T) {
	prompt := BuildPrompt(ContextBundle{}, nil)
	for _, want := range []string{"PROVENANCE AWARENESS", "DRIFT STATE"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q clause:\n%s", want, prompt)
		}
	}
}

func TestBuildPrompt_ImplInventorySectionRendersWhenPresent(t *testing.T) {
	bundle := ContextBundle{
		ImplInventory: ImplInventory{
			GormesOriginalPaths: []string{"cmd/autoloop/main.go", "internal/autoloop/run.go"},
			RecentlyChanged:     []string{"cmd/autoloop/main.go"},
			OwnedSubphases:      []string{"5.O", "5.P"},
		},
	}
	prompt := BuildPrompt(bundle, nil)
	for _, want := range []string{
		"## Implementation Inventory",
		"cmd/autoloop/main.go",
		"5.O",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildPrompt_OmitsImplInventorySectionWhenEmpty(t *testing.T) {
	prompt := BuildPrompt(ContextBundle{}, nil)
	if strings.Contains(prompt, "## Implementation Inventory") {
		t.Fatal("Implementation Inventory section should be omitted when bundle has no inventory")
	}
}
```

- [ ] **Step 3.2: Run failing tests**

```bash
go test ./internal/architectureplanner/ -run 'TestBuildPrompt_(ProvenanceClauseAlwaysPresent|ImplInventorySectionRendersWhenPresent|OmitsImplInventorySectionWhenEmpty)' -v
```
Expected: FAIL.

- [ ] **Step 3.3: Add clauses + section in `prompt.go`**

```go
const provenanceAwarenessClause = `
PROVENANCE AWARENESS (SOFT RULE)

Every progress.json row SHOULD carry a ` + "`provenance`" + ` block declaring its
origin_type ("upstream", "gormes", or "hybrid"). When you create or refine
a row in a Gormes-owned area (see Implementation Inventory section below),
set provenance.origin_type="gormes" and origin_decision describing why.
Do NOT fabricate upstream_ref pointers when none exist — leave the field
empty and rely on the origin_decision to explain.
`

const driftStateClause = `
DRIFT STATE (SOFT RULE)

Subphases progress through three drift states (Subphase.drift_state.status):
  - "porting"   — upstream leads; refine contracts against upstream code
  - "converged" — Gormes matches upstream; planner only checks for upstream
                  changes that warrant new rows
  - "owned"     — Gormes leads; ignore upstream for this surface; refine
                  contracts against the Gormes implementation only

Promote subphases to "converged" when all their rows are shipped and
upstream hasn't changed materially. Promote to "owned" when Gormes has
shipped functionality with no upstream analog (e.g. autoloop, plannertriggers).
The Implementation Inventory below lists OwnedSubphases the impl scan
identified as candidates for owned promotion. This is a one-way ratchet —
do not demote from owned back to converged or porting.
`

func formatImplInventory(inv ImplInventory) string {
	if len(inv.GormesOriginalPaths) == 0 && len(inv.RecentlyChanged) == 0 && len(inv.OwnedSubphases) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n## Implementation Inventory\n\n")
	if len(inv.GormesOriginalPaths) > 0 {
		b.WriteString("Gormes-original surfaces (no upstream research needed):\n")
		for _, p := range inv.GormesOriginalPaths {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		b.WriteString("\n")
	}
	if len(inv.RecentlyChanged) > 0 {
		b.WriteString("Recently changed (last lookback window):\n")
		for _, p := range inv.RecentlyChanged {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		b.WriteString("\n")
	}
	if len(inv.OwnedSubphases) > 0 {
		b.WriteString("Subphases that ARE entirely Gormes-original (candidates for \"owned\"):\n")
		for _, s := range inv.OwnedSubphases {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}
	return b.String()
}
```

In `BuildPrompt`, append `provenanceAwarenessClause`, `driftStateClause`, and `formatImplInventory(bundle.ImplInventory)` to the existing template.

- [ ] **Step 3.4: Run, vet, gofmt, commit**

```bash
go test ./internal/architectureplanner/...
go vet ./internal/architectureplanner/...
gofmt -l internal/architectureplanner/
git add internal/architectureplanner/prompt.go internal/architectureplanner/prompt_test.go
git commit -m "feat(planner): add provenance + drift state prompt clauses"
```

---

## Task 4: Reactive Impl Trigger (systemd Path Unit)

**Files:**
- Modify: `internal/architectureplanner/service.go`
- Modify: `internal/architectureplanner/service_test.go`
- Modify: `internal/architectureplanner/run.go` (honor TriggerReason)
- Modify: `internal/architectureplanner/ledger.go` (no-op; just confirm "impl_change" is a valid Trigger value)

- [ ] **Step 4.1: Write failing tests**

Append to `internal/architectureplanner/service_test.go`:

```go
func TestRenderPlannerImplPathUnit_HasLongerRateLimit(t *testing.T) {
	rendered := RenderPlannerImplPathUnit(PlannerImplPathUnitOptions{
		Description:    "Trigger Gormes architecture planner on impl tree change",
		PathsToWatch:   []string{"/repo/cmd", "/repo/internal"},
		ServiceUnit:    "gormes-architecture-planner.service",
		TriggerReason:  "impl_change",
	})
	for _, w := range []string{
		"PathChanged=/repo/cmd",
		"PathChanged=/repo/internal",
		"TriggerLimitIntervalSec=1800",
		"TriggerLimitBurst=1",
		"Unit=gormes-architecture-planner.service",
	} {
		if !strings.Contains(rendered, w) {
			t.Errorf("rendered impl-path unit missing %q\n%s", w, rendered)
		}
	}
}

func TestInstallPlannerService_WritesAllFourUnits(t *testing.T) {
	dir := t.TempDir()
	opts := PlannerServiceInstallOptions{
		Runner:       &autoloop.FakeRunner{Results: []autoloop.Result{{}, {}, {}, {}}},
		UnitDir:      dir,
		UnitName:     "gormes-architecture-planner.service",
		TimerName:    "gormes-architecture-planner.timer",
		PathName:     "gormes-architecture-planner.path",
		ImplPathName: "gormes-architecture-planner-impl.path",
		PlannerPath:  "/usr/local/bin/planner.sh",
		WorkDir:      "/repo",
		PathToWatch:  "/repo/.codex/architecture-planner/triggers.jsonl",
		ImplPathsToWatch: []string{"/repo/cmd", "/repo/internal"},
	}
	if err := InstallPlannerService(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"gormes-architecture-planner.service",
		"gormes-architecture-planner.timer",
		"gormes-architecture-planner.path",
		"gormes-architecture-planner-impl.path",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("unit %s not written: %v", name, err)
		}
	}
}

func TestRunOnce_TriggerImplChangeFromEnv(t *testing.T) {
	t.Skip("FILL IN: set cfg.TriggerReason='impl_change' (no plannertriggers events queued); run via existing fixture; assert ledger entry has Trigger='impl_change'")
}
```

- [ ] **Step 4.2: Run failing tests, then implement**

Add `RenderPlannerImplPathUnit` + `PlannerImplPathUnitOptions` in `service.go`. Extend `PlannerServiceInstallOptions` with `ImplPathName string` and `ImplPathsToWatch []string`. Extend `InstallPlannerService` to write the impl path unit when `ImplPathName != "" && len(ImplPathsToWatch) > 0` (idempotent — same `writePlannerUnit` discipline).

Render template:

```go
const plannerImplPathUnitTemplate = `[Unit]
Description=%s

[Path]
%s
TriggerLimitIntervalSec=1800
TriggerLimitBurst=1
Unit=%s

[Install]
WantedBy=default.target
`

func RenderPlannerImplPathUnit(opts PlannerImplPathUnitOptions) string {
	var lines []string
	for _, p := range opts.PathsToWatch {
		lines = append(lines, "PathChanged="+p)
	}
	return fmt.Sprintf(plannerImplPathUnitTemplate, opts.Description, strings.Join(lines, "\n"), opts.ServiceUnit)
}
```

Note: the .path unit doesn't expose `TriggerReason` directly; that's threaded via env on the service's ExecStart. The wrapper script `scripts/architecture-planner-loop.sh` should read `$PLANNER_TRIGGER_REASON` (set by the Service's Environment= directive in a future drop-in OR by the wrapper itself based on which path unit fired). For Phase D, the simplest approach: when the impl-path unit installs, it ALSO writes a small drop-in `.service.d/impl-trigger.conf` with `Environment="PLANNER_TRIGGER_REASON=impl_change"`. That's optional; the basic ledger field already accepts the value if present in env.

- [ ] **Step 4.3: Wire TriggerReason in RunOnce**

In `RunOnce`, when computing the `trigger` value (currently just "scheduled" or "event"):

```go
trigger := "scheduled"
if len(triggerEvents) > 0 {
    trigger = "event"
} else if cfg.TriggerReason == "impl_change" {
    trigger = "impl_change"
}
```

Priority order: `event` (quarantine triggers from autoloop) > `impl_change` (impl-tree path unit) > `scheduled` (timer).

- [ ] **Step 4.4: Run, vet, gofmt, commit**

```bash
go test ./internal/architectureplanner/...
go vet ./internal/architectureplanner/...
gofmt -l internal/architectureplanner/
git add internal/architectureplanner/service.go internal/architectureplanner/service_test.go internal/architectureplanner/run.go cmd/architecture-planner-loop/main.go
git commit -m "feat(planner): install impl-tree path unit for divergence-aware trigger"
```

---

## Task 5: Drift Status + Ledger Forensics + Lifecycle Test

**Files:**
- Modify: `internal/architectureplanner/ledger.go`
- Modify: `internal/architectureplanner/status.go`
- Modify: `internal/architectureplanner/status_test.go`
- Modify: `internal/architectureplanner/run.go` (compute DriftPromotions, emit in ledger)
- Create: `internal/architectureplanner/divergence_lifecycle_test.go`

- [ ] **Step 5.1: Add DriftPromotion type + LedgerEvent field**

In `internal/architectureplanner/ledger.go`:

```go
type DriftPromotion struct {
	SubphaseID string `json:"subphase_id"`
	From       string `json:"from"`
	To         string `json:"to"`
	Reason     string `json:"reason,omitempty"`
}

type LedgerEvent struct {
	// ... existing fields ...
	DriftPromotions []DriftPromotion `json:"drift_promotions,omitempty"`
}
```

- [ ] **Step 5.2: Compute DriftPromotions in RunOnce**

Helper `diffSubphaseStates(before, after *progress.Progress) []DriftPromotion` walks both docs by subphase ID and records cases where `before.DriftState.Status != after.DriftState.Status` (only forward transitions: porting→converged, porting→owned, converged→owned). Backward transitions are logged but not emitted as promotions (humans demote).

Wire into the RunOnce ledger emit (alongside `RowsChanged`):

```go
event.DriftPromotions = diffSubphaseStates(beforeDoc, afterDoc)
```

- [ ] **Step 5.3: Status surface drift bucketing**

Extend `RenderStatus` in `status.go`. After the existing "Rows needing human attention" section, append:

```go
func renderDriftStateBuckets(prog *progress.Progress) string {
	if prog == nil {
		return ""
	}
	var porting, converged, owned []string
	for phaseID, phase := range prog.Phases {
		for subID, sub := range phase.Subphases {
			id := phaseID + "." + subID // adjust to match actual subphase ID format
			_ = id
			if sub.DriftState == nil {
				porting = append(porting, subID)
				continue
			}
			switch sub.DriftState.Status {
			case "converged":
				converged = append(converged, subID)
			case "owned":
				owned = append(owned, subID)
			default:
				porting = append(porting, subID)
			}
		}
	}
	sort.Strings(porting); sort.Strings(converged); sort.Strings(owned)

	var b strings.Builder
	b.WriteString("\nDrift state by subphase:\n")
	fmt.Fprintf(&b, "  PORTING (%d): %s\n", len(porting), strings.Join(porting, ", "))
	fmt.Fprintf(&b, "  CONVERGED (%d): %s\n", len(converged), strings.Join(converged, ", "))
	fmt.Fprintf(&b, "  OWNED (%d): %s\n", len(owned), strings.Join(owned, ", "))
	return b.String()
}

func renderRecentDriftPromotions(events []LedgerEvent) string {
	var lines []string
	for _, ev := range events {
		for _, p := range ev.DriftPromotions {
			lines = append(lines, fmt.Sprintf("  - %s: %s → %s (%s, run %s)",
				p.SubphaseID, p.From, p.To, ev.TS, ev.RunID))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "\nRecent drift promotions (last 7d):\n" + strings.Join(lines, "\n") + "\n"
}
```

Call both from `RenderStatus` (events come from a 7-day window via `LoadLedgerWindow`).

- [ ] **Step 5.4: Status tests**

In `status_test.go`:

```go
func TestRenderDriftStateBuckets_BucketsAllThreeStates(t *testing.T) { /* synthesize 3 subphases, one of each */ }

func TestRenderRecentDriftPromotions_EmptyOmitsHeader(t *testing.T) { /* no promotions → empty string */ }

func TestRenderRecentDriftPromotions_ListsRecentEvents(t *testing.T) { /* synthesize ledger with promotions, assert format */ }
```

- [ ] **Step 5.5: Lifecycle test**

Create `internal/architectureplanner/divergence_lifecycle_test.go`:

```go
package architectureplanner

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

// TestLifecycle_DivergenceFullCycle walks one subphase through:
//   Run 1: subphase has no drift_state; planner stamps status="porting"
//   Run 2: planner adds 3 rows with provenance.origin_type="upstream"
//   Run 3: rows ship; planner promotes status to "converged"
//   Run 4: Gormes-original extension lands; planner adds row with
//          provenance.origin_type="gormes" and promotes to "owned"
//   Run 5: ImplInventory.OwnedSubphases includes the subphase;
//          planner stops reading upstream for it
func TestLifecycle_DivergenceFullCycle(t *testing.T) {
	t.Skip("FILL IN: scaffold using direct progress.SaveProgress + fake ScanImplementation results; mock the LLM by directly editing rows + drift_state")
}
```

- [ ] **Step 5.6: Run, vet, gofmt, commit**

```bash
go test ./internal/architectureplanner/...
go vet ./internal/architectureplanner/...
gofmt -l internal/architectureplanner/
git add internal/architectureplanner/ledger.go internal/architectureplanner/status.go internal/architectureplanner/status_test.go internal/architectureplanner/run.go internal/architectureplanner/divergence_lifecycle_test.go
git commit -m "feat(planner): drift status surface + ledger forensics for divergence"
```

---

## Self-Review Checklist

### Spec coverage

| Spec section | Implementing task |
|---|---|
| L1 Provenance + DriftState schema | Task 1 |
| Symmetric preservation extended (4 blocks) | Task 1 (preservation_test.go) |
| L2 ScanImplementation + ImplInventory | Task 2 |
| L2 Config.GormesOriginalPaths + Config.ImplLookback | Task 2 |
| L3 PROVENANCE + DRIFT STATE clauses | Task 3 |
| L3 Implementation Inventory section | Task 3 |
| L4 systemd impl-path unit | Task 4 |
| L4 Trigger="impl_change" via env | Task 4 |
| L5 RenderStatus drift buckets + recent promotions | Task 5 |
| L5 LedgerEvent.DriftPromotions | Task 5 |
| L5 Divergence lifecycle test | Task 5 |

### Placeholder scan

- Tasks 4 and 5 each have ONE deliberate `t.Skip("FILL IN: ...")` stub for tests that need fixture-style scaffolding the implementer must follow. Required test names and scenarios are pinned.
- All other steps include exact code or exact commands.

### Type / API consistency

Names cross-referenced:
- `progress.Provenance` (Task 1) used in Tasks 5
- `progress.DriftState` (Task 1) used in Tasks 2, 3, 5
- `progress.Item.Provenance`, `progress.Subphase.DriftState` (Task 1) used everywhere
- `architectureplanner.ImplInventory` (Task 2) used in Tasks 3, 5
- `architectureplanner.ScanImplementation` (Task 2) used in Task 4 (impl_change trigger consumer)
- `architectureplanner.DefaultGormesOriginalPaths` (Task 2)
- `architectureplanner.Config.GormesOriginalPaths`, `.ImplLookback`, `.TriggerReason` (Task 2)
- `architectureplanner.RenderPlannerImplPathUnit`, `PlannerImplPathUnitOptions` (Task 4)
- `architectureplanner.PlannerServiceInstallOptions.ImplPathName`, `.ImplPathsToWatch` (Task 4)
- `architectureplanner.LedgerEvent.DriftPromotions`, `architectureplanner.DriftPromotion` (Task 5)
- `architectureplanner.diffSubphaseStates` (Task 5; private helper)
- `architectureplanner.renderDriftStateBuckets`, `renderRecentDriftPromotions` (Task 5)

All names cross-reference correctly between tasks.

### Cross-cutting concerns

- **Backward compat:** Rows without Provenance default to "upstream" semantics in L3 prompt. Subphases without DriftState default to "porting" in L5 status bucketing.
- **One-way ratchet:** Status promotion is monotonic; demotion only via direct human edit to progress.json.
- **Symmetric preservation:** All four typed blocks (Health, PlannerVerdict, Provenance, DriftState) survive both writers via Phase B's typed round-trip. No new MarshalJSON code.
