---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
autonomous implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared unattended-loop facts live in [Autoloop Handoff](../autoloop-handoff/):
the main entrypoint, orchestrator plan, candidate source, generated docs,
tests, and candidate policy. Keep those control-plane facts in
`meta.autoloop`, and keep row-specific execution facts in `progress.json`.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Tool registry inventory + schema parity harness

- Phase: 5 / 5.A
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Contract: Operation and tool descriptor parity before handler ports
- Trust class: operator, gateway, child-agent, system
- Ready when: Upstream tool descriptor inventory can be captured without porting handlers in the same slice.
- Not ready when: Handler implementation starts before descriptor parity fixtures exist.
- Degraded mode: Doctor reports disabled tools, missing dependencies, schema drift, and unavailable provider-specific paths.
- Fixture: `internal/tools upstream schema parity manifest fixtures`
- Write scope: `internal/tools/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -count=1`
- Done signal: Tool descriptor parity fixtures capture names, schemas, trust classes, dependencies, and degraded status before handler ports.
- Acceptance: Upstream tool names, toolsets, required env vars, schemas, result envelopes, trust classes, and degraded status are captured in fixtures., No handler port can mark complete until its descriptor parity row exists., Doctor can report missing dependencies or disabled provider-specific paths.
- Source refs: docs/content/upstream-hermes/reference/tools-reference.md, docs/content/building-gormes/architecture_plan/phase-5-final-purge.md
- Unblocks: Pure core tools first, Stateful tool migration queue, CLI command registry parity + active-turn busy policy
- Why now: Unblocks Pure core tools first, Stateful tool migration queue, CLI command registry parity + active-turn busy policy.

<!-- PROGRESS:END -->
