# Reactive Autoloop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make autoloop self-limiting on broken rows and self-healing on noisy worker output by introducing a typed `Health` block on each `progress.json` row that autoloop owns and the planner reads.

**Architecture:** Five-commit layered design. L1 extends `internal/progress` with the `RowHealth` schema and atomic merge IO. L2/L4 add a per-run accumulator that writes health at run-end, soft-skips per-row preflight failures, and degrades the backend after K consecutive backend errors. L3 makes selection skip quarantined rows and penalize ranking by failure history. L5 adds a report-repair pass that salvages worker output that produced a real commit + passing tests but failed strict `ParseFinalReport`. L6 surfaces quarantined rows in the planner context and adds a post-regen validator that rejects health drops.

**Tech Stack:** Go 1.25+, `encoding/json` (typed structs with `omitempty`), `crypto/sha256` (spec hashing), `os.Rename` (atomic write), existing `internal/progress` and `internal/autoloop` packages.

**Reference spec:** `docs/superpowers/specs/2026-04-24-reactive-autoloop-design.md`

**Baseline commit (spec):** `b99c27ab`

---

## File Structure

**New files:**

```text
internal/progress/health.go
internal/progress/health_test.go
internal/progress/health_compat_test.go
internal/progress/health_concurrent_test.go
internal/autoloop/health_writer.go
internal/autoloop/health_writer_test.go
internal/autoloop/lifecycle_test.go
internal/autoloop/candidates_health_test.go
internal/architectureplanner/health_preservation_test.go
```

**Modified files:**

```text
internal/progress/progress.go
internal/autoloop/run.go
internal/autoloop/candidates.go
internal/autoloop/backend.go
internal/autoloop/config.go
internal/autoloop/report.go
internal/autoloop/promote.go
internal/architectureplanner/context.go
internal/architectureplanner/prompt.go
internal/architectureplanner/run.go
```

**Responsibility map:**

- `internal/progress/health.go`: `RowHealth`, `FailureSummary`, `Quarantine`, `FailureCategory` types; `ItemSpecHash`; `HealthUpdate`; `ApplyHealthUpdates`; `SaveProgress` (atomic write helper).
- `internal/progress/progress.go`: add `Health *RowHealth` to `Item`; preserve all existing fields and behavior.
- `internal/autoloop/health_writer.go`: `healthAccumulator`, `pendingHealth`, `rowKey`; per-worker outcome recorders; `Flush` builds `[]HealthUpdate` and calls `ApplyHealthUpdates`.
- `internal/autoloop/run.go`: instantiate accumulator + `backendDegrader` per run; emit new ledger event types; soft-skip preflight failures.
- `internal/autoloop/candidates.go`: `failurePenalty(n)`, quarantine filter, stale-quarantine detection, selection-reason additions.
- `internal/autoloop/backend.go`: `backendDegrader` type; `IsBackendError(workerOutcome)` helper.
- `internal/autoloop/config.go`: register `QUARANTINE_THRESHOLD`, `BACKEND_DEGRADE_THRESHOLD`, `BACKEND_FALLBACK`, `GORMES_INCLUDE_QUARANTINED`, `GORMES_REPORT_REPAIR`, `GORMES_PLANNER_QUARANTINE_LIMIT`.
- `internal/autoloop/report.go`: keep strict `ParseFinalReport` untouched; add `TryRepairReport(ctx RepairContext)` and `RepairNote`.
- `internal/autoloop/promote.go`: extend promotion path to accept repaired reports and write `repairs/<runID>-<workerID>.json`.
- `internal/architectureplanner/context.go`: `QuarantinedRowContext`; `collectQuarantinedRows`; thread into `CollectContext`.
- `internal/architectureplanner/prompt.go`: append two new clauses (HARD: preservation; SOFT: priority).
- `internal/architectureplanner/run.go`: `validateHealthPreservation(before, after)`; reject regenerations that drop or modify health blocks.

---

## Conventions Used In Every Task

- Every task is one TDD cycle ending in one commit. Steps inside a task are 2-5 minutes each.
- Always run the failing test first; never write implementation before a red test.
- Run `go vet ./...` and `gofmt -l .` before every commit; both must be clean.
- Run the full focused test suite (`go test ./internal/progress/... ./internal/autoloop/... ./internal/architectureplanner/...`) before every commit.
- Commit message format:
  - `feat(progress): ...`
  - `feat(autoloop): ...`
  - `feat(planner): ...`
  - `test(autoloop): ...`
- Never modify `internal/progress/Item` field ordering (the existing field order is reflected in checked-in `progress.json`; reordering would create a noisy diff).
- All file IO that writes `progress.json` goes through `internal/progress.SaveProgress` (added in Task 2). No direct `os.WriteFile` of progress JSON anywhere in the codebase after Task 2.
- New env vars all default to current behavior (back-compat). Tests verify defaults.

---

## Task 1: Add `RowHealth` Schema To `internal/progress`

**Files:**
- Create: `internal/progress/health.go`
- Create: `internal/progress/health_test.go`
- Modify: `internal/progress/progress.go` (add `Health *RowHealth` field to `Item`)

- [ ] **Step 1.1: Write the failing test for the schema and round-trip**

Create `internal/progress/health_test.go`:

```go
package progress

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRowHealth_RoundTrip(t *testing.T) {
	row := &RowHealth{
		AttemptCount:        5,
		ConsecutiveFailures: 3,
		LastAttempt:         "2026-04-24T12:00:00Z",
		LastSuccess:         "2026-04-23T08:00:00Z",
		LastFailure: &FailureSummary{
			RunID:      "20260424T120000Z-1234-001",
			Category:   FailureReportValidation,
			Backend:    "codexu",
			StderrTail: "tests failed",
		},
		BackendsTried: []string{"codexu", "claudeu"},
		Quarantine: &Quarantine{
			Reason:       "auto: 3 consecutive failures",
			Since:        "2026-04-24T12:05:00Z",
			AfterRunID:   "20260424T120500Z-1234-001",
			Threshold:    3,
			SpecHash:     "abc123",
			LastCategory: FailureReportValidation,
		},
	}

	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got RowHealth
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.AttemptCount != 5 {
		t.Fatalf("AttemptCount = %d, want 5", got.AttemptCount)
	}
	if got.Quarantine == nil || got.Quarantine.SpecHash != "abc123" {
		t.Fatalf("Quarantine.SpecHash mismatch: %+v", got.Quarantine)
	}
	if got.LastFailure == nil || got.LastFailure.Category != FailureReportValidation {
		t.Fatalf("LastFailure.Category mismatch: %+v", got.LastFailure)
	}
}

func TestRowHealth_OmitemptyKeepsZeroFieldsOut(t *testing.T) {
	row := &RowHealth{}
	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "{}" {
		t.Fatalf("zero-value RowHealth should marshal to {}, got %s", data)
	}
}

func TestItem_HealthOmitemptyByDefault(t *testing.T) {
	item := &Item{Name: "x", Status: StatusPlanned}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "health") {
		t.Fatalf("Item with no health should not emit health key, got %s", data)
	}
}
```

- [ ] **Step 1.2: Run the failing tests**

Run:

```bash
go test ./internal/progress/ -run 'TestRowHealth_RoundTrip|TestRowHealth_OmitemptyKeepsZeroFieldsOut|TestItem_HealthOmitemptyByDefault' -v
```

Expected:

- `FAIL` because `RowHealth`, `FailureSummary`, `Quarantine`, `FailureCategory` constants do not exist yet, and `Item.Health` does not exist.

- [ ] **Step 1.3: Create `internal/progress/health.go`**

```go
package progress

// RowHealth is execution-history metadata about one progress.json item.
// Owned by autoloop. The planner READS it to prioritize repairs and MUST
// preserve any unknown fields verbatim across regenerations.
type RowHealth struct {
	AttemptCount        int             `json:"attempt_count,omitempty"`
	ConsecutiveFailures int             `json:"consecutive_failures,omitempty"`
	LastAttempt         string          `json:"last_attempt,omitempty"`
	LastSuccess         string          `json:"last_success,omitempty"`
	LastFailure         *FailureSummary `json:"last_failure,omitempty"`
	BackendsTried       []string        `json:"backends_tried,omitempty"`
	Quarantine          *Quarantine     `json:"quarantine,omitempty"`
}

// FailureSummary is autoloop's classification of a worker outcome.
type FailureSummary struct {
	RunID      string          `json:"run_id"`
	Category   FailureCategory `json:"category"`
	Backend    string          `json:"backend,omitempty"`
	StderrTail string          `json:"stderr_tail,omitempty"`
}

// FailureCategory is the closed set of failure classifications autoloop emits.
type FailureCategory string

const (
	FailureWorkerError      FailureCategory = "worker_error"
	FailureReportValidation FailureCategory = "report_validation_failed"
	FailureProgressSummary  FailureCategory = "progress_summary_failed"
	FailureTimeout          FailureCategory = "timeout"
	FailureBackendDegraded  FailureCategory = "backend_degraded"
)

// Quarantine is set when ConsecutiveFailures crosses QUARANTINE_THRESHOLD.
// Cleared when (a) a future run succeeds on the row, (b) the row's spec hash
// changes (planner reshape detected), or (c) a human deletes the block.
type Quarantine struct {
	Reason       string          `json:"reason"`
	Since        string          `json:"since"`
	AfterRunID   string          `json:"after_run_id"`
	Threshold    int             `json:"threshold"`
	SpecHash     string          `json:"spec_hash"`
	LastCategory FailureCategory `json:"last_category"`
}
```

Modify `internal/progress/progress.go` `Item` struct: add the new field as the LAST field of the struct (so existing field ordering is preserved):

```go
type Item struct {
	Name     string `json:"name"`
	Status   Status `json:"status"`
	// ... all existing fields preserved in order ...
	PR    string `json:"pr,omitempty"`
	Owner string `json:"owner,omitempty"`
	ETA   string `json:"eta,omitempty"`
	Note  string `json:"note,omitempty"`

	// Health is execution-history metadata owned by autoloop. The planner
	// must preserve this block verbatim across regenerations (see
	// docs/superpowers/specs/2026-04-24-reactive-autoloop-design.md).
	Health *RowHealth `json:"health,omitempty"`
}
```

- [ ] **Step 1.4: Re-run the failing tests**

Run:

```bash
go test ./internal/progress/ -run 'TestRowHealth_RoundTrip|TestRowHealth_OmitemptyKeepsZeroFieldsOut|TestItem_HealthOmitemptyByDefault' -v
```

Expected:

- `PASS` for all three.

- [ ] **Step 1.5: Verify no existing tests regressed**

Run:

```bash
go test ./internal/progress/...
go vet ./internal/progress/...
gofmt -l internal/progress/
```

Expected:

- All tests pass; vet clean; no gofmt diffs.

- [ ] **Step 1.6: Commit**

```bash
git add internal/progress/health.go internal/progress/health_test.go internal/progress/progress.go
git commit -m "feat(progress): add RowHealth schema for autoloop"
```

---

## Task 2: Add `ItemSpecHash` And `ApplyHealthUpdates`

**Files:**
- Modify: `internal/progress/health.go`
- Modify: `internal/progress/health_test.go`
- Create: `internal/progress/health_compat_test.go`
- Create: `internal/progress/health_concurrent_test.go`

- [ ] **Step 2.1: Write failing tests for `ItemSpecHash`**

Append to `internal/progress/health_test.go`:

```go
func TestItemSpecHash_StableAcrossInvocations(t *testing.T) {
	item := &Item{
		Name:           "row-a",
		Status:         StatusInProgress,
		Contract:       "do the thing",
		ContractStatus: ContractStatusFixtureReady,
		BlockedBy:      []string{"row-b", "row-c"},
		WriteScope:     []string{"internal/foo/", "internal/bar/"},
		Fixture:        "internal/foo/foo_test.go",
	}
	a := ItemSpecHash(item)
	b := ItemSpecHash(item)
	if a != b {
		t.Fatalf("hash not deterministic: %s vs %s", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("expected sha256 hex (64 chars), got %d: %s", len(a), a)
	}
}

func TestItemSpecHash_IgnoresBlockedByOrdering(t *testing.T) {
	a := &Item{Name: "x", BlockedBy: []string{"a", "b", "c"}}
	b := &Item{Name: "x", BlockedBy: []string{"c", "b", "a"}}
	if ItemSpecHash(a) != ItemSpecHash(b) {
		t.Fatal("hash should be order-independent for BlockedBy")
	}
}

func TestItemSpecHash_IgnoresStatusAndName(t *testing.T) {
	a := &Item{Name: "row-a", Status: StatusPlanned, Contract: "x"}
	b := &Item{Name: "row-b", Status: StatusComplete, Contract: "x"}
	if ItemSpecHash(a) != ItemSpecHash(b) {
		t.Fatal("hash should ignore Name and Status")
	}
}

func TestItemSpecHash_ChangesWhenContractChanges(t *testing.T) {
	a := &Item{Name: "x", Contract: "old"}
	b := &Item{Name: "x", Contract: "new"}
	if ItemSpecHash(a) == ItemSpecHash(b) {
		t.Fatal("hash should change when Contract changes")
	}
}
```

- [ ] **Step 2.2: Run the failing tests**

Run:

```bash
go test ./internal/progress/ -run TestItemSpecHash -v
```

Expected:

- `FAIL` because `ItemSpecHash` does not exist.

- [ ] **Step 2.3: Implement `ItemSpecHash`**

Append to `internal/progress/health.go`:

```go
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// ItemSpecHash returns a stable SHA-256 hex digest of the row's spec fields
// used for quarantine auto-clear detection. Excludes Name, Status, Health.
// BlockedBy and WriteScope are sorted before hashing so reorderings don't
// invalidate quarantine.
func ItemSpecHash(item *Item) string {
	type specView struct {
		Contract       string         `json:"contract,omitempty"`
		ContractStatus ContractStatus `json:"contract_status,omitempty"`
		BlockedBy      []string       `json:"blocked_by,omitempty"`
		WriteScope     []string       `json:"write_scope,omitempty"`
		Fixture        string         `json:"fixture,omitempty"`
	}
	view := specView{
		Contract:       item.Contract,
		ContractStatus: item.ContractStatus,
		BlockedBy:      append([]string(nil), item.BlockedBy...),
		WriteScope:     append([]string(nil), item.WriteScope...),
		Fixture:        item.Fixture,
	}
	sort.Strings(view.BlockedBy)
	sort.Strings(view.WriteScope)

	body, _ := json.Marshal(view)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Step 2.4: Verify spec-hash tests pass**

Run:

```bash
go test ./internal/progress/ -run TestItemSpecHash -v
```

Expected:

- `PASS` for all four.

- [ ] **Step 2.5: Write failing tests for `ApplyHealthUpdates`**

Append to `internal/progress/health_test.go`:

```go
import (
	"os"
	"path/filepath"
)

func writeProgressJSON(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

const minimalProgress = `{
  "version": "1",
  "phases": {
    "1": {
      "name": "Phase One",
      "subphases": {
        "1.A": {
          "name": "Sub A",
          "items": [
            {"name": "item-1", "status": "planned"}
          ]
        }
      }
    }
  }
}
`

func TestApplyHealthUpdates_AddsHealthBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	writeProgressJSON(t, path, minimalProgress)

	err := ApplyHealthUpdates(path, []HealthUpdate{
		{
			PhaseID:    "1",
			SubphaseID: "1.A",
			ItemName:   "item-1",
			Mutate: func(h *RowHealth) {
				h.AttemptCount = 2
				h.ConsecutiveFailures = 2
				h.LastAttempt = "2026-04-24T12:00:00Z"
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyHealthUpdates: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(body), `"health":`) {
		t.Fatalf("expected health block in output, got:\n%s", body)
	}
	if !strings.Contains(string(body), `"attempt_count": 2`) {
		t.Fatalf("expected attempt_count=2, got:\n%s", body)
	}
}

func TestApplyHealthUpdates_PreservesOtherRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	writeProgressJSON(t, path, `{
  "version": "1",
  "phases": {
    "1": {
      "name": "Phase One",
      "subphases": {
        "1.A": {
          "name": "Sub A",
          "items": [
            {"name": "item-1", "status": "planned", "contract": "do A"},
            {"name": "item-2", "status": "in_progress", "contract": "do B"}
          ]
        }
      }
    }
  }
}
`)

	err := ApplyHealthUpdates(path, []HealthUpdate{
		{PhaseID: "1", SubphaseID: "1.A", ItemName: "item-2", Mutate: func(h *RowHealth) {
			h.AttemptCount = 1
		}},
	})
	if err != nil {
		t.Fatalf("ApplyHealthUpdates: %v", err)
	}

	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), `"contract": "do A"`) {
		t.Fatalf("item-1 contract dropped:\n%s", body)
	}
	if !strings.Contains(string(body), `"contract": "do B"`) {
		t.Fatalf("item-2 contract dropped:\n%s", body)
	}
}

func TestApplyHealthUpdates_UnknownRowReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	writeProgressJSON(t, path, minimalProgress)

	err := ApplyHealthUpdates(path, []HealthUpdate{
		{PhaseID: "9", SubphaseID: "9.Z", ItemName: "ghost", Mutate: func(h *RowHealth) {}},
	})
	if err == nil {
		t.Fatal("expected error when target row does not exist")
	}
}
```

- [ ] **Step 2.6: Run failing `ApplyHealthUpdates` tests**

Run:

```bash
go test ./internal/progress/ -run TestApplyHealthUpdates -v
```

Expected:

- `FAIL` because `HealthUpdate` and `ApplyHealthUpdates` do not exist.

- [ ] **Step 2.7: Implement `HealthUpdate`, `ApplyHealthUpdates`, and `SaveProgress`**

Append to `internal/progress/health.go`:

```go
import (
	"fmt"
	"os"
	"path/filepath"
)

// HealthUpdate is one mutation in a batched run-end write.
type HealthUpdate struct {
	PhaseID    string
	SubphaseID string
	ItemName   string
	// Mutate receives the current Health pointer (never nil — a fresh
	// zero-value RowHealth is allocated if the row has no health block yet).
	Mutate func(current *RowHealth)
}

// ApplyHealthUpdates loads progress.json, applies a batch of mutations in
// memory, and writes the file back atomically (temp + rename). Returns an
// error if any update targets a row that does not exist; the file is left
// untouched on error.
func ApplyHealthUpdates(path string, updates []HealthUpdate) error {
	prog, err := Load(path)
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}

	for _, upd := range updates {
		item, err := findItem(prog, upd.PhaseID, upd.SubphaseID, upd.ItemName)
		if err != nil {
			return fmt.Errorf("apply update %s/%s/%s: %w", upd.PhaseID, upd.SubphaseID, upd.ItemName, err)
		}
		if item.Health == nil {
			item.Health = &RowHealth{}
		}
		upd.Mutate(item.Health)
	}

	return SaveProgress(path, prog)
}

// SaveProgress writes the Progress document atomically: marshal to a temp
// file in the target directory, then rename. Stable key ordering is provided
// by the typed structs.
func SaveProgress(path string, prog *Progress) error {
	body, err := json.MarshalIndent(prog, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal progress: %w", err)
	}
	body = append(body, '\n')

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".progress-*.json")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp into place: %w", err)
	}
	return nil
}

// findItem returns a pointer to the Item inside prog identified by the IDs.
// Returns an error if any segment is missing.
func findItem(prog *Progress, phaseID, subphaseID, itemName string) (*Item, error) {
	if prog == nil || prog.Phases == nil {
		return nil, fmt.Errorf("progress has no phases")
	}
	phase, ok := prog.Phases[phaseID]
	if !ok {
		return nil, fmt.Errorf("phase %q not found", phaseID)
	}
	sub, ok := phase.Subphases[subphaseID]
	if !ok {
		return nil, fmt.Errorf("subphase %q not found in phase %q", subphaseID, phaseID)
	}
	for i := range sub.Items {
		if sub.Items[i].Name == itemName {
			return &sub.Items[i], nil
		}
	}
	return nil, fmt.Errorf("item %q not found in subphase %q", itemName, subphaseID)
}
```

- [ ] **Step 2.8: Verify `ApplyHealthUpdates` tests pass**

Run:

```bash
go test ./internal/progress/ -run TestApplyHealthUpdates -v
```

Expected:

- `PASS` for all three.

- [ ] **Step 2.9: Add the backwards-compatibility round-trip test**

Create `internal/progress/health_compat_test.go`:

```go
package progress

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestApplyHealthUpdates_RoundTripPreservesCheckedInProgressJSON loads the
// real checked-in progress.json, runs an empty update set through
// ApplyHealthUpdates, and asserts the file is byte-equal modulo trailing
// whitespace. Catches any field reordering or formatting drift.
func TestApplyHealthUpdates_RoundTripPreservesCheckedInProgressJSON(t *testing.T) {
	src := filepath.Join("..", "..", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	original, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("checked-in progress.json not found, skipping compat test: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(tmp, original, 0o644); err != nil {
		t.Fatalf("write tmp copy: %v", err)
	}

	if err := ApplyHealthUpdates(tmp, nil); err != nil {
		t.Fatalf("ApplyHealthUpdates with empty updates: %v", err)
	}

	got, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read tmp after round-trip: %v", err)
	}

	a := strings.TrimRight(string(original), "\n")
	b := strings.TrimRight(string(got), "\n")
	if a == b {
		return
	}
	// Surface a small unified diff at the first divergence.
	for i := 0; i < min(len(a), len(b)); i++ {
		if a[i] != b[i] {
			start := max(0, i-50)
			endA := min(len(a), i+50)
			endB := min(len(b), i+50)
			t.Fatalf("round-trip changed bytes at offset %d:\nORIG: %q\nGOT : %q", i,
				bytes.ReplaceAll([]byte(a[start:endA]), []byte("\n"), []byte("\\n")),
				bytes.ReplaceAll([]byte(b[start:endB]), []byte("\n"), []byte("\\n")))
		}
	}
	t.Fatalf("round-trip changed length: orig=%d got=%d", len(a), len(b))
}
```

- [ ] **Step 2.10: Run the compat test and fix any drift**

Run:

```bash
go test ./internal/progress/ -run TestApplyHealthUpdates_RoundTripPreservesCheckedInProgressJSON -v
```

Expected:

- `PASS`. If it fails, `Item` field ordering is off, or `MarshalIndent` is reordering nested keys. Move new fields to end of struct (this is the only safe place to add fields without churn).

- [ ] **Step 2.11: Add the concurrent-write safety test**

Create `internal/progress/health_concurrent_test.go`:

```go
package progress

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestApplyHealthUpdates_ConcurrentDisjointKeysAllSucceed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")

	// Build a progress doc with N rows.
	const N = 8
	body := `{"version":"1","phases":{"1":{"name":"P","subphases":{"1.A":{"name":"S","items":[`
	for i := 0; i < N; i++ {
		if i > 0 {
			body += ","
		}
		body += fmt.Sprintf(`{"name":"row-%d","status":"planned"}`, i)
	}
	body += `]}}}}}`
	writeProgressJSON(t, path, body)

	var wg sync.WaitGroup
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := ApplyHealthUpdates(path, []HealthUpdate{{
				PhaseID:    "1",
				SubphaseID: "1.A",
				ItemName:   fmt.Sprintf("row-%d", idx),
				Mutate: func(h *RowHealth) {
					h.AttemptCount = idx + 1
				},
			}})
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent write returned error: %v", err)
		}
	}

	// File must still be parseable; no partial state.
	_, err := Load(path)
	if err != nil {
		t.Fatalf("Load after concurrent writes: %v", err)
	}
	// At least one update must have survived (last writer wins; others may be lost
	// because the helper has no locking — that's expected single-writer semantics).
	body2, _ := os.ReadFile(path)
	if !contains(string(body2), `"attempt_count":`) {
		t.Fatalf("no health block survived concurrent writes:\n%s", body2)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2.12: Run the concurrent test**

Run:

```bash
go test ./internal/progress/ -run TestApplyHealthUpdates_ConcurrentDisjointKeysAllSucceed -race -v
```

Expected:

- `PASS`. The `-race` flag must also report no data races (atomic write means no concurrent writers see partial state, even without locking).

- [ ] **Step 2.13: Verify the full progress package still passes**

Run:

```bash
go test ./internal/progress/...
go vet ./internal/progress/...
gofmt -l internal/progress/
```

Expected:

- All tests pass; vet clean; no gofmt diffs.

- [ ] **Step 2.14: Commit**

```bash
git add internal/progress/health.go internal/progress/health_test.go internal/progress/health_compat_test.go internal/progress/health_concurrent_test.go
git commit -m "feat(progress): atomic health merge IO + spec hash"
```

---

## Task 3: Add `healthAccumulator` In `internal/autoloop`

**Files:**
- Create: `internal/autoloop/health_writer.go`
- Create: `internal/autoloop/health_writer_test.go`

- [ ] **Step 3.1: Write failing tests for the accumulator**

Create `internal/autoloop/health_writer_test.go`:

```go
package autoloop

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func writeBaseProgress(t *testing.T, path string) {
	t.Helper()
	body := `{
  "version": "1",
  "phases": {
    "2": {
      "name": "P",
      "subphases": {
        "2.B": {
          "name": "S",
          "items": [
            {"name": "row-1", "status": "planned", "contract": "do x"},
            {"name": "row-2", "status": "planned", "contract": "do y"}
          ]
        }
      }
    }
  }
}
`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func fixedNow() func() time.Time {
	t := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func candidateOf(phase, sub, item, contract string) Candidate {
	return Candidate{
		PhaseID:    phase,
		SubphaseID: sub,
		ItemName:   item,
		Contract:   contract,
	}
}

func TestHealthAccumulator_RecordSuccessSetsLastSuccessAndResetsCounter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	writeBaseProgress(t, path)

	acc := newHealthAccumulator("run-A", fixedNow(), 3)
	acc.RecordSuccess(candidateOf("2", "2.B", "row-1", "do x"))
	if err := acc.Flush(path); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	prog, err := progress.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	row := prog.Phases["2"].Subphases["2.B"].Items[0]
	if row.Health == nil {
		t.Fatal("row.Health should be set")
	}
	if row.Health.LastSuccess != "2026-04-24T12:00:00Z" {
		t.Fatalf("LastSuccess = %q", row.Health.LastSuccess)
	}
	if row.Health.ConsecutiveFailures != 0 {
		t.Fatalf("ConsecutiveFailures should be 0, got %d", row.Health.ConsecutiveFailures)
	}
	if row.Health.Quarantine != nil {
		t.Fatalf("Quarantine should be nil after success, got %+v", row.Health.Quarantine)
	}
}

func TestHealthAccumulator_RecordFailureIncrementsConsecutive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	writeBaseProgress(t, path)

	acc := newHealthAccumulator("run-A", fixedNow(), 3)
	acc.RecordFailure(candidateOf("2", "2.B", "row-1", "do x"), progress.FailureWorkerError, "codexu", "boom")
	if err := acc.Flush(path); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	prog, _ := progress.Load(path)
	row := prog.Phases["2"].Subphases["2.B"].Items[0]
	if row.Health.ConsecutiveFailures != 1 {
		t.Fatalf("ConsecutiveFailures = %d, want 1", row.Health.ConsecutiveFailures)
	}
	if row.Health.AttemptCount != 1 {
		t.Fatalf("AttemptCount = %d, want 1", row.Health.AttemptCount)
	}
	if row.Health.Quarantine != nil {
		t.Fatalf("should not quarantine on first failure, got %+v", row.Health.Quarantine)
	}
}

func TestHealthAccumulator_QuarantinesAfterThresholdConsecutiveFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	writeBaseProgress(t, path)

	// Pre-load existing health: 2 consecutive failures already.
	if err := progress.ApplyHealthUpdates(path, []progress.HealthUpdate{{
		PhaseID:    "2",
		SubphaseID: "2.B",
		ItemName:   "row-1",
		Mutate: func(h *progress.RowHealth) {
			h.AttemptCount = 2
			h.ConsecutiveFailures = 2
		},
	}}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	acc := newHealthAccumulator("run-A", fixedNow(), 3)
	acc.RecordFailure(candidateOf("2", "2.B", "row-1", "do x"), progress.FailureReportValidation, "codexu", "report parse failed")
	if err := acc.Flush(path); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	prog, _ := progress.Load(path)
	row := prog.Phases["2"].Subphases["2.B"].Items[0]
	if row.Health.ConsecutiveFailures != 3 {
		t.Fatalf("ConsecutiveFailures = %d, want 3", row.Health.ConsecutiveFailures)
	}
	if row.Health.Quarantine == nil {
		t.Fatal("expected Quarantine to be set after threshold")
	}
	if row.Health.Quarantine.LastCategory != progress.FailureReportValidation {
		t.Fatalf("Quarantine.LastCategory = %q", row.Health.Quarantine.LastCategory)
	}
	if row.Health.Quarantine.SpecHash == "" {
		t.Fatal("Quarantine.SpecHash should be set")
	}
}

func TestHealthAccumulator_SuccessAfterFailuresClearsQuarantine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	writeBaseProgress(t, path)

	// Pre-quarantine the row.
	if err := progress.ApplyHealthUpdates(path, []progress.HealthUpdate{{
		PhaseID:    "2",
		SubphaseID: "2.B",
		ItemName:   "row-1",
		Mutate: func(h *progress.RowHealth) {
			h.ConsecutiveFailures = 3
			h.Quarantine = &progress.Quarantine{Reason: "auto", Threshold: 3, SpecHash: "abc"}
		},
	}}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	acc := newHealthAccumulator("run-A", fixedNow(), 3)
	acc.RecordSuccess(candidateOf("2", "2.B", "row-1", "do x"))
	if err := acc.Flush(path); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	prog, _ := progress.Load(path)
	row := prog.Phases["2"].Subphases["2.B"].Items[0]
	if row.Health.Quarantine != nil {
		t.Fatalf("Quarantine should be cleared after success, got %+v", row.Health.Quarantine)
	}
	if row.Health.ConsecutiveFailures != 0 {
		t.Fatalf("ConsecutiveFailures should reset on success, got %d", row.Health.ConsecutiveFailures)
	}
}

func TestHealthAccumulator_StaleQuarantineClearsAndResetsCounter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	writeBaseProgress(t, path)

	// Quarantine row-1 with a SpecHash that does NOT match its current spec.
	if err := progress.ApplyHealthUpdates(path, []progress.HealthUpdate{{
		PhaseID:    "2",
		SubphaseID: "2.B",
		ItemName:   "row-1",
		Mutate: func(h *progress.RowHealth) {
			h.ConsecutiveFailures = 5
			h.Quarantine = &progress.Quarantine{Reason: "auto", Threshold: 3, SpecHash: "stale-hash"}
		},
	}}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	acc := newHealthAccumulator("run-A", fixedNow(), 3)
	// Mark row as stale-quarantine-detected by selection (no attempt this run).
	acc.MarkStaleQuarantine(candidateOf("2", "2.B", "row-1", "do x"))
	if err := acc.Flush(path); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	prog, _ := progress.Load(path)
	row := prog.Phases["2"].Subphases["2.B"].Items[0]
	if row.Health.Quarantine != nil {
		t.Fatalf("Quarantine should be cleared after stale-quarantine signal, got %+v", row.Health.Quarantine)
	}
	if row.Health.ConsecutiveFailures != 0 {
		t.Fatalf("ConsecutiveFailures should reset on stale-clear, got %d", row.Health.ConsecutiveFailures)
	}
}
```

- [ ] **Step 3.2: Run the failing accumulator tests**

Run:

```bash
go test ./internal/autoloop/ -run TestHealthAccumulator -v
```

Expected:

- `FAIL` because `newHealthAccumulator`, `RecordSuccess`, `RecordFailure`, `MarkStaleQuarantine`, `Flush` do not exist.

- [ ] **Step 3.3: Implement `healthAccumulator`**

Create `internal/autoloop/health_writer.go`:

```go
package autoloop

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

type rowKey struct {
	phaseID    string
	subphaseID string
	itemName   string
}

type pendingHealth struct {
	successes     int
	failures      int
	lastCategory  progress.FailureCategory
	lastBackend   string
	lastStderr    string
	staleClear    bool
	contract      string // captured for later spec-hash recompute
}

type healthAccumulator struct {
	runID     string
	now       func() time.Time
	threshold int
	rows      map[rowKey]*pendingHealth
}

func newHealthAccumulator(runID string, now func() time.Time, threshold int) *healthAccumulator {
	if threshold <= 0 {
		threshold = 3
	}
	return &healthAccumulator{
		runID:     runID,
		now:       now,
		threshold: threshold,
		rows:      map[rowKey]*pendingHealth{},
	}
}

func (a *healthAccumulator) get(c Candidate) *pendingHealth {
	key := rowKey{c.PhaseID, c.SubphaseID, c.ItemName}
	p, ok := a.rows[key]
	if !ok {
		p = &pendingHealth{contract: c.Contract}
		a.rows[key] = p
	}
	return p
}

// RecordSuccess marks one successful worker outcome for the candidate.
func (a *healthAccumulator) RecordSuccess(c Candidate) {
	a.get(c).successes++
}

// RecordFailure marks one failed worker outcome for the candidate.
func (a *healthAccumulator) RecordFailure(c Candidate, cat progress.FailureCategory, backend, stderrTail string) {
	p := a.get(c)
	p.failures++
	p.lastCategory = cat
	p.lastBackend = backend
	p.lastStderr = capStderr(stderrTail, 2048)
}

// MarkStaleQuarantine records that L3 selection treated this candidate as
// stale-quarantined (spec hash mismatch). Used at flush to clear the block.
func (a *healthAccumulator) MarkStaleQuarantine(c Candidate) {
	a.get(c).staleClear = true
}

// Flush applies all accumulated mutations to progress.json in one batched
// write. The mutate closures own all the quarantine math; this is the single
// place where ConsecutiveFailures is incremented or reset.
func (a *healthAccumulator) Flush(progressPath string) error {
	if len(a.rows) == 0 {
		return nil
	}

	now := a.now().UTC().Format(time.RFC3339)
	updates := make([]progress.HealthUpdate, 0, len(a.rows))
	for key, pending := range a.rows {
		p := pending // capture for closure
		k := key     // capture for closure
		updates = append(updates, progress.HealthUpdate{
			PhaseID:    k.phaseID,
			SubphaseID: k.subphaseID,
			ItemName:   k.itemName,
			Mutate: func(h *progress.RowHealth) {
				a.applyMutation(h, p, now)
			},
		})
	}

	// We need the current spec hash for any quarantine we may set; do that
	// inside the mutate closure using progress.ItemSpecHash on the live row.
	// To make that available, the mutate closure needs the row pointer — but
	// HealthUpdate.Mutate only receives *RowHealth. We resolve this by
	// asking the progress package to also expose the row to the closure;
	// the simplest design keeps the spec hash computation here using the
	// captured contract, and the actual hash is recomputed at apply time
	// inside ApplyHealthUpdates if needed. Since we already capture contract,
	// see applyMutation below.
	return progress.ApplyHealthUpdates(progressPath, updates)
}

func (a *healthAccumulator) applyMutation(h *progress.RowHealth, p *pendingHealth, now string) {
	// Stale quarantine clear: reset both block and counter, do NOT touch LastSuccess.
	if p.staleClear && h.Quarantine != nil {
		h.Quarantine = nil
		h.ConsecutiveFailures = 0
	}

	if p.failures > 0 {
		h.AttemptCount += p.failures
		h.LastAttempt = now
		h.LastFailure = &progress.FailureSummary{
			RunID:      a.runID,
			Category:   p.lastCategory,
			Backend:    p.lastBackend,
			StderrTail: p.lastStderr,
		}
		if p.lastBackend != "" && !containsString(h.BackendsTried, p.lastBackend) {
			h.BackendsTried = append(h.BackendsTried, p.lastBackend)
		}
	}

	if p.successes > 0 {
		h.AttemptCount += p.successes
		h.LastAttempt = now
		h.LastSuccess = now
		h.ConsecutiveFailures = 0
		h.Quarantine = nil
		return
	}

	if p.failures > 0 {
		h.ConsecutiveFailures += p.failures
	}

	if h.Quarantine == nil && h.ConsecutiveFailures >= a.threshold && p.failures > 0 {
		// Caller (run.go) is responsible for passing the row's current spec hash
		// when quarantine triggers. For Task 3, we leave SpecHash empty if the
		// caller did not pre-compute it; Task 4 wires the real spec-hash flow.
		h.Quarantine = &progress.Quarantine{
			Reason:       quarantineReason(h.ConsecutiveFailures, p.lastCategory),
			Since:        now,
			AfterRunID:   a.runID,
			Threshold:    a.threshold,
			LastCategory: p.lastCategory,
		}
	}
}

func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func capStderr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

func quarantineReason(consecutive int, cat progress.FailureCategory) string {
	if cat == "" {
		return "auto: " + itoa(consecutive) + " consecutive failures"
	}
	return "auto: " + itoa(consecutive) + " consecutive failures, last category " + string(cat)
}

func itoa(n int) string {
	// Avoid pulling in fmt for one number.
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
```

> NOTE on `Candidate`: Task 3 assumes `Candidate` exposes `PhaseID`, `SubphaseID`, `ItemName`, `Contract`. The first three already exist (per `internal/autoloop/candidates.go`). If `Contract` is not yet on `Candidate`, add it as a string field in this task — it's needed for the spec-hash flow and is a one-line struct change with no behavioral impact.

- [ ] **Step 3.4: Re-run accumulator tests**

Run:

```bash
go test ./internal/autoloop/ -run TestHealthAccumulator -v
```

Expected:

- `PASS` for all 5 accumulator tests.
- The quarantine test passes even though `SpecHash` is empty in this task — the test only asserts `LastCategory` and that `Quarantine != nil`. SpecHash population lands in Task 4 when the run loop wires the real spec hash through.

- [ ] **Step 3.5: Verify no regressions and commit**

Run:

```bash
go test ./internal/autoloop/...
go vet ./internal/autoloop/...
gofmt -l internal/autoloop/
```

Expected:

- All pass; vet clean; no gofmt diffs.

```bash
git add internal/autoloop/health_writer.go internal/autoloop/health_writer_test.go
git commit -m "feat(autoloop): per-run health accumulator"
```

---

## Task 4: Wire Accumulator Into `RunOnce` (L2 + L4 Combined Commit)

**Files:**
- Modify: `internal/autoloop/run.go`
- Modify: `internal/autoloop/config.go`
- Modify: `internal/autoloop/backend.go`
- Modify: `internal/autoloop/health_writer.go` (extend Flush to take spec-hash provider)
- Create: `internal/autoloop/backend_degrader_test.go`
- Modify: `internal/autoloop/run_test.go` (add new behavior tests; do not break existing)

- [ ] **Step 4.1: Write failing tests for the soft-skip and spec-hash quarantine flow**

Append to `internal/autoloop/run_test.go` (or create a new `internal/autoloop/run_health_test.go` if you prefer to keep run_test.go small):

```go
func TestRunOnce_PreflightFailureSoftSkipsAndContinues(t *testing.T) {
	// Set up: 2 candidates; first one fails preflight, second succeeds.
	// Assert: ledger contains "candidate_skipped" event for first, success
	// event for second. RunOnce returns nil (no whole-loop error).
	// (Use the same fake-runner pattern existing run_test.go uses.)
	t.Skip("FILL IN: use the existing run_test.go fixture style; assert the loop continues")
}

func TestRunOnce_QuarantineCarriesCurrentSpecHash(t *testing.T) {
	// Set up: 1 candidate at ConsecutiveFailures=2; this run produces 1 failure.
	// Assert: progress.json on disk has Quarantine.SpecHash == ItemSpecHash(item).
	t.Skip("FILL IN: same fixture style as the test above")
}
```

> The plan-stub above is intentional — the existing `run_test.go` file is 936 lines with a specific fixture/runner harness. Task 4's implementer must follow that harness's idioms (`fakeRunner`, `tempStateDir`, etc.) instead of reinventing them. The test cases to land are listed below; their structure mirrors existing tests.

Concrete test case list (each gets one `Test*` function):

1. `TestRunOnce_PreflightFailureSoftSkipsAndContinues` — first candidate's preflight fails, loop continues to second; ledger has `candidate_skipped` for the first.
2. `TestRunOnce_FailingFastFlagFailsLoopOnPreflight` — set `--fail-fast` (or env), preflight failure returns error.
3. `TestRunOnce_QuarantineCarriesCurrentSpecHash` — verify `Quarantine.SpecHash` equals `progress.ItemSpecHash(item)` for the row.
4. `TestRunOnce_HealthUpdatedEventEmitted` — after Flush, ledger contains a `health_updated` event with `{rows_updated, quarantined, cleared}` summary.
5. `TestRunOnce_HealthUpdateFailedEventOnFlushError` — make `progress.json` read-only mid-run; assert `health_update_failed` event AND RunOnce returns error.

- [ ] **Step 4.2: Run the failing tests**

Run:

```bash
go test ./internal/autoloop/ -run 'TestRunOnce_(PreflightFailureSoftSkipsAndContinues|FailingFastFlagFailsLoopOnPreflight|QuarantineCarriesCurrentSpecHash|HealthUpdatedEventEmitted|HealthUpdateFailedEventOnFlushError)' -v
```

Expected:

- Each new test fails OR is skipped (because the implementation hasn't landed). You will replace the skips with real fixture-style tests in this step.

- [ ] **Step 4.3: Extend `Flush` to receive a spec-hash provider**

Modify `internal/autoloop/health_writer.go`. Change the Flush signature:

```go
// Flush takes a hashProvider so the accumulator can stamp Quarantine.SpecHash
// with the row's CURRENT spec hash at apply time (not at record time, in case
// progress.json changed under us).
type SpecHashProvider func(phaseID, subphaseID, itemName string) string

func (a *healthAccumulator) Flush(progressPath string, hashOf SpecHashProvider) error {
	// ... unchanged scaffolding ...
	updates := make([]progress.HealthUpdate, 0, len(a.rows))
	for key, pending := range a.rows {
		p := pending
		k := key
		updates = append(updates, progress.HealthUpdate{
			PhaseID:    k.phaseID,
			SubphaseID: k.subphaseID,
			ItemName:   k.itemName,
			Mutate: func(h *progress.RowHealth) {
				a.applyMutation(h, p, k, now, hashOf)
			},
		})
	}
	return progress.ApplyHealthUpdates(progressPath, updates)
}
```

And update `applyMutation` to take the `rowKey` + `hashOf` and stamp `Quarantine.SpecHash` when creating quarantine:

```go
func (a *healthAccumulator) applyMutation(h *progress.RowHealth, p *pendingHealth, k rowKey, now string, hashOf SpecHashProvider) {
	// ... existing logic ...
	if h.Quarantine == nil && h.ConsecutiveFailures >= a.threshold && p.failures > 0 {
		specHash := ""
		if hashOf != nil {
			specHash = hashOf(k.phaseID, k.subphaseID, k.itemName)
		}
		h.Quarantine = &progress.Quarantine{
			Reason:       quarantineReason(h.ConsecutiveFailures, p.lastCategory),
			Since:        now,
			AfterRunID:   a.runID,
			Threshold:    a.threshold,
			SpecHash:     specHash,
			LastCategory: p.lastCategory,
		}
	}
}
```

Update Task 3's accumulator tests' `Flush` calls to pass `nil` for `hashOf` (existing tests don't care about SpecHash population).

- [ ] **Step 4.4: Add `backendDegrader`**

Create `internal/autoloop/backend_degrader_test.go`:

```go
package autoloop

import "testing"

func TestBackendDegrader_NoChainNoSwitch(t *testing.T) {
	d := newBackendDegrader([]string{"codexu"}, 3)
	for i := 0; i < 5; i++ {
		d.ObserveOutcome(workerOutcome{IsBackendErrorFlag: true})
	}
	if d.degraded {
		t.Fatal("single-element chain should never degrade")
	}
}

func TestBackendDegrader_DegradesAfterThreshold(t *testing.T) {
	d := newBackendDegrader([]string{"codexu", "claudeu", "opencode"}, 3)
	for i := 0; i < 3; i++ {
		d.ObserveOutcome(workerOutcome{IsBackendErrorFlag: true})
	}
	if !d.degraded {
		t.Fatal("should be degraded after 3 backend errors")
	}
	if d.current != "claudeu" {
		t.Fatalf("current = %q, want claudeu", d.current)
	}
}

func TestBackendDegrader_SuccessResetsCounter(t *testing.T) {
	d := newBackendDegrader([]string{"codexu", "claudeu"}, 3)
	d.ObserveOutcome(workerOutcome{IsBackendErrorFlag: true})
	d.ObserveOutcome(workerOutcome{IsBackendErrorFlag: true})
	d.ObserveOutcome(workerOutcome{IsSuccessFlag: true})
	d.ObserveOutcome(workerOutcome{IsBackendErrorFlag: true})
	if d.degraded {
		t.Fatal("success should have reset the counter")
	}
}

func TestBackendDegrader_RowFailureWithCommitDoesNotDegrade(t *testing.T) {
	d := newBackendDegrader([]string{"codexu", "claudeu"}, 3)
	for i := 0; i < 5; i++ {
		// Row failures with commits are NOT backend errors.
		d.ObserveOutcome(workerOutcome{IsBackendErrorFlag: false})
	}
	if d.degraded {
		t.Fatal("row failures should not degrade backend")
	}
}
```

- [ ] **Step 4.5: Run the failing degrader tests**

Run:

```bash
go test ./internal/autoloop/ -run TestBackendDegrader -v
```

Expected:

- `FAIL` because `backendDegrader`, `newBackendDegrader`, `workerOutcome` do not exist.

- [ ] **Step 4.6: Implement `backendDegrader` and `workerOutcome` in `backend.go`**

Append to `internal/autoloop/backend.go`:

```go
// workerOutcome is the loop's internal classification of a worker's exit.
// IsBackendErrorFlag is true ONLY when the worker produced no commit AND
// no diff AND exited with worker_error (i.e. infra failed, not the row).
type workerOutcome struct {
	IsSuccessFlag      bool
	IsBackendErrorFlag bool
	Commit             string
	DiffLines          int
	Category           string
	Backend            string
}

func (w workerOutcome) IsSuccess() bool      { return w.IsSuccessFlag }
func (w workerOutcome) IsBackendError() bool { return w.IsBackendErrorFlag }

// backendDegrader switches the run loop to the next backend in chain after
// the configured threshold of consecutive backend errors. The chain is
// closed: once degraded past the last entry, further backend errors do not
// trigger another switch.
type backendDegrader struct {
	chain                    []string
	current                  string
	consecutiveBackendErrors int
	threshold                int
	degraded                 bool
}

func newBackendDegrader(chain []string, threshold int) *backendDegrader {
	if threshold <= 0 {
		threshold = 3
	}
	if len(chain) == 0 {
		return &backendDegrader{threshold: threshold}
	}
	return &backendDegrader{
		chain:     chain,
		current:   chain[0],
		threshold: threshold,
	}
}

func (d *backendDegrader) Current() string { return d.current }

func (d *backendDegrader) ObserveOutcome(out workerOutcome) (switched bool, from, to string) {
	if out.IsSuccess() {
		d.consecutiveBackendErrors = 0
		return false, "", ""
	}
	if !out.IsBackendError() {
		return false, "", ""
	}
	d.consecutiveBackendErrors++
	if d.consecutiveBackendErrors < d.threshold {
		return false, "", ""
	}
	if d.degraded {
		return false, "", ""
	}
	idx := indexOf(d.chain, d.current)
	if idx < 0 || idx+1 >= len(d.chain) {
		// No further fallback available.
		d.degraded = true
		return false, "", ""
	}
	previous := d.current
	d.current = d.chain[idx+1]
	d.degraded = true
	return true, previous, d.current
}

func indexOf(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 4.7: Verify degrader tests pass**

Run:

```bash
go test ./internal/autoloop/ -run TestBackendDegrader -v
```

Expected:

- `PASS` for all 4.

- [ ] **Step 4.8: Add new env vars to `internal/autoloop/config.go`**

Modify the existing `ConfigFromEnv` (or matching constructor) to accept and validate:

```go
// New fields on Config (preserve all existing fields):
type Config struct {
	// ... existing fields ...

	QuarantineThreshold       int      // QUARANTINE_THRESHOLD, default 3
	BackendDegradeThreshold   int      // BACKEND_DEGRADE_THRESHOLD, default 3
	BackendFallback           []string // BACKEND_FALLBACK, default empty (no chain)
	IncludeQuarantined        bool     // GORMES_INCLUDE_QUARANTINED, default false
	ReportRepairEnabled       bool     // GORMES_REPORT_REPAIR, default true
	PlannerQuarantineLimit    int      // GORMES_PLANNER_QUARANTINE_LIMIT, default 5 (consumed by L6)
}
```

Default-honoring parse helpers (add to config.go):

```go
func parseInt(env map[string]string, key string, fallback int) int { /* trivial */ }
func parseBool(env map[string]string, key string, fallback bool) bool { /* trivial */ }
func parseList(env map[string]string, key string) []string { /* split on comma, trim */ }
```

Add tests in `config_test.go`:

- defaults match table above
- explicit values are parsed
- `BACKEND_FALLBACK=""` → empty slice (not `[""]`)
- `GORMES_REPORT_REPAIR=0` → false

- [ ] **Step 4.9: Wire accumulator + degrader + soft-skip into `RunOnce`**

Modify `internal/autoloop/run.go`. Key changes:

1. At start of `RunOnce`, instantiate:
   ```go
   acc := newHealthAccumulator(runID, time.Now, opts.Config.QuarantineThreshold)
   chain := opts.Config.BackendFallback
   if len(chain) == 0 {
       chain = []string{opts.Config.Backend}
   }
   degrader := newBackendDegrader(chain, opts.Config.BackendDegradeThreshold)
   ```

2. Build a SpecHashProvider from the loaded `*progress.Progress`:
   ```go
   hashOf := func(phaseID, subphaseID, itemName string) string {
       prog, err := progress.Load(opts.Config.ProgressJSON)
       if err != nil {
           return ""
       }
       phase, ok := prog.Phases[phaseID]
       if !ok { return "" }
       sub, ok := phase.Subphases[subphaseID]
       if !ok { return "" }
       for i := range sub.Items {
           if sub.Items[i].Name == itemName {
               return progress.ItemSpecHash(&sub.Items[i])
           }
       }
       return ""
   }
   ```

3. In the per-candidate loop, after preflight:
   ```go
   for _, candidate := range selected {
       if err := preflightCandidate(candidate); err != nil {
           emitLedger(ledger.Event{Type: "candidate_skipped", Reason: err.Error(), Candidate: candidate})
           acc.RecordFailure(candidate, progress.FailureProgressSummary, "", err.Error())
           if opts.FailFast {
               break // or return err depending on existing semantics
           }
           continue
       }
       outcome := runWorker(candidate, degrader.Current())
       if outcome.IsSuccess() {
           acc.RecordSuccess(candidate)
       } else {
           cat := mapCategory(outcome) // existing helper or inline
           acc.RecordFailure(candidate, cat, outcome.Backend, outcome.StderrTail)
       }
       if switched, from, to := degrader.ObserveOutcome(outcome); switched {
           emitLedger(ledger.Event{Type: "backend_degraded", From: from, To: to})
       }
       if candidate.StaleQuarantine { // L3 will surface this; for now, accept zero-value
           acc.MarkStaleQuarantine(candidate)
       }
   }
   ```

4. At end of `RunOnce`, before return:
   ```go
   if err := acc.Flush(opts.Config.ProgressJSON, hashOf); err != nil {
       emitLedger(ledger.Event{Type: "health_update_failed", Error: err.Error()})
       return summary, err
   }
   emitLedger(ledger.Event{
       Type: "health_updated",
       Detail: map[string]any{
           "rows_updated": len(acc.rows),
           // quarantined / cleared counts can be derived during Flush;
           // for now emit just rows_updated to keep the change minimal.
       },
   })
   ```

- [ ] **Step 4.10: Re-run all autoloop tests**

Run:

```bash
go test ./internal/autoloop/...
go vet ./internal/autoloop/...
gofmt -l internal/autoloop/
```

Expected:

- All pass; vet clean; no gofmt diffs.
- New behavior tests from Step 4.1 pass.
- Existing tests remain green.

- [ ] **Step 4.11: Commit**

```bash
git add internal/autoloop/run.go internal/autoloop/config.go internal/autoloop/backend.go internal/autoloop/backend_degrader_test.go internal/autoloop/health_writer.go internal/autoloop/run_test.go
git commit -m "feat(autoloop): wire health writer + backend degrade into run loop"
```

---

## Task 5: Selection Honors Health (L3)

**Files:**
- Modify: `internal/autoloop/candidates.go`
- Create: `internal/autoloop/candidates_health_test.go`

- [ ] **Step 5.1: Write failing selection tests**

Create `internal/autoloop/candidates_health_test.go`:

```go
package autoloop

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func itemWithHealth(name string, h *progress.RowHealth) progress.Item {
	return progress.Item{Name: name, Status: progress.StatusPlanned, Contract: "do " + name, Health: h}
}

func progressDoc(items ...progress.Item) *progress.Progress {
	return &progress.Progress{
		Phases: map[string]*progress.Phase{
			"1": {
				Name: "P",
				Subphases: map[string]*progress.Subphase{
					"1.A": {Name: "S", Items: items},
				},
			},
		},
	}
}

func TestNormalizeCandidates_NoHealthBehavesLikeBaseline(t *testing.T) {
	doc := progressDoc(
		itemWithHealth("a", nil),
		itemWithHealth("b", nil),
	)
	got := NormalizeCandidates(doc, CandidateOptions{})
	if len(got) != 2 {
		t.Fatalf("got %d candidates, want 2", len(got))
	}
}

func TestNormalizeCandidates_QuarantineFiltersByDefault(t *testing.T) {
	doc := progressDoc(
		itemWithHealth("a", &progress.RowHealth{
			ConsecutiveFailures: 3,
			Quarantine: &progress.Quarantine{
				Threshold: 3,
				SpecHash:  progress.ItemSpecHash(&progress.Item{Name: "a", Contract: "do a"}),
			},
		}),
		itemWithHealth("b", nil),
	)
	got := NormalizeCandidates(doc, CandidateOptions{})
	if len(got) != 1 {
		t.Fatalf("expected 1 (b), got %d", len(got))
	}
	if got[0].ItemName != "b" {
		t.Fatalf("got %q, want b", got[0].ItemName)
	}
}

func TestNormalizeCandidates_IncludeQuarantinedReturnsAll(t *testing.T) {
	doc := progressDoc(
		itemWithHealth("a", &progress.RowHealth{
			ConsecutiveFailures: 3,
			Quarantine: &progress.Quarantine{
				Threshold: 3,
				SpecHash:  progress.ItemSpecHash(&progress.Item{Name: "a", Contract: "do a"}),
			},
		}),
		itemWithHealth("b", nil),
	)
	got := NormalizeCandidates(doc, CandidateOptions{IncludeQuarantined: true})
	if len(got) != 2 {
		t.Fatalf("expected both, got %d", len(got))
	}
}

func TestNormalizeCandidates_StaleQuarantineFlagsAndIncludes(t *testing.T) {
	doc := progressDoc(
		itemWithHealth("a", &progress.RowHealth{
			ConsecutiveFailures: 5,
			Quarantine: &progress.Quarantine{
				Threshold: 3,
				SpecHash:  "some-old-stale-hash",
			},
		}),
	)
	got := NormalizeCandidates(doc, CandidateOptions{})
	if len(got) != 1 {
		t.Fatalf("expected stale quarantine to surface candidate, got %d", len(got))
	}
	if !got[0].StaleQuarantine {
		t.Fatal("StaleQuarantine flag should be true")
	}
}

func TestFailurePenalty_TableDriven(t *testing.T) {
	cases := []struct {
		consecutive int
		want        int
	}{
		{0, 0},
		{1, 5},
		{2, 20},
		{3, 45},
		{10, 45},
	}
	for _, c := range cases {
		got := failurePenalty(c.consecutive)
		if got != c.want {
			t.Errorf("failurePenalty(%d) = %d, want %d", c.consecutive, got, c.want)
		}
	}
}
```

- [ ] **Step 5.2: Run failing selection tests**

Run:

```bash
go test ./internal/autoloop/ -run 'TestNormalizeCandidates_(NoHealthBehavesLikeBaseline|QuarantineFiltersByDefault|IncludeQuarantinedReturnsAll|StaleQuarantineFlagsAndIncludes)|TestFailurePenalty_TableDriven' -v
```

Expected:

- `FAIL` because `failurePenalty`, `Candidate.StaleQuarantine`, `CandidateOptions.IncludeQuarantined` do not exist.

- [ ] **Step 5.3: Implement selection changes in `candidates.go`**

Add:

```go
// failurePenalty returns the ranking penalty for n consecutive failures.
// 0→0, 1→5, 2→20, 3+→45 (cap; rows past threshold should already be
// quarantined, but defensive cap covers manual-override scenarios).
func failurePenalty(n int) int {
	switch {
	case n <= 0:
		return 0
	case n == 1:
		return 5
	case n == 2:
		return 20
	default:
		return 45
	}
}
```

Add fields to `Candidate`:

```go
type Candidate struct {
	// ... existing fields ...
	Health          *progress.RowHealth `json:"health,omitempty"`
	StaleQuarantine bool                `json:"stale_quarantine,omitempty"`
	PenaltyApplied  int                 `json:"penalty_applied,omitempty"`
}
```

Add field to `CandidateOptions`:

```go
type CandidateOptions struct {
	// ... existing fields ...
	IncludeQuarantined bool
}
```

Modify `NormalizeCandidates` (the existing entry point):

```go
// During iteration over items, after constructing each Candidate:
candidate.Health = item.Health
if item.Health != nil && item.Health.Quarantine != nil {
	currentHash := progress.ItemSpecHash(item)
	if currentHash != item.Health.Quarantine.SpecHash {
		candidate.StaleQuarantine = true
		// fall through; do NOT exclude.
	} else if !opts.IncludeQuarantined {
		// Excluded — skip this candidate entirely. Optionally emit a
		// "quarantine_excluded" trace via opts (debug surface).
		continue
	}
}
// During scoring (existing ranking pipeline), after the base score:
if item.Health != nil {
	pen := failurePenalty(item.Health.ConsecutiveFailures)
	pen += 2 * len(item.Health.BackendsTried)
	score -= pen
	candidate.PenaltyApplied = pen
}
```

Update `Candidate.SelectionReason()` to append `" penalty=N"` when `PenaltyApplied > 0` and `" quarantine_stale_cleared"` when `StaleQuarantine`.

- [ ] **Step 5.4: Re-run selection tests**

Run:

```bash
go test ./internal/autoloop/ -run 'TestNormalizeCandidates|TestFailurePenalty' -v
```

Expected:

- `PASS` for all five.

- [ ] **Step 5.5: Wire `Config.IncludeQuarantined` into the call site**

In `RunOnce` (or wherever `NormalizeCandidates` is invoked), pass `opts.Config.IncludeQuarantined` into `CandidateOptions{IncludeQuarantined: …}`.

Same for `cmd/autoloop` dry-run printing — when a candidate has `StaleQuarantine` or `PenaltyApplied`, surface it in the dry-run output. The existing `runAutoloop` loop in `cmd/autoloop/main.go` already prints `reason=`; append the new reason fragments via the updated `SelectionReason()`.

- [ ] **Step 5.6: Verify and commit**

Run:

```bash
go test ./internal/autoloop/...
go vet ./internal/autoloop/...
gofmt -l internal/autoloop/
```

Expected:

- All pass; vet clean; no gofmt diffs.

```bash
git add internal/autoloop/candidates.go internal/autoloop/candidates_health_test.go internal/autoloop/run.go
git commit -m "feat(autoloop): selection honors row health"
```

---

## Task 6: Report Repair Pass (L5)

**Files:**
- Modify: `internal/autoloop/report.go`
- Modify: `internal/autoloop/promote.go`
- Modify: `internal/autoloop/report_test.go` (add repair tests; do not weaken existing strict tests)
- Modify: `internal/autoloop/run.go` (wire repair into promotion path)

- [ ] **Step 6.1: Write failing tests for `TryRepairReport`**

Append to `internal/autoloop/report_test.go`:

```go
func TestTryRepairReport_NoCommitFails(t *testing.T) {
	dir := t.TempDir()
	// Init an empty repo at dir (no commits).
	mustGitInit(t, dir)
	rep, _, err := TryRepairReport(RepairContext{
		WorkerStdout: "PASS\nok",
		WorktreePath: dir,
		BaseBranch:   "main",
	})
	if err == nil || rep != nil {
		t.Fatalf("expected repair to fail with no commit; got rep=%v err=%v", rep, err)
	}
}

func TestTryRepairReport_NoPassFails(t *testing.T) {
	dir := setupRepoWithCommit(t)
	rep, _, err := TryRepairReport(RepairContext{
		WorkerStdout: "FAIL: foo",
		WorktreePath: dir,
		BaseBranch:   "main",
	})
	if err == nil || rep != nil {
		t.Fatalf("expected repair to fail without PASS token")
	}
}

func TestTryRepairReport_AcceptanceMissingFails(t *testing.T) {
	dir := setupRepoWithCommit(t)
	rep, _, err := TryRepairReport(RepairContext{
		WorkerStdout:    "ok\nPASS",
		WorktreePath:    dir,
		BaseBranch:      "main",
		AcceptanceLines: []string{"acceptance-line-A"},
	})
	if err == nil || rep != nil {
		t.Fatalf("expected repair to fail when acceptance line missing")
	}
}

func TestTryRepairReport_AcceptanceEmptyAcceptsOnPassEvidence(t *testing.T) {
	dir := setupRepoWithCommit(t)
	rep, notes, err := TryRepairReport(RepairContext{
		WorkerStdout: "all good\nPASS\nok\n",
		WorktreePath: dir,
		BaseBranch:   "main",
		// AcceptanceLines empty → fallback rule: accept on PASS evidence.
	})
	if err != nil || rep == nil {
		t.Fatalf("expected repair to succeed, got err=%v rep=%v", err, rep)
	}
	if len(notes) == 0 {
		t.Fatal("expected at least one RepairNote")
	}
	if rep.Commit == "" {
		t.Fatal("expected reconstructed commit")
	}
}

func TestTryRepairReport_AllAcceptanceLinesPresentAccepts(t *testing.T) {
	dir := setupRepoWithCommit(t)
	rep, _, err := TryRepairReport(RepairContext{
		WorkerStdout:    "acceptance-A done\nacceptance-B done\nPASS\nok",
		WorktreePath:    dir,
		BaseBranch:      "main",
		AcceptanceLines: []string{"acceptance-A", "acceptance-B"},
	})
	if err != nil || rep == nil {
		t.Fatalf("expected repair to succeed with all acceptance lines, got err=%v", err)
	}
}
```

> Helpers `mustGitInit` and `setupRepoWithCommit` should be added to a small test helpers file or inline. They run `git init`, set user.email/user.name (test config), commit a tiny file. Existing `worktree_test.go` likely has a similar helper — reuse if present, else add.

- [ ] **Step 6.2: Run failing repair tests**

Run:

```bash
go test ./internal/autoloop/ -run TestTryRepairReport -v
```

Expected:

- `FAIL` because `TryRepairReport`, `RepairContext`, `RepairNote` do not exist.

- [ ] **Step 6.3: Implement `TryRepairReport`**

Add to `internal/autoloop/report.go`:

```go
import (
	"errors"
	"os/exec"
	"strings"
)

// RepairContext bundles the secondary evidence sources TryRepairReport uses
// to reconstruct a FinalReport when ParseFinalReport fails.
type RepairContext struct {
	WorkerStdout    string
	WorkerStderr    string
	WorktreePath    string
	BaseBranch      string
	AcceptanceLines []string
}

// RepairNote records one piece of evidence used during reconstruction.
type RepairNote struct {
	Field  string
	Source string
	Detail string
}

// TryRepairReport reconstructs a FinalReport from secondary evidence when
// ParseFinalReport fails. Returns (nil, nil, error) when the worker did not
// actually produce sound work — strictly never accepts work without:
//   1. A new commit on the worker's branch
//   2. A non-empty diff
//   3. At least one PASS token in the stdout
//   4. Either every acceptance line appears in stdout, OR no acceptance set
func TryRepairReport(ctx RepairContext) (*FinalReport, []RepairNote, error) {
	if ctx.WorktreePath == "" {
		return nil, nil, errors.New("repair: WorktreePath required")
	}

	notes := []RepairNote{}

	commit, err := gitLastCommit(ctx.WorktreePath, ctx.BaseBranch)
	if err != nil || commit == "" {
		return nil, nil, errors.New("repair: no commit on worker branch")
	}
	notes = append(notes, RepairNote{Field: "commit", Source: "git_log", Detail: commit})

	diff, err := gitDiff(ctx.WorktreePath, ctx.BaseBranch)
	if err != nil || strings.TrimSpace(diff) == "" {
		return nil, nil, errors.New("repair: empty diff")
	}

	if !strings.Contains(ctx.WorkerStdout, "PASS") {
		return nil, nil, errors.New("repair: no PASS token in stdout")
	}
	notes = append(notes, RepairNote{Field: "evidence", Source: "stdout_grep", Detail: "found PASS token"})

	if len(ctx.AcceptanceLines) > 0 {
		for _, line := range ctx.AcceptanceLines {
			if !strings.Contains(ctx.WorkerStdout, line) {
				return nil, nil, errors.New("repair: acceptance line missing: " + line)
			}
		}
		notes = append(notes, RepairNote{Field: "acceptance", Source: "stdout_grep", Detail: "matched all acceptance lines"})
	} else {
		notes = append(notes, RepairNote{Field: "acceptance", Source: "fallback", Detail: "no acceptance lines required"})
	}

	report := &FinalReport{
		Commit:        commit,
		Diff:          diff,
		// Best-effort excerpts — exact fields come from the existing FinalReport
		// shape; populate what's available without inventing data.
	}
	return report, notes, nil
}

func gitLastCommit(dir, baseBranch string) (string, error) {
	args := []string{"-C", dir, "log", "--format=%H", "-1"}
	if baseBranch != "" {
		args = []string{"-C", dir, "log", "--format=%H", baseBranch + "..HEAD", "-1"}
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitDiff(dir, baseBranch string) (string, error) {
	args := []string{"-C", dir, "diff"}
	if baseBranch != "" {
		args = []string{"-C", dir, "diff", baseBranch + "..HEAD"}
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
```

- [ ] **Step 6.4: Verify repair tests pass**

Run:

```bash
go test ./internal/autoloop/ -run TestTryRepairReport -v
```

Expected:

- `PASS` for all 5.

- [ ] **Step 6.5: Wire repair pass into the promotion path**

Modify `internal/autoloop/run.go` (or wherever the strict parse happens for worker output). After:

```go
report, err := ParseFinalReport(workerStdout)
```

Add:

```go
if err != nil {
	if opts.Config.ReportRepairEnabled {
		repaired, notes, repErr := TryRepairReport(RepairContext{
			WorkerStdout:    workerStdout,
			WorkerStderr:    workerStderr,
			WorktreePath:    workerWorktree,
			BaseBranch:      opts.Config.BaseBranch,
			AcceptanceLines: candidate.Acceptance,
		})
		if repErr == nil && repaired != nil {
			report = repaired
			err = nil
			emitLedger(ledger.Event{
				Type:      "report_repaired",
				Candidate: candidate,
				Notes:     notes,
			})
			writeRepairArtifact(opts.Config.RunRoot, runID, workerID, candidate, repaired, notes, workerStdout)
			// Health: still record this as a report_validation_failed signal so
			// the planner sees the row produces noisy output.
			acc.RecordFailure(candidate, progress.FailureReportValidation, outcome.Backend, "")
			// (record the failure BEFORE marking success so accumulator carries
			// both a success and a failure for this run; success math wins.)
			acc.RecordSuccess(candidate)
		} else {
			// Repair declined or failed; fall through to existing failure path.
			emitLedger(ledger.Event{Type: "report_repair_failed", Reason: repErr.Error()})
		}
	}
}
```

> Important: the dual record (`RecordFailure` + `RecordSuccess` for the same row in the same run) is intentional. Per the spec's L5 section: *"`Health.LastFailure` is recorded as `report_validation_failed` so the planner can see the row produces messy reports (data point, not a death sentence)."* The accumulator's success-resets-counter rule means LastSuccess will be set, ConsecutiveFailures will be 0, but LastFailure will reflect the noisy report — exactly what the planner needs to see.

Add `writeRepairArtifact` to `internal/autoloop/promote.go`:

```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func writeRepairArtifact(runRoot, runID, workerID string, c Candidate, rep *FinalReport, notes []RepairNote, stdout string) {
	dir := filepath.Join(runRoot, "state", "repairs")
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.json", runID, workerID))

	tail := stdout
	if len(tail) > 4096 {
		tail = tail[len(tail)-4096:]
	}

	body := map[string]any{
		"candidate":        c,
		"commit":           rep.Commit,
		"diff_lines":       countLines(rep.Diff),
		"notes":            notes,
		"stdout_excerpt":   tail,
	}
	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func countLines(s string) int {
	n := 0
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}
```

- [ ] **Step 6.6: Add the `GORMES_REPORT_REPAIR=0` disable test**

Append to `internal/autoloop/run_test.go` (or a new health-focused test file):

- `TestRunOnce_ReportRepairDisabledFallsThroughToFailure` — set `Config.ReportRepairEnabled = false`, simulate a worker whose strict parse fails but secondary evidence would have repaired; assert ledger has no `report_repaired` event and the candidate is marked failed.

- [ ] **Step 6.7: Verify and commit**

Run:

```bash
go test ./internal/autoloop/...
go vet ./internal/autoloop/...
gofmt -l internal/autoloop/
```

Expected:

- All pass; vet clean; no gofmt diffs.

```bash
git add internal/autoloop/report.go internal/autoloop/report_test.go internal/autoloop/promote.go internal/autoloop/run.go internal/autoloop/run_test.go
git commit -m "feat(autoloop): report repair pass salvages noisy worker output"
```

---

## Task 7: Planner Consumes Health (L6)

**Files:**
- Modify: `internal/architectureplanner/context.go`
- Modify: `internal/architectureplanner/prompt.go`
- Modify: `internal/architectureplanner/run.go`
- Create: `internal/architectureplanner/health_preservation_test.go`

- [ ] **Step 7.1: Write failing tests for `validateHealthPreservation`**

Create `internal/architectureplanner/health_preservation_test.go`:

```go
package architectureplanner

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func docWithItem(item progress.Item) *progress.Progress {
	return &progress.Progress{
		Phases: map[string]*progress.Phase{
			"1": {
				Name: "P",
				Subphases: map[string]*progress.Subphase{
					"1.A": {Name: "S", Items: []progress.Item{item}},
				},
			},
		},
	}
}

func TestValidateHealthPreservation_IdenticalAccepted(t *testing.T) {
	h := &progress.RowHealth{AttemptCount: 3, ConsecutiveFailures: 1}
	before := docWithItem(progress.Item{Name: "x", Status: progress.StatusInProgress, Contract: "c", Health: h})
	after := docWithItem(progress.Item{Name: "x", Status: progress.StatusInProgress, Contract: "c", Health: h})
	if err := validateHealthPreservation(before, after); err != nil {
		t.Fatalf("expected accepted, got %v", err)
	}
}

func TestValidateHealthPreservation_ModifiedHealthRejected(t *testing.T) {
	before := docWithItem(progress.Item{Name: "x", Contract: "c", Health: &progress.RowHealth{AttemptCount: 3}})
	after := docWithItem(progress.Item{Name: "x", Contract: "c", Health: &progress.RowHealth{AttemptCount: 99}})
	if err := validateHealthPreservation(before, after); err == nil {
		t.Fatal("expected error when health was modified")
	}
}

func TestValidateHealthPreservation_DroppedHealthRejected(t *testing.T) {
	before := docWithItem(progress.Item{Name: "x", Contract: "c", Health: &progress.RowHealth{AttemptCount: 3}})
	after := docWithItem(progress.Item{Name: "x", Contract: "c", Health: nil})
	if err := validateHealthPreservation(before, after); err == nil {
		t.Fatal("expected error when health was dropped")
	}
}

func TestValidateHealthPreservation_DeletedRowAccepted(t *testing.T) {
	before := docWithItem(progress.Item{Name: "x", Contract: "c", Health: &progress.RowHealth{AttemptCount: 3}})
	// after: same phase/subphase but no items
	after := &progress.Progress{
		Phases: map[string]*progress.Phase{
			"1": {Name: "P", Subphases: map[string]*progress.Subphase{"1.A": {Name: "S", Items: nil}}},
		},
	}
	if err := validateHealthPreservation(before, after); err != nil {
		t.Fatalf("deletion should be accepted, got %v", err)
	}
}

func TestValidateHealthPreservation_SplitRowAccepted(t *testing.T) {
	before := docWithItem(progress.Item{Name: "x", Contract: "umbrella", Health: &progress.RowHealth{AttemptCount: 3}})
	after := &progress.Progress{
		Phases: map[string]*progress.Phase{
			"1": {Name: "P", Subphases: map[string]*progress.Subphase{
				"1.A": {Name: "S", Items: []progress.Item{
					{Name: "x-a", Contract: "split a"},
					{Name: "x-b", Contract: "split b"},
				}},
			}},
		},
	}
	if err := validateHealthPreservation(before, after); err != nil {
		t.Fatalf("split (rename) should be accepted, got %v", err)
	}
}

func TestValidateHealthPreservation_SpecChangedHealthPreservedAccepted(t *testing.T) {
	h := &progress.RowHealth{AttemptCount: 3}
	before := docWithItem(progress.Item{Name: "x", Contract: "old", Health: h})
	after := docWithItem(progress.Item{Name: "x", Contract: "NEW SPEC", Health: h})
	if err := validateHealthPreservation(before, after); err != nil {
		t.Fatalf("spec change with health preserved should be accepted, got %v", err)
	}
}
```

- [ ] **Step 7.2: Run failing tests**

Run:

```bash
go test ./internal/architectureplanner/ -run TestValidateHealthPreservation -v
```

Expected:

- `FAIL` because `validateHealthPreservation` does not exist.

- [ ] **Step 7.3: Implement `validateHealthPreservation`**

Append to `internal/architectureplanner/run.go`:

```go
import (
	"fmt"
	"reflect"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

// validateHealthPreservation rejects planner regenerations that drop or
// modify any existing Health block. Rows missing from the after-doc are
// considered intentional deletions (planner removed them) and pass.
func validateHealthPreservation(before, after *progress.Progress) error {
	beforeIndex := indexItems(before)
	afterIndex := indexItems(after)

	for key, beforeItem := range beforeIndex {
		afterItem, exists := afterIndex[key]
		if !exists {
			// Row was deleted intentionally; health dies with row.
			continue
		}
		if !healthEqual(beforeItem.Health, afterItem.Health) {
			return fmt.Errorf("planner output dropped or modified health block for %s/%s/%s", key.phaseID, key.subphaseID, key.itemName)
		}
	}
	return nil
}

type itemKey struct{ phaseID, subphaseID, itemName string }

func indexItems(prog *progress.Progress) map[itemKey]*progress.Item {
	out := map[itemKey]*progress.Item{}
	if prog == nil {
		return out
	}
	for phaseID, phase := range prog.Phases {
		for subID, sub := range phase.Subphases {
			for i := range sub.Items {
				it := &sub.Items[i]
				out[itemKey{phaseID, subID, it.Name}] = it
			}
		}
	}
	return out
}

func healthEqual(a, b *progress.RowHealth) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}
```

- [ ] **Step 7.4: Verify validator tests pass**

Run:

```bash
go test ./internal/architectureplanner/ -run TestValidateHealthPreservation -v
```

Expected:

- `PASS` for all 6.

- [ ] **Step 7.5: Wire validator into planner `RunOnce`**

In `internal/architectureplanner/run.go`, in the existing `RunOnce` (or whatever runs the LLM and writes the new doc), after the LLM produces a new `*progress.Progress` and before saving:

```go
if err := validateHealthPreservation(beforeDoc, afterDoc); err != nil {
	return summary, fmt.Errorf("reject planner regeneration: %w", err)
}
```

Where `beforeDoc` is the doc loaded at the start of the planner run.

- [ ] **Step 7.6: Add `QuarantinedRowContext` to `context.go`**

Modify `internal/architectureplanner/context.go`:

```go
type QuarantinedRowContext struct {
	PhaseID            string                   `json:"phase_id"`
	SubphaseID         string                   `json:"subphase_id"`
	ItemName           string                   `json:"item_name"`
	Contract           string                   `json:"contract,omitempty"`
	LastCategory       progress.FailureCategory `json:"last_category,omitempty"`
	AttemptCount       int                      `json:"attempt_count,omitempty"`
	BackendsTried      []string                 `json:"backends_tried,omitempty"`
	QuarantinedSince   string                   `json:"quarantined_since,omitempty"`
	SpecHash           string                   `json:"spec_hash,omitempty"`
	LastFailureExcerpt string                   `json:"last_failure_excerpt,omitempty"`
	AuditCorroboration string                   `json:"audit_corroboration,omitempty"`
}

// collectQuarantinedRows returns up to limit quarantined rows sorted by
// (AttemptCount desc, QuarantinedSince asc). Pass limit=0 for unlimited.
func collectQuarantinedRows(prog *progress.Progress, audit AutoloopAudit, limit int) []QuarantinedRowContext {
	out := []QuarantinedRowContext{}
	if prog == nil {
		return out
	}
	for phaseID, phase := range prog.Phases {
		for subID, sub := range phase.Subphases {
			for i := range sub.Items {
				it := &sub.Items[i]
				if it.Health == nil || it.Health.Quarantine == nil {
					continue
				}
				excerpt := ""
				if it.Health.LastFailure != nil {
					excerpt = capExcerpt(it.Health.LastFailure.StderrTail, 1024)
				}
				out = append(out, QuarantinedRowContext{
					PhaseID:            phaseID,
					SubphaseID:         subID,
					ItemName:           it.Name,
					Contract:           it.Contract,
					LastCategory:       it.Health.Quarantine.LastCategory,
					AttemptCount:       it.Health.AttemptCount,
					BackendsTried:      append([]string(nil), it.Health.BackendsTried...),
					QuarantinedSince:   it.Health.Quarantine.Since,
					SpecHash:           it.Health.Quarantine.SpecHash,
					LastFailureExcerpt: excerpt,
					AuditCorroboration: corroborateFromAudit(audit, phaseID, subID, it.Name),
				})
			}
		}
	}
	// Sort: AttemptCount desc, QuarantinedSince asc.
	sort.Slice(out, func(i, j int) bool {
		if out[i].AttemptCount != out[j].AttemptCount {
			return out[i].AttemptCount > out[j].AttemptCount
		}
		return out[i].QuarantinedSince < out[j].QuarantinedSince
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func capExcerpt(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

// corroborateFromAudit returns a short note when SummarizeAutoloopAudit
// already flagged this row as toxic/hot. Returns "" if no corroboration.
func corroborateFromAudit(audit AutoloopAudit, phaseID, subID, itemName string) string {
	// Implementation detail: scan the existing audit struct for matching keys.
	// Returns a single human-readable line like "audit: 5 attempts in last 24h".
	// If AutoloopAudit doesn't have a row-level lookup, return "" — the row
	// already has its own attempt history in Health.
	return ""
}
```

Then thread `collectQuarantinedRows` into `CollectContext` so the planner sees the list:

```go
// In CollectContext (existing function):
ctx.QuarantinedRows = collectQuarantinedRows(prog, ctx.AutoloopAudit, cfg.PlannerQuarantineLimit)
```

Add `QuarantinedRows []QuarantinedRowContext` to whatever Context struct CollectContext returns.

Add a test in `internal/architectureplanner/context_test.go` (or extend existing):

- `TestCollectQuarantinedRows_SortsByAttemptCountThenSince` — feed 3 rows, assert order.
- `TestCollectQuarantinedRows_HonorsLimit` — feed 10 rows, limit=5, assert len==5.
- `TestCollectQuarantinedRows_ExcludesNonQuarantined` — mixed list, only quarantined surface.

- [ ] **Step 7.7: Add prompt clauses to `prompt.go`**

Modify `internal/architectureplanner/prompt.go`. Append to the existing prompt template (or as new sections rendered by the existing template engine):

```go
const healthPreservationClause = `
HEALTH BLOCK PRESERVATION (HARD RULE)
Every progress.json item may carry a ` + "`health`" + ` block (RowHealth). This block
is OWNED by the autoloop runtime — you must reproduce it verbatim in your
output for any row you keep. Do not modify, omit, or reformat any field
inside ` + "`health`" + `. If you delete a row, the health block dies with it (that
is expected). If you split a row into multiple new rows, the original
health block is dropped (the split is a new contract; quarantine resets
naturally via spec-hash detection).
`

const quarantinePriorityClause = `
QUARANTINE PRIORITY (SOFT RULE)
Rows in quarantined_rows[] are top priority for repair. For each one:
  - Read its last_category and last_failure_excerpt
  - Examine its contract and acceptance
  - Decide ONE of:
    (a) Sharpen the contract — make done_signal more concrete, add an
        explicit fixture path, narrow write_scope
    (b) Split the row — if it's an umbrella that workers can't complete
        atomically, split into 2-3 smaller rows with explicit dependencies
    (c) Mark it for human review — if the failure is infrastructural
        (category=worker_error or backend_degraded with no diff), set
        contract_status: "draft" and add a note in degraded_mode
        explaining what's needed
  Whatever you choose, the row's contract/contract_status/blocked_by/
  write_scope/fixture must change in some material way. Otherwise
  quarantine will not auto-clear and autoloop will keep skipping the row.
`
```

Wire both clauses into the prompt render function. Add a test:

- `TestRenderPrompt_IncludesHealthClauses` — render the prompt with a non-empty quarantined list; assert both clauses appear in the output.

- [ ] **Step 7.8: Verify and commit**

Run:

```bash
go test ./internal/architectureplanner/...
go vet ./internal/architectureplanner/...
gofmt -l internal/architectureplanner/
```

Expected:

- All pass; vet clean; no gofmt diffs.

```bash
git add internal/architectureplanner/context.go internal/architectureplanner/prompt.go internal/architectureplanner/run.go internal/architectureplanner/health_preservation_test.go internal/architectureplanner/context_test.go
git commit -m "feat(planner): consume row health and preserve across regen"
```

---

## Task 8: End-To-End Lifecycle Cross-Cutting Test

**Files:**
- Create: `internal/autoloop/lifecycle_test.go`

- [ ] **Step 8.1: Write the lifecycle test**

Create `internal/autoloop/lifecycle_test.go`:

```go
package autoloop

import (
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

// TestLifecycle_FailingRowQuarantinesThenPlannerRepairUnlocksIt walks one
// row through the full reactive-autoloop loop:
//   Run 1: row attempted, fails → ConsecutiveFailures=1
//   Run 2: row attempted, fails → ConsecutiveFailures=2
//   Run 3: row attempted, fails → ConsecutiveFailures=3, quarantine set
//   Run 4: row in candidate pool but excluded by quarantine filter
//   Planner edit: contract changed (simulated by direct progress.json mutation)
//   Run 5: row in pool, stale-quarantine flagged, attempted, succeeds
//          → quarantine cleared, ConsecutiveFailures=0, LastSuccess set
//
// Uses a deterministic fake worker that returns success/failure on demand
// and a fake clock so timestamps are predictable.
func TestLifecycle_FailingRowQuarantinesThenPlannerRepairUnlocksIt(t *testing.T) {
	dir := t.TempDir()
	progressPath := filepath.Join(dir, "progress.json")
	writeBaseProgress(t, progressPath)
	// (Implementation note: reuse fixtures from run_test.go if helpful.
	// The point of this test is verifying the COMPOSITION of all five layers
	// — not re-testing each layer's logic in isolation.)

	threshold := 3

	// Run 1
	acc := newHealthAccumulator("R1", fixedNow(), threshold)
	acc.RecordFailure(candidateOf("2", "2.B", "row-1", "do x"), progress.FailureWorkerError, "codexu", "")
	if err := acc.Flush(progressPath, nil); err != nil {
		t.Fatalf("R1 flush: %v", err)
	}
	prog, _ := progress.Load(progressPath)
	if prog.Phases["2"].Subphases["2.B"].Items[0].Health.ConsecutiveFailures != 1 {
		t.Fatal("R1: expected CF=1")
	}

	// Run 2
	acc = newHealthAccumulator("R2", fixedNow(), threshold)
	acc.RecordFailure(candidateOf("2", "2.B", "row-1", "do x"), progress.FailureWorkerError, "codexu", "")
	acc.Flush(progressPath, nil)

	// Run 3 (threshold hit)
	acc = newHealthAccumulator("R3", fixedNow(), threshold)
	acc.RecordFailure(candidateOf("2", "2.B", "row-1", "do x"), progress.FailureWorkerError, "codexu", "")
	hashOf := func(p, s, n string) string {
		prog, _ := progress.Load(progressPath)
		for i := range prog.Phases[p].Subphases[s].Items {
			if prog.Phases[p].Subphases[s].Items[i].Name == n {
				return progress.ItemSpecHash(&prog.Phases[p].Subphases[s].Items[i])
			}
		}
		return ""
	}
	acc.Flush(progressPath, hashOf)
	prog, _ = progress.Load(progressPath)
	if prog.Phases["2"].Subphases["2.B"].Items[0].Health.Quarantine == nil {
		t.Fatal("R3: expected quarantine to be set")
	}
	originalHash := prog.Phases["2"].Subphases["2.B"].Items[0].Health.Quarantine.SpecHash

	// Run 4: selection excludes the row.
	got := NormalizeCandidates(prog, CandidateOptions{})
	if len(got) != 1 {
		t.Fatalf("R4: expected 1 candidate (row-2 only), got %d", len(got))
	}
	if got[0].ItemName != "row-2" {
		t.Fatalf("R4: expected row-2, got %s", got[0].ItemName)
	}

	// Planner edit: change row-1's contract → spec hash will differ.
	if err := progress.ApplyHealthUpdates(progressPath, nil); err != nil {
		t.Fatalf("noop reload: %v", err)
	}
	prog, _ = progress.Load(progressPath)
	prog.Phases["2"].Subphases["2.B"].Items[0].Contract = "do x — sharpened by planner"
	if err := progress.SaveProgress(progressPath, prog); err != nil {
		t.Fatalf("save planner edit: %v", err)
	}

	// Run 5: stale-quarantine surfaces row-1, then it succeeds.
	prog, _ = progress.Load(progressPath)
	got = NormalizeCandidates(prog, CandidateOptions{})
	var staleCand Candidate
	for _, c := range got {
		if c.ItemName == "row-1" {
			staleCand = c
			break
		}
	}
	if staleCand.ItemName == "" || !staleCand.StaleQuarantine {
		t.Fatalf("R5: expected row-1 with StaleQuarantine flag, got %+v", staleCand)
	}

	acc = newHealthAccumulator("R5", fixedNow(), threshold)
	acc.MarkStaleQuarantine(staleCand)
	acc.RecordSuccess(candidateOf("2", "2.B", "row-1", "do x — sharpened by planner"))
	acc.Flush(progressPath, hashOf)

	prog, _ = progress.Load(progressPath)
	row := prog.Phases["2"].Subphases["2.B"].Items[0]
	if row.Health.Quarantine != nil {
		t.Fatalf("R5: quarantine should be cleared, got %+v", row.Health.Quarantine)
	}
	if row.Health.ConsecutiveFailures != 0 {
		t.Fatalf("R5: ConsecutiveFailures should reset, got %d", row.Health.ConsecutiveFailures)
	}
	if row.Health.LastSuccess == "" {
		t.Fatal("R5: LastSuccess should be set")
	}
	_ = originalHash // referenced for clarity; not asserted post-clear
}
```

- [ ] **Step 8.2: Run lifecycle test**

Run:

```bash
go test ./internal/autoloop/ -run TestLifecycle_FailingRowQuarantinesThenPlannerRepairUnlocksIt -v
```

Expected:

- `PASS`. If it fails, the failure points to a composition bug between layers — diagnose by isolating which run's assertion failed.

- [ ] **Step 8.3: Run all package tests one final time**

Run:

```bash
go test ./internal/progress/... ./internal/autoloop/... ./internal/architectureplanner/...
go vet ./...
gofmt -l .
```

Expected:

- All pass; vet clean; no gofmt diffs anywhere in the repo.

- [ ] **Step 8.4: Commit**

```bash
git add internal/autoloop/lifecycle_test.go
git commit -m "test(autoloop): end-to-end reactive lifecycle"
```

---

## Self-Review Checklist

### Spec coverage

- L1 schema + IO: Task 1 (RowHealth + Health field) + Task 2 (ItemSpecHash + ApplyHealthUpdates + compat + concurrent)
- L2 run-loop writes health: Task 3 (accumulator) + Task 4 (wired into RunOnce)
- L3 selection honors health: Task 5
- L4 loop resilience (soft-skip + backend degrade): Task 4
- L5 report repair pass: Task 6
- L6 planner consumes health: Task 7
- Cross-cutting tests: Task 2 (compat + concurrent), Task 8 (lifecycle), per-layer tests embedded in each task

### Placeholder scan

- Task 4 has a deliberate skip-stub (`t.Skip("FILL IN: …")`) for the run_test.go-style behavior tests because the existing `run_test.go` is 936 lines with a specific fixture harness; the implementer must follow that harness's idioms instead of inventing new ones. The required test cases are enumerated explicitly. This is the ONLY skip-stub in the plan.
- Task 7 has placeholder `corroborateFromAudit` returning `""` if `AutoloopAudit` doesn't expose row-level lookup; this is a documented soft-fail (the row's own Health already carries attempt history).
- No `TBD`, `TODO`, "implement later" markers anywhere else.
- Every code-changing step contains the actual code.
- Every test step contains the actual test commands.

### Type / API consistency

Names used across tasks (verify each is referenced consistently):
- `progress.RowHealth`, `progress.FailureSummary`, `progress.Quarantine`, `progress.FailureCategory`
- `progress.FailureWorkerError`, `FailureReportValidation`, `FailureProgressSummary`, `FailureTimeout`, `FailureBackendDegraded`
- `progress.ItemSpecHash(*Item) string`
- `progress.HealthUpdate{PhaseID, SubphaseID, ItemName, Mutate}`
- `progress.ApplyHealthUpdates(path string, updates []HealthUpdate) error`
- `progress.SaveProgress(path string, prog *Progress) error`
- `autoloop.healthAccumulator` (unexported) + `newHealthAccumulator`, `RecordSuccess`, `RecordFailure`, `MarkStaleQuarantine`, `Flush`
- `autoloop.SpecHashProvider func(phaseID, subphaseID, itemName string) string`
- `autoloop.workerOutcome{IsSuccessFlag, IsBackendErrorFlag, Commit, DiffLines, Category, Backend}` + `IsSuccess()`, `IsBackendError()`
- `autoloop.backendDegrader` + `newBackendDegrader`, `Current`, `ObserveOutcome`
- `autoloop.failurePenalty(int) int`
- `autoloop.Candidate.Health`, `.StaleQuarantine`, `.PenaltyApplied`
- `autoloop.CandidateOptions.IncludeQuarantined`
- `autoloop.RepairContext`, `autoloop.RepairNote`, `autoloop.TryRepairReport`
- `architectureplanner.QuarantinedRowContext`, `collectQuarantinedRows`, `validateHealthPreservation`
- Config: `QuarantineThreshold`, `BackendDegradeThreshold`, `BackendFallback`, `IncludeQuarantined`, `ReportRepairEnabled`, `PlannerQuarantineLimit`
- New ledger event types: `candidate_skipped`, `backend_degraded`, `health_updated`, `health_update_failed`, `report_repaired`, `report_repair_failed`
- New env vars: `QUARANTINE_THRESHOLD`, `BACKEND_DEGRADE_THRESHOLD`, `BACKEND_FALLBACK`, `GORMES_INCLUDE_QUARANTINED`, `GORMES_REPORT_REPAIR`, `GORMES_PLANNER_QUARANTINE_LIMIT`

All names cross-reference correctly between tasks.
