---
name: gormes-builder
description: Use when an agent must select a missing or partial Hermes behavior atom from the parity evidence inventory, implement Go runtime/docs/web changes, run TDD and e2e verification, build Goncho inside Gormes, or continue Gormes delivery.
---

# Gormes Builder

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Build Gormes until it is Hermes in Go, with Goncho as the in-repo Honcho-compatible Go port. No smaller "MVP" is the final goal. Each builder pass still works as one bounded, test-proven slice.

This skill runs bounded builder passes: read the parity evidence atom inventory at `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, pick one `missing` or `partial` behavior atom, implement it with tests, and report evidence. If Juan explicitly asks for a builder-loop style subsystem, plan it as a first-class Gormes feature with clear interfaces, parity coverage, validation gates, and operator controls.

If the task might be planning, parity audit, interface design, or skill creation instead of implementation, route through `gormes-skill-manager` first.

Hermes Agent is the Python upstream/father implementation for Gormes. Prefer
`./hermes-agent` as the local behavior reference and fall back to
`../hermes-agent` only when absent. Resolve it as `$HERMES_SRC` before trusting
source refs. Do not depend on the Python runtime from Gormes code.

## Source Order

1. Read `AGENTS.md`.
2. Read the selected atom in `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md` — the evidence-first classification of every Hermes behavior atom. Every atom has an upstream Hermes file+line ref, a Gormes file+line ref or explicit `missing`, and a status (covered/partial/missing/owned). Use `grep` to find atoms by subsystem, status, or name.
3. Read the relevant section of `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md` for subsystem context.
4. Read exact upstream Hermes/Honcho files named in the atom's `HERMES` column and current Gormes code in the `GORMES` column before editing.
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

## Atom Selection

If the user names a specific behavior, find its atom in the parity evidence doc. Otherwise choose the highest-value atom that is actually buildable:

- status is `missing` or `partial` in the parity evidence doc.
- the upstream source file named in the `HERMES` column exists and is readable.
- the Gormes target package named in the `GORMES` column exists (or is a new file in an existing package).
- the atom is not blocked by a prerequisite that is also `missing`.

Prefer atoms that unblock the widest cluster of other missing atoms. Consult the `Notes` column in the evidence doc for dependency hints.

Do not implement broad umbrella atoms that cover multiple subsystems. If the best atom is too large, split it into smaller atoms and update the parity evidence doc first.

Useful discovery:

```sh
grep "| missing |" docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md | wc -l
grep -A2 "partial" docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md | grep -E "^\|" | head -30
grep "provider" docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md | grep "missing" -i
```

## Build Workflow

### 1. Understand The Atom

Summarize the selected atom in your own words:

- behavior to ship
- feature-map area or upstream concept;
- target Go package and public interface;
- exact write scope
- upstream behavior being ported (from `HERMES` column)
- active upstream file+line and why it is the right contract
- expected tests
- done signal

If any of these are missing, stop and refine the atom entry in the parity evidence doc before implementing.

Start with a zoomed-out map of the relevant public interface and callers. Do not edit until you know which behavior the user, CLI, API, channel, tool, memory, or provider boundary must expose.

Before editing, build a short builder packet:

- selected atom name and status;
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

If implementation changes public behavior, update the matching docs, generated data, or website content. If the atom is fully implemented, update its status in the parity evidence doc from `missing` → `covered` or `partial` → `covered`, and add a brief note with the commit hash or behavior proven.

Never create a parallel backlog. Missing follow-up work goes into the parity evidence doc as new atoms or status updates.

For complex atoms, use `gormes-tdd-slice` for the red-green loop. For uncertain API/package shape, use `gormes-interface-designer` before coding. For missing upstream coverage, send the work back through `gormes-parity-auditor` or `gormes-planner`.

## Goncho Builder Rules

- Internal package and product name: Goncho.
- External compatibility names: preserve Honcho-compatible `honcho_*` names when public tools or clients require them.
- Prefer local SQLite/FTS/graph-backed behavior already present in Gormes.
- Build compatibility fixtures that prove Honcho behavior without requiring live Honcho.
- Do not call Goncho done until sessions, messages, memory CRUD/search, provenance, and compatibility tool contracts are covered by tests.

## Failure Handling

If tests fail from your own changes, fix them before reporting done. If the selected atom is unbuildable because the parity classification is wrong, update the evidence doc or report the exact blocker.

If the working tree is dirty before you start, preserve existing changes. Do not revert user or previous-agent work. Work with the current `development` checkout and mention any verification limits caused by unrelated changes.

If `git push origin development` is rejected because another worker advanced the branch, fetch and inspect the upstream commit before rebasing. Keep your code/tests, update the parity evidence doc if your atom was already covered by the upstream commit, and commit only the remaining coherent delta.

## Final Report

Report:

1. Atom name and parity status before/after
2. Behavior shipped
3. Files changed
4. Tests/e2e run
5. Feature-map/evidence doc consistency
6. Parity evidence doc updates
7. Remaining blockers or next atom to build
8. Hermes source refs used to verify behavior
