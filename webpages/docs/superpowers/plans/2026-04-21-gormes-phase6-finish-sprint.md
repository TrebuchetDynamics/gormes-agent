# Gormes Phase 6 Finish Sprint

> Execution: strict TDD, observable metrics for every learning-loop decision.

**Goal:** Ship the full native learning loop (complexity detection, extraction, storage, retrieval, feedback, and user-facing skill browsing).

## Verified baseline
- Phase 6 status: **0/6 complete**.
- Remaining: all Phase 6 items (`6.A`..`6.F`).

## Definition of Done
1. All Phase 6 items marked `complete`.
2. Skills are extracted, scored, retrievable, and browsable from TUI + Telegram.
3. Learning loop decisions are auditable and reversible.
4. `go test ./... -count=1` and docs tests pass.

## Slice P6-A — Complexity detector + extractor
Targets:
- `6.A` heuristic/LLM complexity detector
- `6.B` LLM-assisted skill extraction

Files:
- `internal/skills/*`
- `internal/kernel/*`

Verify:
```bash
go test ./internal/skills ./internal/kernel -count=1
```

Commit:
`feat(learning): add complexity detector and extractor pipeline`

## Slice P6-B — Skill storage + retrieval matching
Targets:
- `6.C` portable SKILL.md storage format
- `6.D` hybrid lexical + semantic retrieval

Files:
- `internal/skills/*`
- `internal/memory/*` (semantic hooks)

Verify:
```bash
go test ./internal/skills ./internal/memory -count=1
```

Commit:
`feat(learning): add portable storage and hybrid retrieval`

## Slice P6-C — Feedback and operator surface
Targets:
- `6.E` skill effectiveness scoring
- `6.F` TUI + Telegram browsing

Files:
- `internal/skills/*`
- `internal/channels/telegram/*`
- `internal/tui/*`
- `cmd/gormes/*`

Verify:
```bash
go test ./internal/skills ./internal/tui ./internal/channels/telegram ./cmd/gormes -count=1
```

Commit:
`feat(learning): add scoring loop and skill browsing surfaces`

## Slice P6-D — Docs and ledger closeout
Files:
- `docs/content/building-gormes/architecture_plan/progress.json`
- `docs/content/building-gormes/architecture_plan/_index.md`

Verify:
```bash
go test ./docs -count=1
go test ./... -count=1
```

Commit:
`docs(phase6): finalize learning-loop phase closeout`
