# Kit `tasks/` becomes the canonical work queue

Gormes replaces its `progress.json`-backed work selection with the kit task store under `tasks/`. The owner chose this migration on 2026-11-03 so a kit `docs-sync` pass can maintain the roadmap and progress with one task store. `progress.json` stops being the queue and becomes archived history after the cutover.

## Context

`progress.json` is a single logical backlog with 1,231 rows. `internal/planning/progress` and `cmd/progress` read and write it, and they regenerate the README rollup, the executive roadmap, the completion ledger, and the builder-loop pages.

Kit `docs-sync` expects a different owner. Its task store lives in `tasks/goals/<ID>.md`, `tasks/items/<ID>.md`, and `tasks/evidence/<ID>.toml`. Selection, dispatch, integration, and closure read that store through `autowork tasks`. A repository without a `tasks/` tree reports `state: none`, so `docs-sync` cannot reconcile the roadmap with accepted work.

`autowork tasks prepare --check .` reports `state: none`, 1,039 documents, and 5 accepted headings. It does not see `progress.json`, because that path is a directory of module JSON files.

## Decision

1. `tasks/` is the canonical queue for open work. Goals map to roadmap phases. Items map to open progress rows.
2. Complete rows stay in the roadmap and parity documents as history. They do not become done task records, because the queue tracks open work, not the completed trajectory.
3. The cutover keeps the generated roadmap and progress pages readable. Each page states that `tasks/` is the live queue and that `progress.json` is archived history.
4. `AGENTS.md`, the planner and builder skills, and the progress tooling move to the task store in the cutover slice. No parallel queue remains after that slice.
5. `docs-sync` owns document roles and goal-gap analysis after the cutover. `autowork tasks` owns task records. Gormes keeps its parity evidence documents.
6. The legacy `TODO.md` moves to `TODO.archive.md`. Its only planned item, Navivox interactive approval routing, is superseded by the Navivox retirement decision and does not become a task.

## Mapping

| Task store field | Source |
|---|---|
| Goal | Roadmap phase, sourced from `phase-2-gateway.md`, `phase-4-brain-transplant.md`, `phase-5-final-purge.md`, or `completion-plan.md#current-finish-ledger` |
| `title` | Progress row `name` |
| `scope` | Progress row `contract` |
| `acceptance` | Progress row `acceptance` |
| `checks` | Progress row `test_commands` |
| `touches` | Progress row `write_scope` |
| `exclusions` | Progress row `not_ready_when` |
| `depends_on` | Open-row `blocked_by` entries that also migrate. Completed blockers are satisfied and need no dependency. |
| `section` | `Now` for the P0 row, `Next` for ready P1 to P3 rows, `Needs decision` for drafts, umbrellas, deferred UI work, and externally gated rows |
| `ownership` | `worker` for fixture-ready implementation rows, `coordinator` for plan rows, `owner` for rows that need an owner decision |
| `live` | `denied` for every migrated row |

## Consequences

- The migration must not lose history. The roadmap and parity pages keep every completed row, and `progress.json` is archived before deletion.
- The cutover is one slice. Until then, the repository has two sources, and that state is transitional, not a second queue.
- `autowork` becomes a repository tool dependency for selection. The Go test suite must not require it.
- `docs-sync` can now run goal-gap analysis against the queue. Its diagnostics become the maintenance signal for the roadmap.

## Status

Accepted. The decision is the owner's. The first slice creates the task store. The cutover slice retires `progress.json` and updates the repository instructions and skills.
