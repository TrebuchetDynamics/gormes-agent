---
title: "Builder Loop Handoff"
weight: 10
aliases:
  - /building-gormes/autoloop-handoff/
  - /building-gormes/autoloop/autoloop-handoff/
---

# Builder Loop Handoff

This page is generated from `meta.builder_loop` in the canonical progress
file: `docs/content/building-gormes/architecture_plan/progress.json`.

It keeps shared unattended-loop facts in one place so autonomous workers do
not guess the entrypoint, plan, candidate source, generated docs, or
selection policy from scattered prose. Row-specific execution facts stay in
[Agent Queue](../agent-queue/) and canonical progress rows.

The `builder-loop` command is the executor side of the
Planner-Builder Loop architecture (see `AGENTS.md` at the repo root): it selects work from
`progress.json`, uses the generated building-gormes pages as the
operator-facing handoff, and passes selected row metadata into worker
prompts so the agent loop can develop the full `gormes-agent` one phase
slice at a time. Do not maintain a parallel queue outside this docs tree.

<!-- PROGRESS:START kind=builder-loop-handoff -->
## Control Plane

- Entrypoint: `scripts/gormes-auto-codexu-orchestrator.sh`
- Plan: `docs/superpowers/plans/2026-04-24-orchestrator-oiling-release-1-plan.md`
- Candidate source: `docs/content/building-gormes/architecture_plan/progress.json`
- Agent queue: `docs/content/building-gormes/builder-loop/agent-queue.md`
- Progress schema: `docs/content/building-gormes/builder-loop/progress-schema.md`
- Unit tests: `scripts/orchestrator/tests/run.sh unit`

## Candidate Policy

- Skip rows with blocked_by until ready_when is satisfied.
- Skip slice_size=umbrella rows until they are split.
- Default cmd/builder-loop CLI runs cap eligible roadmap work at Phase 4 unless MAX_PHASE is explicitly overridden; the production service unit sets MAX_PHASE=0 so the planner-builder loop can keep advancing across phases.
- MAX_AGENTS is a safety cap: if fewer metadata-ready rows are available, run fewer workers instead of selecting filler or random work.
- Each worker runs in an isolated git worktree under RUN_ROOT/worktrees and promotion rejects committed paths outside the selected row's write_scope.
- When git worktrees are available and MAX_AGENTS is greater than 1, cmd/builder-loop launches selected workers concurrently, then validates and promotes each branch through the same ledgered safety gates.
- When PRE_PROMOTION_VERIFY_COMMANDS is configured, cmd/builder-loop verifies the worker branch before cherry-picking, repairs failures on that worker branch, and keeps main untouched until the gate passes.
- After all promotions, cmd/builder-loop runs the mandatory post-promotion full-suite gate before emitting run_completed or health_updated.
- On post-promotion gate failure, cmd/builder-loop starts one backend repair attempt by default, requires the checkout to be clean, reruns the suite, and records final health only if the gate passes.
- Backend worker failures preserve captured stderr/stdout detail in the ledger so repair and planner loops can diagnose the real failure instead of only seeing backend_failed.
- Prefer contract rows with write_scope, test_commands, and done_signal.
- Inject selected progress metadata into the worker prompt instead of asking workers to rescan the whole roadmap.
<!-- PROGRESS:END -->
