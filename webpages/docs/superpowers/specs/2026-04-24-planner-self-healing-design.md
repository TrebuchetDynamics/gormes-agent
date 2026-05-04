# Planner Self-Healing Design

**Status:** Draft
**Author:** Codex (Claude Opus 4.7 1M)
**Date:** 2026-04-24

## Context

The architecture-planner-loop (`cmd/architecture-planner-loop`,
`internal/architectureplanner`) refines `progress.json` to reflect upstream
changes, current Gormes implementation reality, and (post-Phase B) autoloop's
quarantined rows. The planner runs on a fixed 6-hour systemd timer; it has no
event-driven cadence, no per-run history beyond "latest" artifacts, no retry
mechanism when its output is rejected, no self-evaluation of whether its
reshapes actually unstuck rows, no escalation path for intractable rows, and
no way to focus a run on a specific topic ("Honcho", "memory", "skills tools").

Phase B added autoloop's `Health` block and the planner's preservation
contract for it. Phase C closes the loop in the other direction: gives the
planner reactivity, observability, retry resilience, self-feedback, escalation,
and topical focus.

Five recurring effectiveness gaps inform this design:

1. **Latency.** A row quarantined at 03:01 is invisible to the planner until
   the next 6h timer fires.
2. **No retry on rejection.** When `validateHealthPreservation` rejects a
   regen, the run is wasted; the next chance is 6h later.
3. **No planner-side ledger.** Only "latest" artifacts exist, so the planner
   cannot evaluate its own effectiveness.
4. **No self-evaluation.** The planner reshapes a row and forgets — it has no
   feedback on whether autoloop later succeeded.
5. **No human escalation.** Rows that resist N reshapes have no operational
   signal pathway out.

Plus one operator-facing gap surfaced during brainstorm:

6. **No topical focus.** Operators cannot say "this run, focus only on
   Honcho-related rows."

## Goals

1. React to autoloop quarantine events within minutes, not hours.
2. Recover from validation rejections within a single planner run.
3. Persist a queryable history of every planner run.
4. Correlate planner reshapes with autoloop outcomes and feed the result back
   into the next prompt.
5. Mark intractable rows for human review with a clear, sticky signal.
6. Let operators run topical planner passes via keyword arguments.
7. Ship as five independently-shippable commits so each layer can be
   reverted in isolation if it regresses.

## Non-goals

- External notification channels (Slack, GitHub issues, email) for
  escalations. The status CLI surface is the only Phase C signal.
- Adaptive planner cadence (the 6h timer stays; event-driven runs are
  additive).
- Coordination locks between planner/autoloop runs (Phase B atomic IO is
  enough).
- Preview-via-autoloop-dry-run before accepting planner output.
- Live LLM tests against backends; all tests use mock runners.
- Topical event triggers (autoloop emits ALL events; topical narrowing is
  user-driven only).
- Auto-unsetting `NeedsHuman` after K successful runs (sticky by design;
  humans clear it explicitly).

## Decision Summary

The accepted design is **Planner Self-Healing** — six layers added to the
planner runtime, with small extensions to autoloop and the progress schema.

```
┌────────────────────────────────────────────────────────────────────────────┐
│  progress.json                                                             │
│  - Item.Health        (autoloop owns; planner preserves) ← Phase B         │
│  - Item.PlannerVerdict (planner owns; autoloop preserves) ← Phase C        │
└──────────────────────────────────┬──────────────────────┬──────────────────┘
                                   │ planner reads/writes │ autoloop reads/writes
                                   ▼                      ▼
   ┌───────────────────────────────────┐    ┌────────────────────────────────┐
   │ architecture-planner-loop          │    │  autoloop run loop              │
   │  - reads Health (Phase B)          │    │  - emits triggers.jsonl on    ←─┘
   │  - writes PlannerVerdict           │    │    quarantine_added /            │
   │  - retry-on-rejection (L3) ───────►│    │    quarantine_stale_cleared    (L2)
   │  - self-evaluation (L4)            │    │  - skips PlannerVerdict.       │
   │  - escalates after N reshapes (L5) │    │    NeedsHuman rows (L5)        │
   │  - topical focus on keywords (L6)  │    │                                │
   └────────┬──────────────────┬────────┘    └─────────────┬──────────────────┘
            │ writes runs.jsonl │                          │ writes runs.jsonl
            │      (L1)         │                          │
            ▼                   ▼                          ▼
  ┌──────────────────────┐  ┌────────────────────────┐  ┌────────────────────┐
  │ .codex/              │  │ .codex/                │  │ .codex/            │
  │ architecture-planner │  │ architecture-planner/  │  │ orchestrator/      │
  │ /state/runs.jsonl    │  │ triggers.jsonl   ←─────┴─►│ state/runs.jsonl   │
  │ (planner ledger, L1) │  │ (event queue, L2)         │ (autoloop ledger)  │
  │                      │  │ + cursor.json             │                    │
  └──────────────────────┘  └────────────────────────┘  └────────────────────┘
                                       ▲
                                       │ systemd path unit watches mtime
                                       │ → fires planner.service immediately
```

Six layers, five commits (L6 ships with L1's commit since both are small and
share no other dependencies):

| Layer | What it adds | Touches | Commit |
|---|---|---|---|
| **L1 — Planner ledger** | Per-run records: trigger, before-summary, after-diff, validation result, rows changed | new `internal/architectureplanner/ledger.go`, run.go | C1 |
| **L2 — Event trigger** | Autoloop appends `triggers.jsonl`; systemd path unit fires planner; cursor-based consumption | autoloop run.go (emit), new `triggers.go`, service.go (path unit) | C2 |
| **L3 — Retry-with-feedback** | Re-prompt LLM with explicit "you dropped row X" feedback up to N times | run.go, new `retry.go` | C3 |
| **L4 — Self-evaluation** | Each run correlates planner ledger ↔ autoloop ledger; outcomes feed next prompt | new `evaluation.go`, prompt.go | C4 |
| **L5 — `PlannerVerdict` + escalation** | New typed field; sticky `NeedsHuman` after N reshapes; autoloop selection skips it | `internal/progress/progress.go`, `internal/autoloop/candidates.go`, planner run.go, status command | C5 |
| **L6 — Topical focus** | Keyword arguments narrow planner context; LLM gets a topical clause | new `topics.go`, cmd/architecture-planner-loop, prompt.go | C1 (folded) |

L1 is foundational. L2-L5 build on L1 but are otherwise independent.

## Schema Additions

### `PlannerVerdict` on `Item`

```go
// PlannerVerdict is execution-history metadata about one progress.json item,
// OWNED by the architecture-planner runtime. Autoloop READS it (to skip
// rows escalated for human review) and MUST preserve it verbatim across
// writes (structural via typed JSON round-trip).
//
// Symmetric to RowHealth (autoloop-owned + planner-preserved).
type PlannerVerdict struct {
    NeedsHuman   bool   `json:"needs_human,omitempty"`
    Reason       string `json:"reason,omitempty"`
    Since        string `json:"since,omitempty"`         // RFC3339
    ReshapeCount int    `json:"reshape_count,omitempty"`
    LastReshape  string `json:"last_reshape,omitempty"`  // RFC3339
    LastOutcome  string `json:"last_outcome,omitempty"`  // "unstuck" | "still_failing" | "no_attempts_yet"
}

type Item struct {
    // ... existing fields preserved ...
    Health         *RowHealth      `json:"health,omitempty"`
    PlannerVerdict *PlannerVerdict `json:"planner_verdict,omitempty"`
}
```

### Planner ledger entry

`.codex/architecture-planner/state/runs.jsonl` — one JSON object per line:

```go
type LedgerEvent struct {
    TS            string        `json:"ts"`              // RFC3339
    RunID         string        `json:"run_id"`
    Trigger       string        `json:"trigger"`         // "scheduled" | "event" | "manual" | "retry"
    TriggerEvents []string      `json:"trigger_events,omitempty"`
    Backend       string        `json:"backend"`
    Mode          string        `json:"mode"`
    Status        string        `json:"status"`          // "ok" | "validation_rejected" | "backend_failed" | "no_changes" | "needs_human_set"
    Detail        string        `json:"detail,omitempty"`
    BeforeStats   ProgressStats `json:"before_stats,omitempty"`
    AfterStats    ProgressStats `json:"after_stats,omitempty"`
    RowsChanged   []RowChange   `json:"rows_changed,omitempty"`
    RetryAttempt  int           `json:"retry_attempt,omitempty"`
    Keywords      []string      `json:"keywords,omitempty"` // L6
}

type RowChange struct {
    PhaseID    string `json:"phase_id"`
    SubphaseID string `json:"subphase_id"`
    ItemName   string `json:"item_name"`
    Kind       string `json:"kind"` // "added" | "deleted" | "spec_changed" | "verdict_set"
    Detail     string `json:"detail,omitempty"`
}

type ProgressStats struct {
    Shipped     int `json:"shipped"`
    InProgress  int `json:"in_progress"`
    Planned     int `json:"planned"`
    Quarantined int `json:"quarantined"`
    NeedsHuman  int `json:"needs_human"`
}
```

### Trigger ledger + cursor

`.codex/architecture-planner/triggers.jsonl` (autoloop writes, planner reads):

```go
type TriggerEvent struct {
    ID            string `json:"id"`
    TS            string `json:"ts"`
    Source        string `json:"source"`        // "autoloop"
    Kind          string `json:"kind"`          // "quarantine_added" | "quarantine_stale_cleared" | "manual"
    PhaseID       string `json:"phase_id,omitempty"`
    SubphaseID    string `json:"subphase_id,omitempty"`
    ItemName      string `json:"item_name,omitempty"`
    Reason        string `json:"reason,omitempty"`
    AutoloopRunID string `json:"autoloop_run_id,omitempty"`
}

// .codex/architecture-planner/state/triggers_cursor.json
type TriggerCursor struct {
    LastConsumedID string `json:"last_consumed_id"`
    LastReadAt     string `json:"last_read_at"`
}
```

### Schema invariants

1. **Symmetric ownership.** Autoloop never writes `PlannerVerdict`. Planner
   never writes `Health`. Both blocks are preserved structurally via Phase B's
   typed-struct round-trip.
2. **`PlannerVerdict.ReshapeCount` is monotonic.** Resets only on row
   delete or split.
3. **`PlannerVerdict.NeedsHuman` is sticky.** Once set, only a human can
   clear it (by editing `progress.json` directly). Planner never auto-unsets.
4. **Both ledgers are append-only.** No deletion or rotation in Phase C.
5. **Cursor is atomic-replaced** (temp + rename), matching `SaveProgress`.
6. **Trigger events are idempotent on consumption.** Cursor advances after
   processing; reprocessing the same event is harmless (planner's prompt
   bullet list contains the row keys; LLM's reshape decision is naturally
   idempotent).

## L1 — Planner ledger

Establishes the ledger pattern. Reuses autoloop's existing
`AppendLedgerEvent` IO contract (`O_APPEND|O_CREATE|O_WRONLY`).

New file `internal/architectureplanner/ledger.go`:

```go
func AppendLedgerEvent(path string, event LedgerEvent) error
func LoadLedger(path string) ([]LedgerEvent, error)
func LoadLedgerWindow(path string, window time.Duration, now time.Time) ([]LedgerEvent, error)
```

Wire-in to `RunOnce`: after `runValidation` and before return, build a
`LedgerEvent`, compute `BeforeStats`/`AfterStats` and `RowsChanged` via a new
`diffRows(beforeDoc, afterDoc)` helper (reuses Phase B's `indexItems`),
append to `.codex/architecture-planner/state/runs.jsonl`. Failure is
soft-logged — ledger is observability, not the run's success criterion.

## L2 — Event-driven trigger

### Autoloop side: emit triggers

In `internal/autoloop/health_writer.go::Flush`, after `progress.ApplyHealthUpdates`
succeeds, classify each row's transition and emit:

```go
func (a *healthAccumulator) classifyForTrigger(before, after *progress.RowHealth, p *pendingHealth) (kind string, fire bool) {
    if (before == nil || before.Quarantine == nil) && after != nil && after.Quarantine != nil {
        return "quarantine_added", true
    }
    if before != nil && before.Quarantine != nil &&
       after != nil && after.Quarantine == nil &&
       p.staleClear {
        return "quarantine_stale_cleared", true
    }
    return "", false
}
```

The fire-list is appended to `triggers.jsonl` from the run.go `flushHealth`
closure. `Config.PlannerTriggersPath` (env: `PLANNER_TRIGGERS_PATH`,
default `.codex/architecture-planner/triggers.jsonl`).

### Planner side: consume triggers

New file `internal/architectureplanner/triggers.go`:

```go
func AppendTriggerEvent(path string, event TriggerEvent) error
func ReadTriggersSinceCursor(path string, cursor TriggerCursor) ([]TriggerEvent, error)
func LoadCursor(path string) (TriggerCursor, error)
func SaveCursor(path string, cursor TriggerCursor) error
```

In `RunOnce`, before building the prompt: load cursor, read new events,
thread into prompt as a "Recent Autoloop Signals" section. After `RunOnce`
completes (success OR failure), advance cursor.

### systemd: path unit fires planner

New unit `gormes-architecture-planner.path`:

```ini
[Unit]
Description=Trigger Gormes architecture planner on autoloop signal

[Path]
PathChanged=%h/.../.codex/architecture-planner/triggers.jsonl
TriggerLimitIntervalSec=60
TriggerLimitBurst=1
Unit=gormes-architecture-planner.service

[Install]
WantedBy=default.target
```

Rate-limited to one trigger-driven run per 60s. The 6h timer stays as the
deep-pass safety net.

## L3 — Retry-with-feedback

When `validateHealthPreservation(beforeDoc, afterDoc)` rejects a regen, the
planner re-prompts the same LLM up to N times with explicit feedback about
the dropped rows. The strict validator from Phase B is unchanged; we retry
around it.

New file `internal/architectureplanner/retry.go`:

```go
const DefaultMaxRetries = 2

func RetryFeedback(rejection error, beforeDoc, afterDoc *progress.Progress) string

type retryAttempt struct {
    Index       int
    Status      string // "ok" | "validation_rejected" | "backend_failed"
    Detail      string
    DroppedRows []string
}
```

The retry feedback string names the dropped rows, references the HARD rule,
and tells the LLM to skip the upstream-sync analysis on the retry (only fix
the dropped blocks).

`RunOnce` becomes a loop:

```go
for i := 0; i <= maxRetries; i++ {
    invoke backend
    load after-doc
    validate
    if accepted: break
    if i < maxRetries: prompt = initialPrompt + "\n\n" + RetryFeedback(...)
    else: fail run
}
```

The L1 ledger entry's `RetryAttempt` field records the index of the
successful (or final-failed) attempt. A new `attempts []retryAttempt` field
captures the full sequence for forensics.

`Config.MaxRetries` (env: `PLANNER_MAX_RETRIES`, default 2). Backend failure
is NOT retried; only validation rejection.

## L4 — Self-evaluation

Each planner run includes a "look back at my last K reshapes" pass that reads
the planner ledger AND the autoloop ledger, correlates per-row, and reports
outcomes. The outcomes feed the next planner prompt as observational signal.

New file `internal/architectureplanner/evaluation.go`:

```go
const DefaultEvaluationWindow = 7 * 24 * time.Hour

type ReshapeOutcome struct {
    PhaseID            string `json:"phase_id"`
    SubphaseID         string `json:"subphase_id"`
    ItemName           string `json:"item_name"`
    ReshapedAt         string `json:"reshaped_at"`
    ReshapedBy         string `json:"reshaped_by"`
    Outcome            string `json:"outcome"`         // "unstuck" | "still_failing" | "no_attempts_yet"
    AutoloopRuns       int    `json:"autoloop_runs"`
    LastFailure        string `json:"last_failure,omitempty"`
    LastSuccess        string `json:"last_success,omitempty"`
    StaleClearObserved bool   `json:"stale_clear_observed"`
}

func Evaluate(plannerLedgerPath, autoloopLedgerPath string, window time.Duration, now time.Time) ([]ReshapeOutcome, error)
```

Classification rule: for each `RowChange{Kind:"spec_changed"}` in the
planner ledger window, walk autoloop ledger events for the same row AFTER
the reshape TS. Promoted → `unstuck`. Failed-after-promotion → `still_failing`.
No autoloop events → `no_attempts_yet`.

Wire-in to `RunOnce`: call `Evaluate` after `BeforeStats`, thread results
into `ContextBundle.PreviousReshapes`, render via a new "Previous Reshape
Outcomes" section in `BuildPrompt`. A new SOFT prompt clause tells the LLM
how to use the signal:

```text
SELF-EVALUATION (SOFT RULE)

The "Previous Reshape Outcomes" section reports what autoloop did with rows
you reshaped in past runs. Use this signal:
  - UNSTUCK rows confirm your previous approach worked
  - STILL FAILING rows have resisted reshape — try a different decomposition,
    escalate to "needs_human" via PlannerVerdict (L5), or tighten ready_when
  - NO ATTEMPTS YET rows may be legitimately blocked
```

`Config.EvaluationWindow` (env: `PLANNER_EVALUATION_WINDOW`, default `168h`).

## L5 — `PlannerVerdict` + escalation

`PlannerVerdict` is written by a deterministic post-processing pass — NOT
by the LLM. This keeps verdict math out of the LLM's reasoning surface.

New file `internal/architectureplanner/verdict.go`:

```go
const DefaultEscalationThreshold = 3

func StampVerdicts(afterDoc *progress.Progress, rowsChanged []RowChange, outcomes []ReshapeOutcome, threshold int, now time.Time) []RowChange
```

Rules:
- For every row in `rowsChanged{Kind:"spec_changed"}`: increment
  `ReshapeCount`, set `LastReshape = now`.
- For every row in `outcomes`:
  - `unstuck` → `LastOutcome = "unstuck"`; do NOT auto-clear `NeedsHuman`
    (sticky).
  - `still_failing` → `LastOutcome = "still_failing"`; if
    `ReshapeCount >= threshold` AND `!NeedsHuman`, set `NeedsHuman=true`
    with `Reason`, `Since`.
  - `no_attempts_yet` → `LastOutcome = "no_attempts_yet"`.

Returns the list of rows whose verdict materially changed (for L1 ledger
emission as `RowChange{Kind:"verdict_set"}`).

Wire-in to `RunOnce`: after validation passes, before `SaveProgress`. The
single combined save writes BOTH the LLM regen AND the verdict stamps
atomically.

### Autoloop side: skip `NeedsHuman`

In `NormalizeCandidates`, after the existing quarantine filter:

```go
if item.PlannerVerdict != nil && item.PlannerVerdict.NeedsHuman {
    if !opts.IncludeNeedsHuman {
        continue
    }
    candidate.NeedsHumanFlag = true
}
```

`Config.IncludeNeedsHuman` (env: `GORMES_INCLUDE_NEEDS_HUMAN`, default
`false`). Mirrors `IncludeQuarantined` from Phase B exactly.

### Status surface

Extend `architecture-planner-loop status` to print after the existing
metadata lines:

```
Reshape outcomes (last 7d):
  unstuck: 5
  still failing: 2
  no attempts yet: 1

Rows needing human attention: 2
  - 2/2.C/row-3 — auto: 4 reshapes without unsticking; last category report_validation_failed
                  reshape count: 4   since: 2026-04-23T14:00:00Z
                  → suggested action: split into smaller rows or set contract_status="draft"
```

Suggested action mapping by latest `Health.LastFailure.Category`:

| Category | Suggested action |
|---|---|
| `report_validation_failed` | "split into smaller rows or set contract_status='draft'" |
| `worker_error` / `backend_degraded` | "investigate infrastructure (backend or worktree state)" |
| `progress_summary_failed` | "manual contract review — autoloop preflight is failing" |
| `timeout` | "split into smaller rows; the work is too large for the worker budget" |
| (other / empty) | "manual review" |

### How a human clears `NeedsHuman`

Edit `progress.json` directly to remove or set false. Next planner run
re-evaluates; if the row is still failing, the next run will re-set
`NeedsHuman=true`. So a human's clear is "let me try one more time."

`Config.EscalationThreshold` (env: `PLANNER_ESCALATION_THRESHOLD`,
default 3).

## L6 — Topical focus mode

Extend `cmd/architecture-planner-loop` to accept positional keyword
arguments after `run`:

```sh
go run ./cmd/architecture-planner-loop run honcho
go run ./cmd/architecture-planner-loop run memory skills
go run ./cmd/architecture-planner-loop run --codexu "skills tools"
```

Multiple keywords → OR semantics. Whitespace-quoted keywords get split.
No keywords → current full-pass behavior.

New file `internal/architectureplanner/topics.go`:

```go
func MatchKeywords(items []ItemRef, keywords []string) []ItemRef
func FilterContextByKeywords(bundle ContextBundle, keywords []string) ContextBundle
```

Mechanical narrowing rules (case-insensitive substring, OR across keywords):
- `Item.Name`
- `Item.Contract`
- `Item.SourceRefs[]`
- `Item.WriteScope[]`
- `Item.Fixture`
- `Subphase.Name` / `Phase.Name` (matching name brings ALL its items)

`FilterContextByKeywords` narrows `QuarantinedRows`, `PreviousReshapes`,
`Inventory`. Leaves `AutoloopAudit` and `SourceRoots` intact (audit is
aggregate; sources are ground truth).

A new prompt clause when keywords are present:

```text
TOPICAL FOCUS

This run was invoked with keyword arguments: ["honcho", "memory"]. The
context above (Quarantined Rows, Previous Reshapes, Implementation Inventory)
has been narrowed to only rows that mechanically match these keywords.

Focus your refinement work on these areas. You may still adjust adjacent
rows if a topical row's blocked_by/unblocks dependencies require it, but
do NOT widen the scope to unrelated phases.
```

Wire-in to `cmd/architecture-planner-loop/main.go`: extract positional
keyword args from `run [flags] [keywords...]`. Thread into
`RunOptions.Keywords []string`. In `RunOnce`, apply
`FilterContextByKeywords(bundle, keywords)` and pass keywords to
`BuildPrompt`.

L1 ledger entry's `Keywords` field records the focus per run.

L2 event-triggered runs carry NO keywords by default — the trigger IS the
focus signal.

The `architecture-planner-loop status` output adds a `Keywords:` line when
the most recent run was topical.

## Testing

Per-layer tests are enumerated within each layer's section in the
brainstorm. Five cross-cutting test surfaces close interaction gaps:

### 1. End-to-end planner-self-healing lifecycle (`internal/architectureplanner/lifecycle_test.go`, new)

Walks one row through the full Phase C loop:

```
Run 1 (autoloop): 3 failures → quarantine → quarantine_added trigger
Run 2 (planner, event): cursor advances; LLM reshapes; verdict.ReshapeCount=1
Run 3 (autoloop): stale-quarantine flagged; row attempted; fails again →
                  quarantine re-set
Run 4 (planner, event): L4 outcome=still_failing; verdict.ReshapeCount=2
Run 5 (autoloop): row fails again → quarantine re-set
Run 6 (planner, event): verdict.ReshapeCount=3 → NeedsHuman=true;
                        ledger status=needs_human_set
Run 7 (autoloop): selection EXCLUDES row (NeedsHuman=true)
```

Drives the full pipeline using fake runners. No real LLM. Sub-second runtime.

### 2. Symmetric preservation regression (`internal/progress/preservation_test.go`, new)

Tests both directions of the symmetric preservation contract in one place:
autoloop's `ApplyHealthUpdates` preserves `PlannerVerdict`; `SaveProgress`
after a verdict-only edit preserves `Health`. Final file has both blocks
intact AND spec hash stable.

### 3. Trigger ledger + cursor integrity under concurrency (`internal/architectureplanner/triggers_concurrent_test.go`, new)

N=8 writer goroutines (autoloop processes) + M=2 reader goroutines (planner
runs) operate on `triggers.jsonl` and the cursor simultaneously. All events
land, no torn reads, no double-consumption.

### 4. Backwards-compat round-trip with both blocks (extends Phase B's `health_compat_test.go`)

Adds a row with both `Health.Quarantine` AND `PlannerVerdict.NeedsHuman` to
the existing compat round-trip test. Verifies idempotency on the second
SaveProgress pass.

### 5. Status surface end-to-end (`internal/architectureplanner/status_test.go`, new)

Synthesize planner ledger + autoloop ledger + progress.json with NeedsHuman
rows. Invoke status. Assert outcome buckets, NeedsHuman entries with
reason/since/reshape-count/suggested-action, optional `Keywords:` line.

### Out of test scope

- Live LLM tests against codexu/claudeu (cost + flake).
- systemd path-unit firing (trust systemd).
- Real-process integration test (race-prone scheduling, no value over
  in-process simulation).
- Performance benchmarks (low-volume in production).

### CI

All new tests run under standard `go test ./...`. No new CI configuration.
~3-5 seconds added to existing suite.

## Rollout Notes

This redesign is intentionally additive across layers:

- L1 introduces ledger but adds no behavioral change.
- L2 emits triggers and reacts to them; the 6h timer stays as safety net.
  Existing planner runs have `trigger: "scheduled"`; event-triggered runs
  have `trigger: "event"`. No regression in existing behavior.
- L3 retries on rejection, reducing wasted runs. If `MaxRetries=0`, behavior
  is exactly pre-L3.
- L4 evaluation feeds the prompt observationally; LLM's response is informed
  but not controlled. If the planner ledger is empty (fresh install),
  outcomes is empty and the prompt section is omitted.
- L5 introduces `PlannerVerdict`. Autoloop's `ApplyHealthUpdates` preserves
  it structurally (no code change required there beyond the selection-skip
  rule).
- L6 is purely additive: no keywords means current behavior.

Each layer ships as one commit. The schema (L5) is the only layer with a
migration concern, and `omitempty` keeps the migration trivial: untouched
rows look identical.

The headline metric expected to move: **planner-driven row recovery rate**
(percentage of quarantined rows that a planner reshape unsticks within K
autoloop runs). Today this is unmeasurable (no ledger). After Phase C, L4's
outcomes ARE the measurement.
