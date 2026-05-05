---
title: "Skill Builder Handoff"
weight: 10
aliases:
  - /building-gormes/autoloop-handoff/
  - /building-gormes/autoloop/autoloop-handoff/
---

# Skill Builder Handoff

This page is generated from `meta.builder_loop` in the canonical progress
file. The field name is historical; the active workflow is skill-driven.

It keeps shared skill handoff facts in one place so agents do not guess the
entrypoint, plan, candidate source, generated docs, or selection policy from
scattered prose. Row-specific execution facts stay in
[Agent Queue](../agent-queue/) and canonical progress rows.

`gormes-builder` and `gormes-tdd-slice` replace the deleted builder-loop
command. They select work from `progress.json`, use the
generated building-gormes pages as the operator-facing handoff, and carry
selected row metadata into the implementation pass. Do not maintain a parallel
queue outside this docs tree.

## Skill-Routed Handoff

Before a worker starts, the pass must route through repo-local skills:

1. Use `gormes-builder` to select and restate the row.
2. Use `gormes-tdd-slice` for the red-green-refactor loop.
3. If the row is vague, stop and send it back through `gormes-planner`.
4. If upstream parity is unclear, stop and use `gormes-parity-auditor`.
5. If the package/API shape is unclear, stop and use
   `gormes-interface-designer`.

This is not optional process overhead. The skill route is how Gormes avoids
token-expensive planner drift and keeps every implementation pass tied to the
Hermes-in-Go completion plan.

## Row Selection Contract

A row is ready for a builder skill when all are true:

- it is not `complete`;
- it is not `slice_size: umbrella`;
- all `blocked_by` entries are satisfied or irrelevant to the slice;
- `ready_when` names locally checkable prerequisites;
- `write_scope` names concrete files, directories, or packages;
- `test_commands` exists, or `no_test_required` explains the rare exception;
- `acceptance` and `done_signal` describe observable behavior.

If any condition is false, use `gormes-planner` to repair the row before
implementation.

## Worker Definition Of Done

A worker pass is complete only when:

- the selected row's public behavior is implemented;
- row-local `test_commands` pass;
- focused package tests for touched code pass;
- progress validation still passes;
- docs/web surfaces are updated when public behavior changed;
- the final report names the done signal and any follow-up row.

<!-- PROGRESS:START kind=builder-loop-handoff -->
## Control Plane

- Entrypoint: `docs/development-skills/gormes-skill-manager/SKILL.md`
- Plan: `docs/content/building-gormes/architecture_plan/completion-plan.md`
- Candidate source: `docs/content/building-gormes/architecture_plan/progress.json`
- Agent queue: `docs/content/building-gormes/builder-loop/agent-queue.md`
- Progress schema: `docs/content/building-gormes/builder-loop/progress-schema.md`
- Unit tests: `go test ./internal/progress -count=1`

## Candidate Policy

- Skip rows with blocked_by until ready_when is satisfied.
- Skip slice_size=umbrella rows until they are split.
- Use gormes-builder and gormes-tdd-slice for one row at a time; do not recreate the deleted builder-loop selector.
- Use gormes-planner to split vague rows before implementation.
- A builder pass must restate the row contract, write_scope, test_commands, acceptance, and done_signal before editing.
- Run row-local tests first, then focused package tests, then go run ./cmd/progress validate.
- Update progress evidence only for the row being built.
- If validation fails from stale loop assumptions, fix the docs or progress row instead of invoking old loop binaries.
- Canonical skill files live in docs/development-skills; .agents/skills, .claude/skills, and .codex/skills are symlink loader views.
- Prefer contract rows with write_scope, test_commands, and done_signal.
- Do not create side queues; missing work becomes a progress.json row.
<!-- PROGRESS:END -->
