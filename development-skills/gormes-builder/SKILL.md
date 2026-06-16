---
name: gormes-builder
description: Implement one builder-ready Gormes progress row with tests. Use for Go runtime/docs/web changes, Goncho/Hermes parity slices, TDD/e2e verification, or delivery continuation.
---

# Gormes Builder

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Build Gormes until it is Hermes in Go, with Goncho as the in-repo Honcho-compatible Go port. No smaller "MVP" is the final goal. Each builder pass still works as one bounded, test-proven slice.

This skill runs bounded builder passes: read one builder-ready logical progress row, confirm the source-backed parity or owned-Gormes contract, implement it with tests, and report evidence. If Juan explicitly asks for a builder-loop style subsystem, plan it as a first-class Gormes feature with clear interfaces, parity coverage, validation gates, and operator controls.

If the task might be planning, parity audit, interface design, or skill creation instead of implementation, route through `gormes-skill-manager` first.

Hermes Agent is the Python upstream/father implementation for Gormes. Prefer
`./hermes-agent` as the local behavior reference and fall back to
`../hermes-agent` only when absent. Resolve it as `$HERMES_SRC` before trusting
source refs. Do not depend on the Python runtime from Gormes code.

## Source Order

1. Read `AGENTS.md`.
2. Read the selected logical progress row through `cmd/progress` / `internal/planning/progress`. Use parity evidence docs only as source-backed classification evidence, not as the executable backlog.
3. Read the relevant section of `webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md` for subsystem context.
4. Read exact upstream Hermes/Honcho source refs named by the row or evidence doc and current Gormes target files before editing.
5. Read `references/delivery-gates.md` before final verification.

For Hermes UX parity rows, confirm the active upstream contract before
writing tests. The current full-screen terminal UI comes from
`$HERMES_SRC/ui-tui`; the older prompt-toolkit CLI in
`$HERMES_SRC/cli.py` is still useful for classic command
behavior but is not authoritative for the modern TUI chrome. Runtime,
installer, `go run`, `bin/gormes`, installed-binary, PATH, and
`sessions.db`-lock behavior belongs under `gormes-dev-runtime` before this
skill implements a row.

For persona/default-template rows, first check whether the existing
`Gormes agent template reset command` row and `internal/agenttemplate` already
cover the behavior. Extend that proven surface with focused tests instead of
creating a second reset/default-template implementation.

For tool-loop rows, first check the completed `Kernel tool loop` progress row,
`internal/kernel/kernel.go`, and `internal/kernel/tools_test.go`. If those
fixtures pass, a live report like raw `tool iteration limit exceeded (10)` is
likely stale binary/runtime validation or channel rendering, not a new kernel
loop implementation.

## Row Selection

If the user names a specific behavior, find the matching logical progress row or route to `gormes-planner` to create/refine one. Otherwise choose the highest-value builder-ready row that is actually buildable.

Before reporting "builder blocked", always refresh the control plane in the live checkout:

```sh
go run ./cmd/progress validate
go run ./cmd/progress next-work
go run ./cmd/progress next-work --repo-only
```

Do not rely on a pasted or earlier `next-work` result after planner/progress files have changed. If `next-work --repo-only` returns `decision=build`, treat that row as the selected builder row even when its `contract_status` is `draft`; the progress selector is authoritative. If `next-work --repo-only` returns `decision=plan`, stop runtime work and emit a planner task packet naming the blocked/vague rows or parity atoms that need sharpening.

Before selecting implementation work, run a stale-classification preflight for the candidate cluster. This prevents duplicate builds when Gormes already implements behavior but the evidence doc still says `missing`:

- Search the likely Go packages and tests named by the row/evidence entry, subsystem, command, tool name, gateway channel, or upstream symbol.
- If repository code plus focused tests already prove the behavior, do not implement it again. Update the progress row or parity evidence from `missing` to `covered`/`partial` with exact Go files and validation, or route to `gormes-planner` when the correction touches broader taxonomy.
- If implementation is present but incomplete, keep the row/atom `partial`, narrow the remaining gap, and build only that remaining gap.
- If the behavior is truly absent and the row is builder-ready, proceed with TDD implementation.

A buildable row has these properties:

- it is missing or partial and not an umbrella;
- upstream source refs exist and are readable when parity is claimed;
- the Gormes target package exists or the row names the new package;
- `write_scope`, `test_commands`, acceptance, degraded mode, and done signal are concrete;
- it is not blocked by a missing prerequisite.

If `cmd/progress next-work --repo-only` selects a row that appears to violate one of these checks, do not silently skip it or report generic blockage. Name the exact missing field/source/write-scope issue and route that precise repair to `gormes-planner`. If the issue is only stale parity evidence while code already exists, perform the stale-classification preflight and update evidence or route to planner instead of duplicating runtime code.

Prefer rows that unblock the widest cluster of other missing work. Do not implement broad umbrella rows that cover multiple subsystems; route them to `gormes-planner` for splitting first.

For repeated "actually implemented features" sweeps, builder should not be the first skill. Route to `gormes-hermes-parity` + `gormes-planner` to reconcile stale classifications, then return here only when a corrected missing/partial progress row has a concrete remaining runtime gap.

Useful discovery:

```sh
go run ./cmd/progress next-work
go run ./cmd/progress list --module <module>
rg -n "missing|partial|provider|<topic>" webpages/docs/content/building-gormes webpages/docs/parity-evidence
```

## Build Workflow

### 1. Understand The Row

Summarize the selected row in your own words:

- behavior to ship
- feature-map area or upstream concept;
- target Go package and public interface;
- exact write scope
- upstream behavior being ported (from row source refs or evidence docs)
- active upstream file+line and why it is the right contract
- expected tests
- done signal

If any of these are missing, stop and refine the progress row or route to `gormes-planner` before implementing.

Start with a zoomed-out map of the relevant public interface and callers. Do not edit until you know which behavior the user, CLI, API, channel, tool, memory, or provider boundary must expose.

Before editing, build a short builder packet:

- selected row/behavior name and status;
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
- For visible Hermes UX bugs, test the artifact the user actually sees:
  stale "Hermes" labels, duplicate replies, hourglass/status messages,
  hidden or leaked tool-call noise, debug chrome, old prompt boxes/rules, and
  session-state fallback copy are all first-class assertions.
- For installer/runtime rows, test the source/binary/install matrix when the
  bug can differ by surface. Use isolated `GORMES_HOME` temp dirs for smoke
  tests and assert Gormes output never tells users to start Hermes services.
- For channel-visible tool-loop rows, include a fixture that proves the raw
  budget error, leaked tool-call XML/text, and duplicate final send are absent.

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

If a row's `source_refs` point at stale upstream files for the behavior being
implemented, do not silently build against them. Patch the row or hand it to
`gormes-planner` first, then implement against the corrected contract.

### 5. Update Surfaces

If implementation changes public behavior, update the matching docs, generated data, or website content. If the behavior is fully implemented, update the progress row and any parity evidence status from missing/partial to covered with exact files and validation.

Never create a parallel backlog. Missing follow-up work goes into logical progress rows; parity evidence docs may carry source-backed classification details.

For complex rows, use `gormes-tdd-slice` for the red-green loop. For uncertain API/package shape, use `gormes-interface-designer` before coding. For missing upstream coverage, send the work back through `gormes-parity-auditor` or `gormes-planner`.

## Goncho Builder Rules

- Internal package and product name: Goncho.
- External compatibility names: preserve Honcho-compatible `honcho_*` names when public tools or clients require them.
- Prefer local SQLite/FTS/graph-backed behavior already present in Gormes.
- Build compatibility fixtures that prove Honcho behavior without requiring live Honcho.
- Do not call Goncho done until sessions, messages, memory CRUD/search, provenance, and compatibility tool contracts are covered by tests.

## Failure Handling

If tests fail from your own changes, fix them before reporting done. If the selected row is unbuildable because the parity classification is wrong, update the progress/evidence surface or report the exact blocker.

A valid blocked report must include the fresh `next-work --repo-only` output, the exact selected row or `decision=plan` reason, and the smallest planner repair packet. Do not say "no builder-ready rows" when the fresh command prints `decision=build`.

If the working tree is dirty before you start, preserve existing changes. Do not revert user or previous-agent work. Work with the current `development` checkout and mention any verification limits caused by unrelated changes.

Do not push from this skill unless the user explicitly routed through `gormes-git`. If upstream advanced, fetch and inspect before any integration action; preserve your code/tests and route commit/push recovery to `gormes-git`.

## Final Report

Report:

1. Progress row / behavior and status before/after
2. Behavior shipped
3. Files changed
4. Tests/e2e run
5. Feature-map/progress/evidence consistency
6. Progress or parity evidence updates
7. Remaining blockers or next row to build
8. Hermes source refs used to verify behavior
