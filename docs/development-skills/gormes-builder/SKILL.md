---
name: gormes-builder
description: Build Gormes from progress.json rows toward full Hermes-in-Go parity. Use when an agent must select or execute a builder-ready row, implement Go runtime/docs/web changes, run TDD and e2e verification, build Goncho inside Gormes, or continue delivery from docs/content/building-gormes/architecture_plan/progress.json without inventing a separate backlog.
---

# Gormes Builder

## Mission

Build Gormes until it is Hermes in Go, with Goncho as the in-repo Honcho-compatible Go port. No smaller "MVP" is the final goal. Each builder pass still works as one bounded, test-proven slice.

This skill replaces the deleted autonomous builder-loop command: read
`progress.json`, pick one ready row, implement it with tests, and report
evidence.

If the task might be planning, parity audit, interface design, or skill creation instead of implementation, route through `gormes-skill-manager` first.

## Source Order

1. Read `AGENTS.md`.
2. Read the selected row in `docs/content/building-gormes/architecture_plan/progress.json`.
3. Read the relevant section of `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`.
4. Read generated handoff docs under `docs/content/building-gormes/builder-loop/` when useful.
5. Read exact `source_refs`, upstream Hermes/Honcho/GBrain files, and current Gormes code before editing.
6. Read `references/delivery-gates.md` before final verification.

## Row Selection

If the user names a row, use that row. Otherwise choose the highest-value row that is actually builder-ready:

- status is selectable by the skill rules in this file and the row contract.
- row has `write_scope`.
- row has `test_commands` or explicit `no_test_required`.
- `ready_when` is satisfied.
- `not_ready_when` is false.
- dependencies in `blocked_by` are shipped or irrelevant to this slice.

Prefer P0/P1 closure rows in Phase 4.I and Phase 3.G when they are ready,
because they prove the Hermes-in-Go turn and Goncho/Honcho compatibility. Do
not bypass an explicit user-named row or a real dependency blocker.

Do not implement broad umbrella rows. If the best row is too vague, switch to planning: refine/split the row instead of guessing.

Useful discovery:

```sh
go run ./cmd/progress validate
jq -r '.phases | to_entries[] | .key as $phase | .value.subphases | to_entries[] | .key as $subphase | .value.items[]? | select(.status=="planned" or .status=="porting" or .status=="fixture_ready") | [$phase,$subphase,.name,.priority] | @tsv' docs/content/building-gormes/architecture_plan/progress.json | head -20
```

## Build Workflow

### 1. Understand The Contract

Summarize the selected row in your own words:

- behavior to ship
- feature-map area or upstream concept;
- target Go package and public interface;
- exact write scope
- upstream behavior being ported
- expected tests
- done signal

If any of these are missing, stop and refine the row.

Start with a zoomed-out map of the relevant public interface and callers. Do not edit until you know which behavior the user, CLI, API, channel, tool, memory, or provider boundary must expose.

Before editing, build a short builder packet:

- selected row and status;
- feature-map section;
- upstream refs;
- Go target package/interface;
- first failing test or no-test rationale;
- degraded mode;
- done signal.

### 2. Write Or Confirm A Failing Test

Use TDD unless the row is explicitly `no_test_required`.

- Prefer focused package tests first.
- Add fixture data instead of live provider/network dependencies.
- For Goncho/Honcho work, include compatibility expectations for public `honcho_*` names when relevant.
- For gateway/provider work, include degraded-mode and error classification tests.
- For CLI/TUI/API work, include command/server behavior tests or e2e where practical.

Run the focused test and capture the failure before implementing when feasible.

Use tracer-bullet TDD. Write one behavior test, make it pass, then repeat. Do not write all imagined tests first and then all implementation. Good tests verify observable behavior through public interfaces and survive internal refactors.

### 3. Implement In Scope

Edit only files allowed by the row's `write_scope` unless the row is clearly wrong; if wrong, fix the row first or report the mismatch.

Follow existing package patterns. Prefer typed Go interfaces, hermetic fixtures, and local stores over ad hoc scripts or network calls.

Favor deep modules: small caller-facing interfaces with substantial implementation hidden behind them. Avoid shallow pass-through abstractions unless two real adapters or callers prove the seam is useful.

### 4. Verify The Row

Run every row-local `test_commands` entry. Then run focused package tests for touched packages. Broaden verification when touching shared behavior.

Required minimum for runtime rows:

```sh
go test ./... -count=1
go run ./cmd/progress validate
```

Use `references/delivery-gates.md` to decide when docs, web, Playwright, or orchestrator suites are required.

If implementation changes a feature-map claim, update the row or send the work
back through `gormes-planner` before reporting done.

### 5. Update Surfaces

If implementation changes public behavior, update the matching docs, generated data, or website content. If the row is fully proven, update only the relevant progress evidence/status fields according to repo patterns; do not reshape unrelated roadmap rows.

Never create a parallel backlog. Missing follow-up work goes into `progress.json`.

For complex rows, use `gormes-tdd-slice` for the red-green loop. For uncertain API/package shape, use `gormes-interface-designer` before coding. For missing upstream coverage, send the work back through `gormes-parity-auditor` or `gormes-planner`.

## Goncho Builder Rules

- Internal package and product name: Goncho.
- External compatibility names: preserve Honcho-compatible `honcho_*` names when public tools or clients require them.
- Prefer local SQLite/FTS/graph-backed behavior already present in Gormes.
- Build compatibility fixtures that prove Honcho behavior without requiring live Honcho.
- Do not call Goncho done until sessions, messages, memory CRUD/search, provenance, and compatibility tool contracts are covered by tests.

## Failure Handling

If tests fail from your own changes, fix them before reporting done. If the selected row is unbuildable because the contract is wrong, update/refine the row or report the exact blocker.

If the worktree is dirty before you start, preserve existing changes. Do not revert user or previous-agent work. Work with the current tree and mention any verification limits caused by unrelated changes.

## Final Report

Report:

1. Row selected
2. Behavior shipped
3. Files changed
4. Tests/e2e run
5. Feature-map/progress consistency
6. Progress/docs updates
7. Remaining blockers or next row
