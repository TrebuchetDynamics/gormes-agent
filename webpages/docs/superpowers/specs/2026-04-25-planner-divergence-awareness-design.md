# Planner Divergence Awareness Design

**Status:** Draft
**Author:** Codex (Claude Opus 4.7 1M)
**Date:** 2026-04-25

## Context

Phase C closed the autoloop ↔ planner feedback loop with reactive triggers,
self-evaluation, and human escalation. But the planner still treats upstream
sources (Hermes, GBrain, Honcho) as eternally authoritative: every run pulls
upstream, inventories the tree, and refines contracts assuming "do what they
do." This works while porting. It fails the moment Gormes ships something
upstream doesn't have — the autoloop itself, `internal/plannertriggers`, the
`Health`/`PlannerVerdict` schema, the architecture-planner-loop. The planner
has no way to know "stop looking upstream for this surface; we own it."

As Gormes diverges from Hermes (Phase 6+ in particular), the planner needs:
1. **Per-row provenance** — does this row's contract come from upstream, or
   is it Gormes-original?
2. **Surface-level drift state** — for each subphase, are we still porting,
   converged at parity, or now leading?
3. **Reverse research** — when impl ships something new, the planner should
   notice and propose contracts for it (not search upstream for an analog).
4. **Reactive trigger when impl moves** — file watcher on `cmd/` and
   `internal/` so a recent landing wakes the planner.
5. **Drift-aware prompt clauses** — tell the LLM how to set provenance and
   when to promote subphases between drift states.

## Goals

1. Let the planner declare "this row is Gormes-original" instead of
   fabricating upstream pointers.
2. Mark subphases as `owned` so the planner skips deep upstream research for
   surfaces Gormes leads on.
3. Read the impl tree directly (`cmd/+internal/`) and surface its inventory
   to the LLM.
4. React when impl moves materially (path unit on `cmd/+internal/`,
   rate-limited 30 min).
5. Extend the status surface to show drift state by subphase.
6. Keep symmetric ownership: planner owns Provenance + DriftState; autoloop
   preserves them structurally via Phase B's typed round-trip.

## Non-goals

- Auto-discovery of "owned" status. Humans/planner mark explicitly via the
  prompt; deny-list of Gormes-original paths covers the common case.
- AST-level reverse search of the impl tree. Path-prefix match is the
  entirety of inventory logic.
- Cross-repo provenance linking beyond a simple `UpstreamRef` string field.
- Auto-demotion (owned → porting if upstream resurges). One-way ratchet.
- Live LLM tests. All planner tests use mock runners.

## Decision Summary

The accepted design is **Planner Divergence Awareness** — five layers added
to `internal/architectureplanner/` and `internal/progress/`. No autoloop
package changes (autoloop just preserves the new typed blocks for free).

```
┌──────────────────────────────────────────────────────────────────────────┐
│  progress.json                                                           │
│  - Item.Provenance       (planner owns; autoloop preserves) ← Phase D    │
│  - Subphase.DriftState   (planner owns; autoloop preserves) ← Phase D    │
│  - Item.PlannerVerdict   (planner owns; autoloop preserves) ← Phase C    │
│  - Item.Health           (autoloop owns; planner preserves) ← Phase B    │
└─────────────────────────────────┬──────────────────────────────────────────
                                  │ planner reads/writes all of the above
                                  ▼
   ┌──────────────────────────────────────────────────────────────┐
   │ architecture-planner-loop                                    │
   │  - L2 ScanImplementation walks cmd/+internal/                │
   │  - L3 BuildPrompt adds PROVENANCE + DRIFT STATE clauses      │
   │  - L4 PathUnit on impl tree fires planner on impl change     │
   │  - L5 RenderStatus shows drift state per subphase            │
   └────────────────────┬─────────────────────────────────────────┘
                        │ rate-limited (30 min) systemd path unit
                        ▼
              ┌────────────────────────────────┐
              │  cmd/                          │
              │  internal/                     │
              │  (impl tree — watched paths)   │
              └────────────────────────────────┘
```

Five layers, five commits:

| Layer | What it adds | Touches | Commit |
|---|---|---|---|
| **L1 — Schema** | Provenance struct + Item.Provenance; DriftState struct + Subphase.DriftState; round-trip tests | `internal/progress/health.go`, `progress.go`, tests | C1 |
| **L2 — Reverse research** | ScanImplementation + ContextBundle.ImplInventory + Config.GormesOriginalPaths | new `implscan.go`, `context.go`, `config.go`, `run.go`, tests | C2 |
| **L3 — Drift-aware prompt** | PROVENANCE AWARENESS + DRIFT STATE soft clauses; ImplInventory section in prompt | `prompt.go`, tests | C3 |
| **L4 — Reactive impl trigger** | systemd path unit on cmd/+internal/; PLANNER_TRIGGER_REASON env threading; LedgerEvent.Trigger="impl_change" support | `service.go`, env wiring, tests | C4 |
| **L5 — Drift status + forensics** | Status surface drift bucketing; LedgerEvent.DriftPromotions; lifecycle test | `status.go`, `ledger.go`, new `divergence_lifecycle_test.go` | C5 |

L1 is foundational. L2/L3 stack on L1. L4 is independent. L5 touches L1+L3.

## Schema Additions

### `Provenance` on `Item`

```go
// internal/progress/health.go (alongside RowHealth and PlannerVerdict)

// Provenance is per-row source-of-truth metadata. OWNED by the planner;
// autoloop preserves it via typed-struct round-trip (Phase B). The planner
// sets origin_type="gormes" for rows that have no upstream analog (e.g.
// rows in the L5 owned subphases like 5.O autoloop control plane).
type Provenance struct {
    OriginType  string `json:"origin_type"`            // "upstream" | "gormes" | "hybrid"
    UpstreamRef string `json:"upstream_ref,omitempty"` // e.g. "hermes:gateway/platforms/api_server.py@abc123"
    OwnedSince  string `json:"owned_since,omitempty"`  // RFC3339 when origin_type became "gormes"
    Note        string `json:"note,omitempty"`
}
```

`Item.Provenance *Provenance` — added as the LAST field (after Phase C's
`PlannerVerdict`). Field-order discipline preserved.

### `DriftState` on `Subphase`

```go
// Subphase-level convergence state.
type DriftState struct {
    Status            string `json:"status"`                          // "porting" | "converged" | "owned"
    LastUpstreamCheck string `json:"last_upstream_check,omitempty"`   // RFC3339
    OriginDecision    string `json:"origin_decision,omitempty"`       // "Gormes invented L2 trigger ledger; no upstream analog"
}
```

`Subphase.DriftState *DriftState` — added as the LAST field of Subphase.

### Schema invariants

1. **Symmetric ownership extended.** Autoloop owns Health. Planner owns
   PlannerVerdict, Provenance, DriftState.
2. **`OwnedSince` is monotonic.** Set when origin_type first becomes "gormes".
3. **`DriftState.Status` is one-way ratchet.** porting → converged → owned.
   Never auto-demotes. Humans can demote by editing progress.json directly.
4. **Default behavior backward-compatible.** A row without Provenance is
   treated as "upstream" by default (matches pre-Phase-D behavior).
5. **All four typed blocks** (Health, PlannerVerdict, Provenance,
   DriftState) survive structural round-trip via Phase B's typed marshaling.

## L2 — Reverse Research (`implscan.go`)

```go
// internal/architectureplanner/implscan.go (new)

type ImplInventory struct {
    GormesOriginalPaths []string `json:"gormes_original_paths,omitempty"`
    RecentlyChanged     []string `json:"recently_changed,omitempty"`
    OwnedSubphases      []string `json:"owned_subphases,omitempty"`
}

// ScanImplementation walks repoRoot/cmd and repoRoot/internal, applies the
// Gormes-original deny-list to identify paths with no upstream analog, and
// reports paths modified within `lookback` of `now`. Used by L3 prompt to
// give the LLM a concrete "what's here that you don't need to research
// upstream for" inventory.
func ScanImplementation(repoRoot string, gormesOriginalPaths []string, lookback time.Duration, now time.Time) (ImplInventory, error)
```

**Gormes-original path matching** = strict prefix match. Default deny-list
(env-overridable via `PLANNER_GORMES_ORIGINAL_PATHS`):

```
cmd/autoloop/
cmd/architecture-planner-loop/
internal/autoloop/
internal/architectureplanner/
internal/plannertriggers/
internal/progress/health.go
www.gormes.ai/internal/site/installers/
```

(Any file under those is Gormes-original; the LLM is told not to look upstream.)

**Recently changed** = `os.Stat` mtime within `[now-lookback, now]`. Default
lookback = 24h.

**OwnedSubphases** computed from progress.json: a subphase whose every row's
`write_scope` is entirely under `GormesOriginalPaths`. The planner can then
auto-suggest promoting these subphases to `DriftState.Status="owned"`.

`Config.GormesOriginalPaths []string` (env `PLANNER_GORMES_ORIGINAL_PATHS`,
default seeded from the list above).

`ContextBundle.ImplInventory ImplInventory` field added.

## L3 — Drift-Aware Prompt Clauses

Two new always-on soft clauses in `prompt.go`:

```text
PROVENANCE AWARENESS (SOFT RULE)

Every progress.json row SHOULD carry a `provenance` block declaring its
origin_type ("upstream", "gormes", or "hybrid"). When you create or refine
a row in a Gormes-owned area (see Implementation Inventory section below),
set provenance.origin_type="gormes" and origin_decision describing why.
Do NOT fabricate upstream_ref pointers when none exist — leave the field
empty and rely on the origin_decision to explain.

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
```

Plus a new context section rendered when ImplInventory is non-empty:

```text
## Implementation Inventory

Gormes-original surfaces (no upstream research needed):
- internal/autoloop/
- internal/architectureplanner/
- internal/plannertriggers/
- ...

Recently changed (last 24h):
- internal/autoloop/run.go
- internal/architectureplanner/verdict.go
- ...

Subphases that ARE entirely Gormes-original (candidates for "owned"):
- 5.O autoloop control plane
- 5.P installer parity
- 5.Q architecture-planner-loop
```

## L4 — Reactive Impl Trigger

New systemd path unit `gormes-architecture-planner-impl.path`:

```ini
[Unit]
Description=Trigger Gormes architecture planner on impl tree change

[Path]
PathChanged=%h/.../cmd
PathChanged=%h/.../internal
TriggerLimitIntervalSec=1800   # 30 min — bulk dev work must not flood planner
TriggerLimitBurst=1
Unit=gormes-architecture-planner.service

[Install]
WantedBy=default.target
```

The wrapper script `scripts/architecture-planner-loop.sh` checks
`$PLANNER_TRIGGER_REASON` (set by the path unit's ExecStart override) and
threads it into env. Config picks up the value into `Config.TriggerReason`.

`Trigger` field on `LedgerEvent` gains the new value `"impl_change"`. RunOnce
sets `trigger = "impl_change"` when `cfg.TriggerReason == "impl_change"` AND
no plannertriggers events are queued (impl_change has lower priority than
quarantine events, so if both are present, "event" wins).

`InstallPlannerService` extended to write the third `.path` file alongside
the existing two.

## L5 — Drift Status + Forensics

### Status surface extension

`architecture-planner-loop status` adds:

```
Drift state by subphase:
  PORTING (12): 2.B, 2.C, 3.A, 3.B, 4.A, ...
  CONVERGED (5): 1.A, 1.B, 2.A, 3.D, 4.E
  OWNED (4): 5.O, 5.P, 5.Q, 5.R

Recent drift promotions (last 7d):
  - 5.O autoloop control plane: porting → owned (2026-04-24, planner run R3)
  - 5.P installer parity: porting → converged (2026-04-23, planner run R2)
```

### Ledger forensics

```go
type DriftPromotion struct {
    SubphaseID string `json:"subphase_id"`
    From       string `json:"from"`     // "porting" | "converged"
    To         string `json:"to"`       // "converged" | "owned"
    Reason     string `json:"reason,omitempty"`
}

type LedgerEvent struct {
    // ... existing fields ...
    DriftPromotions []DriftPromotion `json:"drift_promotions,omitempty"`
}
```

The planner runtime detects drift state changes between before/after docs
(diffSubphaseStates helper) and emits them as DriftPromotions in the
LedgerEvent. Status surface reads these to show "Recent drift promotions"
across the 7-day window.

### Lifecycle test

`internal/architectureplanner/divergence_lifecycle_test.go` walks one
subphase through:

```
Run 1: subphase has no drift_state; planner sets status="porting"
Run 2: planner adds 3 rows with provenance.origin_type="upstream"
Run 3: rows ship in autoloop; planner promotes status to "converged"
Run 4: Gormes ships an extension with no upstream analog;
       planner adds row with provenance.origin_type="gormes",
       origin_decision="extends converged surface with Gormes-original
       feature", and promotes subphase to "owned"
Run 5: subphase now "owned"; ImplInventory.OwnedSubphases includes it;
       planner stops reading upstream for this surface
```

Assertions cover: provenance fields populated correctly; drift promotions
recorded in ledger; subphase one-way-ratchets through states; ScanImplementation
correctly identifies owned subphases.

## Testing

Per-layer tests are enumerated in each layer's section above. Five
cross-cutting test surfaces:

1. **Symmetric preservation extended** — all FOUR typed blocks (Health,
   PlannerVerdict, Provenance, DriftState) survive both autoloop's writes
   and planner's writes byte-equal. Extends Phase C's `preservation_test.go`.

2. **ScanImplementation deterministic** — same repo state → same inventory.
   File mtime sensitivity tested via `t.Setenv` and synthetic `os.Chtimes`.

3. **Drift state ratchet** — subphase cannot move from owned → converged or
   converged → porting via planner-side code. Only direct progress.json
   edit (simulated by test) can do that.

4. **Path unit rate limit** — render unit content; assert
   TriggerLimitIntervalSec=1800 and TriggerLimitBurst=1.

5. **End-to-end divergence lifecycle** — covered by L5's
   `divergence_lifecycle_test.go`.

### Out of test scope

- Live LLM tests against backends.
- Real systemd path-unit firing (trust systemd; test unit content).
- Cross-version provenance migration (when typed schema gains/loses fields,
  preservation handles it via typed round-trip).

## Rollout Notes

This is intentionally additive across layers:

- L1 introduces schema but no behavioral change. Rows without Provenance
  default to "upstream" semantics in L3's prompt.
- L2 surfaces ImplInventory in context but doesn't change autoloop behavior.
- L3 prompt clauses tell the LLM how to use the new fields; LLM may take
  multiple runs to populate them.
- L4 path unit only fires after a real install; existing installs add it
  via `service install --force`.
- L5 status surface gracefully renders empty buckets; ledger forensics
  fields are `omitempty`.

Each layer ships as one commit. The schema (L1) is the only layer with a
back-compat concern, and `omitempty` keeps it trivial: untouched rows look
identical.

The headline change: **planner stops fabricating upstream pointers for
Gormes-original code**, and operators can see at a glance which subphases
are still chasing upstream vs which have moved past it.
