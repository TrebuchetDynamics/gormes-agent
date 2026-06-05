---
name: gormes-delivery-loop
description: Run bounded architecture → planner → Hermes parity → builder/TDD cycles. Use when asked for the Gormes delivery loop or to repeat that skill chain with validation controls.
---

# Gormes Delivery Loop

Run a bounded in-agent delivery loop that turns architecture findings into canonical progress rows, validates the active Hermes/Honcho contract, implements one builder-ready slice, and repeats only while the next slice is safe and evidence-backed.

This is a skill extension for bounded delivery-loop orchestration.

## Branch Rule

Stay on the existing `development` branch. Do not create feature branches or worktrees. If the checkout is not on `development`, stop before editing.

## Inputs

- Optional loop topic, for example `learning loop`, `Termux runtime`, `provider auth`, or `gateway channels`.
- Optional iteration budget. Default: **1 full iteration**. Maximum without explicit user confirmation: **3 iterations**.
- Optional stop gate, for example focused package tests, `go test ./... -count=1`, progress validation, PR checks, or a release gate.

## Loop Shape

Each iteration is one vertical slice:

```text
architecture review -> progress row shaping -> parity evidence -> builder/TDD -> validation -> commit/push -> decide next iteration
```

### 0. Preflight And Scope Guard

Run:

```sh
pwd
git rev-parse --show-toplevel
git rev-parse --abbrev-ref HEAD
git status --short --branch --untracked-files=all
```

If unrelated dirty work exists, name it and preserve it. Do not run broad formatters or generators. Stage only files in the current slice.

### 1. Architecture Candidate

Use `gormes-architecture-zoomout` for the selected topic. Produce or update one compact packet:

```text
architecture_review_packet:
  area:
  candidate:
  score:
  smell:
  evidence_quality:
  preserve_contracts:
  characterization_test:
  allowed_write_scope:
  forbidden_scope:
  next_skill:
```

Continue only when evidence quality is A or B and the packet names one characterization test plus one allowed write scope.

### 2. Planner Row Shaping

Use `gormes-planner` when the packet changes backlog shape or lacks a builder-ready row.

Rules:

- Use parity evidence docs for source-backed classification, not as a backlog.
- Use `cmd/progress` / `internal/planning/progress` for logical backlog rows; do not hand-create side queues.
- Update existing rows before adding new rows.
- A row must have concrete `write_scope`, `test_commands`, `ready_when`, and `not_ready_when` before builder work.

Validation after planner-only edits:

```sh
go run ./cmd/progress validate
git diff --check
```

### 3. Hermes/Honcho Parity Evidence

Use `gormes-hermes-parity` to confirm the active upstream contract before implementation.

Required output:

```text
parity_packet:
  topic:
  active_upstream_contract:
  source_refs:
  gormes_surface:
  classification: covered|missing|stale-upstream|owned|excluded
  builder_row:
  red_test_hint:
  validation:
```

Stop if `$HERMES_SRC`/`$HONCHO_SRC` cannot be resolved for a parity claim that depends on it, unless the row is Gormes-owned and source-backed locally.

### 4. Builder/TDD Slice

Use `gormes-builder` plus `gormes-tdd-slice` for exactly one builder-ready row.

Rules:

- Write or identify a failing characterization test first when behavior changes.
- Preserve public contracts unless the progress row explicitly changes them.
- Keep implementation inside the row `write_scope`; if scope is wrong, return to planner.
- Implement the smallest vertical behavior that makes the test pass.

### 5. Validation And Git Delivery

Run the row's `test_commands`, then focused package tests for touched packages. Broaden to the full gate when the slice touches shared runtime behavior:

```sh
go test ./... -count=1
go run ./cmd/progress validate
git diff --check
```

Commit and push each validated slice on `development` in a coherent commit. If a commit is not pushed, report the exact blocker.

### 6. Continue Or Stop

Continue to the next iteration only if all are true:

- current slice is validated;
- current slice is committed and pushed, or user explicitly asked not to commit;
- the next candidate has A/B evidence;
- no unrelated dirty work would be touched;
- iteration budget remains.

Otherwise stop with a handoff packet.

## Stop Conditions

Stop immediately when:

- branch is not `development`;
- required upstream evidence is missing for a parity claim;
- the next row is not builder-ready;
- validation fails twice with the same blocker;
- the slice would require touching unrelated dirty files;
- a public contract change lacks a progress row;
- the loop would need live credentials, private home data, or external services not explicitly authorized by the user.

## Output Shape

```text
Delivery loop iteration <n>/<budget>
Topic:
Architecture packet:
Planner row/action:
Parity packet:
Builder row/action:
Files changed:
Validation:
Commit/push:
Next iteration decision:
Blocked work:
Pivoted work completed:
```

## Anti-Patterns

- Do not run unattended cron jobs, background daemons, or private JSON queues without explicit user approval.
- Do not run unbounded loops. A human-readable budget is mandatory.
- Do not skip parity evidence and jump from architecture vibes to implementation.
- Do not mark rows complete without tests or exact no-test rationale.
- Do not mix multiple unrelated architecture candidates into one commit.
