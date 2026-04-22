# Gormes Phase 3 Finish Sprint

> Execution: strict TDD, SQLite-first invariants, deterministic exports and ordering.

**Goal:** Finish the full Phase 3 closeout queue (`3.E.1` to `3.E.8`) while preserving shipped 3.A–3.D behavior.

## Verified baseline

From `docs/content/building-gormes/architecture_plan/progress.json`:

- Phase 3 status: **16 complete / 33 total**.
- Remaining (non-complete): **17** items.
- Shipped baseline: 3.A, 3.B, 3.C, 3.D, and 3.D.5 all complete.

## Remaining queue (priority order)

- **P0**
  - `3.E.1` Session Index Mirror (2 items)
  - `3.E.2` Tool Execution Audit Log (3 items)
- **P1**
  - `3.E.4` Extraction State Visibility (2 items)
  - `3.E.6` Memory Decay (2 items)
- **P2**
  - `3.E.3` Transcript Export Command (2 items)
  - `3.E.7` Cross-Chat Synthesis (2 items)
- **P3**
  - `3.E.5` Insights Audit Log (2 items)
- **P4**
  - `3.E.8` Session Lineage + Cross-Source Search (2 items)

---

## Phase 3 Definition of Done

1. Every Phase 3 item in `progress.json` is `complete`.
2. `go test ./internal/memory ./internal/session ./internal/progress ./docs -count=1` passes.
3. `go run ./cmd/progress-gen --validate` passes.
4. `go test ./... -count=1` passes.

---

## Execution order (strict)

1. **P3-A (P0):** `3.E.1` + `3.E.2`
2. **P3-B (P1):** `3.E.4` + `3.E.6`
3. **P3-C (P2):** `3.E.3` + `3.E.7`
4. **P3-D (P3/P4):** `3.E.5` + `3.E.8`
5. **P3-E:** docs and ledger closeout

---

## Slice P3-A — P0 mirrors and audit spine

**Targets**
- `3.E.1` Read-only `sessions.db` -> `index.yaml` mirror
- `3.E.1` deterministic mirror refresh (idempotent, non-mutating)
- `3.E.2` append-only JSONL audit writer + schema
- `3.E.2` kernel/delegate_task audit hooks
- `3.E.2` outcome, duration, and error capture

**Files (expected)**
- `internal/session/*`
- `internal/memory/*`
- `internal/kernel/*`
- `cmd/gormes/*` (status/export wiring)

**Verify**
```bash
go test ./internal/session ./internal/memory ./internal/kernel -count=1 -race
```

**Commit**
`feat(memory): add p0 session mirror and tool-execution audit spine`

---

## Slice P3-B — P1 visibility and decay

**Targets**
- `3.E.4` `gormes memory status`
- `3.E.4` extractor queue + dead-letter visibility
- `3.E.6` relationship `last_seen` tracking
- `3.E.6` deterministic recall-time weight attenuation

**Files**
- `cmd/gormes/*`
- `internal/memory/*`
- `internal/doctor/*`

**Verify**
```bash
go test ./cmd/gormes ./internal/memory ./internal/doctor -count=1
```

**Commit**
`feat(memory): add extraction visibility and decay controls`

---

## Slice P3-C — P2 export and cross-chat synthesis

**Targets**
- `3.E.3` markdown transcript export command
- `3.E.3` render turns/tool-calls/timestamps from SQLite
- `3.E.7` `user_id` concept above `chat_id`
- `3.E.7` cross-chat merge + recall fence

**Files**
- `cmd/gormes/*`
- `internal/memory/*`
- `internal/kernel/*`

**Verify**
```bash
go test ./cmd/gormes ./internal/memory ./internal/kernel -count=1
```

**Commit**
`feat(memory): add transcript export and cross-chat synthesis`

---

## Slice P3-D — P3/P4 insights and lineage search

**Targets**
- `3.E.5` append-only daily `usage.jsonl`
- `3.E.5` session/token/cost rollups
- `3.E.8` `parent_session_id` lineage for compression splits
- `3.E.8` source-filtered FTS/session search across chats

**Files**
- `internal/memory/*`
- `internal/session/*`
- `internal/progress/*` (if ordering/reporting hooks required)
- `cmd/gormes/*`

**Verify**
```bash
go test ./internal/memory ./internal/session ./internal/progress ./cmd/gormes -count=1
```

**Commit**
`feat(memory): add insights audit and lineage-aware cross-source search`

---

## Slice P3-E — docs and ledger closeout

**Files**
- `docs/content/building-gormes/architecture_plan/progress.json`
- `docs/content/building-gormes/architecture_plan/phase-3-memory.md`
- `docs/content/building-gormes/architecture_plan/subsystem-inventory.md`
- `docs/content/building-gormes/core-systems/memory.md`
- `docs/content/building-gormes/architecture_plan/_index.md`

**Verify**
```bash
go run ./cmd/progress-gen --write
go run ./cmd/progress-gen --validate
go test ./docs -count=1
go test ./... -count=1
```

**Commit**
`docs(phase3): finalize memory closeout queue and shipped narrative`

---

## Global guardrail

After every slice:

```bash
go test ./... -count=1
```

If red, land immediate fix commit:

`fix(regression): phase3 <short-description>`

No new feature commits until green.