---
title: "Gormes Completion Plan"
weight: 15
---

# Gormes Completion Plan

This is the execution plan for finishing `gormes-agent`. The finish line is not
an MVP and not a partial wrapper: **Gormes is complete when it is Hermes in Go,
with Goncho as the Honcho-compatible Go port inside Gormes.**

The canonical backlog remains
[`progress.json`](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/docs/content/building-gormes/architecture_plan/progress.json).
This page explains how to drive that backlog without spending unbounded planner
tokens or creating parallel queues.

Use this page with:

- [Completion Lane Roadmap](../lane-roadmap/) for phase-to-lane ownership and
  lane-specific gates.
- [Hermes And Honcho Feature Map](../hermes-honcho-feature-map/) for the
  upstream feature-to-Go package map.
- [Hermes/Honcho To Gormes Go Runtime Plan](../hermes-honcho-go-runtime-plan/)
  for the reconciled implementation plan, subsystem classification, nested
  coverage matrix, and Go package dependency order.
- [Upstream Coverage Ledger](../upstream-coverage-ledger/) for the audit rule
  that tells us whether every feature-bearing Hermes/Honcho source class has
  been mapped.
- [Swarm Feature Parity Audit](../swarm-feature-parity-audit/) for the
  feature-level gap register found by parallel sub-agent parity audits.
- [Agent Operating Model](../agent-operating-model/) for exactly how Codex,
  Claude, claudeu, and codexu should run planner, builder, parity, TDD, and
  interface-design passes.
- [Contract Readiness](../../contract-readiness/) for the row-level handoff
  fields that make a slice builder-executable.

## Non-Negotiables

1. **Hermes parity is the product definition.** CLI, agent loop, provider
   routing, tool execution, memory, skills, plugins, API, TUI, gateway,
   channels, cron, packaging, observability, and operations must have Go-native
   equivalents or explicit tested divergences.
2. **Goncho is in-process Gormes memory.** Internal code stays `goncho`; public
   compatibility can expose `honcho_*` names when tools, MCP clients, or
   existing users depend on them.
3. **`progress.json` is the only backlog.** Missing work becomes a row. Broad
   work becomes an umbrella row until split. No side TODOs, private queues, or
   agent-local task lists.
4. **Rows must be builder-executable.** A runnable row names source refs,
   write scope, test commands, acceptance, ready/not-ready conditions, and a
   done signal.
5. **Every runtime claim needs tests.** Prefer hermetic fixtures. Live provider,
   live platform, and live cloud checks are opt-in smoke tests, not row-local
   proof.
6. **Planning is bounded.** Planner passes map parity and sharpen rows. They do
   not run indefinitely and do not implement runtime code.

## Current Finish Ledger

As of the current `progress.json`, the remaining work is concentrated in the
native agent spine, tool/security surface, release surface, learning loop, and
paused channel backlog. Do not let Phase 7 channel expansion outrun the core
agent, Goncho, tool, and release lanes.

| Phase | Open rows | Planner meaning |
|---|---:|---|
| Phase 1 — Dashboard / control plane | 0 | Skill-era control rows are complete: planning/building route through canonical development skills and symlink loader views instead of deleted loop binaries. |
| Phase 2 — Gateway | 15 | Mostly channel polish, home-channel ownership, and mid-run steering. Keep these behind Lane 1/2 unless a row unblocks normal operation. |
| Phase 3 — Memory | 3 | Goncho now has an explicit drop-in compatibility closure: SDK-style harness, normal-turn integration, and empty peer-card hint. |
| Phase 4 — Brain Transplant | 30 | Highest strategic pressure: provider, context, prompt, routing, OAuth, retry, telemetry, and the new normal-turn e2e closure decide whether Gormes is really Hermes in Go. |
| Phase 5 — Final Purge | 70 | Largest backlog: tools, sandboxing, browser/media, security, CLI/API/TUI, packaging, and release. Split umbrellas before building. |
| Phase 6 — Learning Loop | 11 | Depends on the skills substrate and memory evidence; build after the skill storage/retrieval rows are sharper. |
| Phase 7 — Paused Channels | 16 | Explicit backlog. Build only fixture-ready slices or channel dependencies that unblock Lane 4. |

The first closure target is not "all green"; it is a **Python-free normal
agent turn** with local Goncho memory and tested tool-call continuation:

```text
CLI/API/gateway input
  -> Go prompt/context assembly
  -> Go provider adapter
  -> Go tool execution
  -> Goncho/memory recall and persistence
  -> Go final response + audit/status evidence
```

## Docs Spine

| Need | Start here |
|---|---|
| Overall finish line | This page |
| Phase-to-lane ownership and gates | [Completion Lane Roadmap](../lane-roadmap/) |
| Upstream feature-to-Go package map | [Hermes And Honcho Feature Map](../hermes-honcho-feature-map/) |
| Reconciled Go implementation plan | [Hermes/Honcho To Gormes Go Runtime Plan](../hermes-honcho-go-runtime-plan/) |
| Completeness audit for upstream mapping | [Upstream Coverage Ledger](../upstream-coverage-ledger/) |
| Feature-level swarm gap register | [Swarm Feature Parity Audit](../swarm-feature-parity-audit/) |
| How agents should run each pass | [Agent Operating Model](../agent-operating-model/) |
| Current generated roadmap | [Architecture Plan](../) |
| Upstream feature inventory | [Subsystem Inventory](../subsystem-inventory/) |
| Row handoff requirements | [Contract Readiness](../../contract-readiness/) |
| Skill-builder queue and selection | [Skill Builder Handoff](../../builder-loop/builder-loop-handoff/) |
| Test expectations | [Testing](../../testing/) |

## Skill-Routed Operating Model

Every substantial agent pass starts by choosing a repo-local skill. Canonical
skill files live under `docs/development-skills/`; `.agents/skills/`,
`.claude/skills/`, and `.codex/skills/` are symlink loader views.

| Situation | Skill path |
|---|---|
| Unsure what workflow applies | `gormes-skill-manager` |
| Mapping upstream Hermes/Honcho/GBrain gaps | `gormes-parity-auditor` |
| Updating `progress.json`, phases, or docs | `gormes-planner` |
| Designing a Go package/API boundary | `gormes-interface-designer` |
| Implementing one row | `gormes-builder` |
| Delivering one behavior with red-green-refactor | `gormes-tdd-slice` |
| Stress-testing a plan with the operator | `grill-me` |

The default flow is:

```text
parity audit -> planner row refinement -> builder row execution -> TDD slice -> validation
```

Do not recreate the old loop binaries. `gormes-planner` and `gormes-builder`
are manual skill-routed workflows; repository evidence and `progress.json`
are the source of truth.

The operating rule is:

```text
skill -> bounded scan -> row/doc change -> validation -> short handoff
```

If a pass cannot name its lane, subsystem, expected files, and validation gates,
it is too vague to run.

## New Closure Subphases

The roadmap now has explicit closure subphases for the work that was previously
spread across prose or broad phase headings.

| Subphase | Purpose | First rows |
|---|---|---|
| 1.D — Skill-Driven Control Plane | Keep all agents on skills + `progress.json` after deleting loop binaries. | Skill-manager selection matrix hardening; skill-pack coverage audit. |
| 3.G — Goncho Drop-In Compatibility Closure | Prove Goncho is the Honcho-compatible Go memory port, not just local memory pieces. | Goncho Honcho SDK compatibility e2e harness; Goncho memory integration into normal agent turn. |
| 4.I — Native Agent Turn Closure | Prove the actual Hermes-in-Go normal turn across provider, tools, memory, final response, and audit. | Python-free normal agent turn e2e harness; provider-tool-memory golden transcript suite; Hermes/Honcho feature map. |

## No-Loop Execution Ladder

The deleted loop binaries are replaced by this repeatable ladder. Every agent
uses it; no private scheduler, side queue, or ad hoc task list is allowed.

| Step | Skill | Output | Validation |
|---|---|---|---|
| 1. Route | `gormes-skill-manager` | One selected workflow and reason. | None beyond naming the skill. |
| 2. Audit | `gormes-parity-auditor` | Covered/planned/vague/missing/owned map for one lane surface. | Exact upstream and Gormes paths named. |
| 3. Plan | `gormes-planner` | Updated docs/rows with builder-ready contracts. | `go run ./cmd/progress validate`, docs/progress tests when changed. |
| 4. Design | `gormes-interface-designer` when needed | Chosen Go package/API boundary. | Row updated with write scope and tests. |
| 5. Build | `gormes-builder` | One row implemented. | Row-local tests, focused package gate, progress validation. |
| 6. TDD | `gormes-tdd-slice` | Red-green-refactor evidence for the behavior. | Test first or explicit reason why not feasible. |
| 7. Handoff | Same skill used for the pass | Done signal, files changed, tests run, next row. | Final report is resumable. |

If a row is not executable by step 5, the correct action is to return to step
3 and sharpen the row. Do not compensate by asking a builder to rediscover the
architecture.

## Completion Lanes

These lanes cut across the existing phases. Each lane is done only when the
corresponding `progress.json` rows are shipped and tests prove the user-visible
contract.

### Lane 0 — Control Plane Discipline

Goal: make autonomous work reliable enough to finish the product.

Done means:

- builder skills select only rows with test proof or explicit `no_test_required`;
- planner rows preserve row health fields and do not mutate runtime code;
- invalid `progress.json` blocks work until planning fixes it;
- repo-local skills route all future agent work;
- docs, progress rows, and generated queue pages agree.

Primary gates for control-plane code changes:

```sh
go test ./internal/progress -count=1
go run ./cmd/progress validate
go test ./docs -count=1
```

Planner-doc passes should use non-loop validation only:

```sh
go run ./cmd/progress validate
go test ./internal/progress -count=1
go test ./docs -count=1
```

### Lane 1 — Native Agent Spine

Goal: replace Hermes' Python `run_agent.py` responsibilities with Go-native
provider, prompt, context, kernel, retry, tool-call, and telemetry contracts.

Done means:

- provider adapters normalize requests, streaming, usage, errors, and retries;
- prompt/context assembly handles project instructions, skills, memory, session
  search, compression, redaction, and references;
- tool-call parsing and repair happen before tool execution;
- model routing, cost, budgets, and provider degradation are visible in status
  and audit logs;
- the agent loop can run without Python for normal chat/tool sessions.

### Lane 2 — Goncho Memory And Honcho Compatibility

Goal: make in-process Goncho the memory substrate while preserving
Honcho-compatible public behavior.

Done means:

- sessions, messages, users/workspaces, memories/facts, search, provenance,
  timestamps, updates, and deletions are available through Go APIs;
- public compatibility fixtures cover required `honcho_*` tool/MCP names;
- SQLite/FTS/graph storage is local and auditable;
- cross-session recall, source scoping, parent lineage, and import/export are
  tested without a live Honcho service;
- memory injection into the agent loop is deterministic and bounded.

### Lane 3 — Tool Surface, Security, And Skills

Goal: port Hermes' tool ecosystem without copying Python gravity.

Done means:

- tool descriptors drive schemas, CLI/gateway exposure, doctor checks, audit
  kinds, trust classes, timeouts, and result budgets;
- core file, shell, web, browser, image, audio, sandbox, MCP, ACP, plugin,
  approval, and operator tools have Go contracts or explicit deferred rows;
- toolset restrictions and availability checks prevent impossible tool calls;
- the skills runtime supports discovery, install/sync, guard metadata,
  preprocessing, slash-command exposure, and lockfile provenance;
- dangerous tools are covered by path, URL, approval, and policy tests before
  they are exposed to untrusted callers.

### Lane 4 — Gateway, Channels, Cron, And Delivery

Goal: make Gormes usable from every supported interface with one shared runtime.

Done means:

- Telegram, Discord, Slack, WhatsApp, WeChat, Email/SMS, Webhook, API, and
  paused long-tail channels either ship or have explicit deferral rows;
- session context, delivery routing, home channel, contact directory, pairing,
  restart, status, hooks, and mid-run steering are unified across channels;
- cron/admin/tool/API control surfaces share one scheduler and audit store;
- platform-specific live checks are optional, with fake adapters proving row
  behavior by default.

### Lane 5 — CLI, API, TUI, Packaging, And Release

Goal: make the Go binary the only runtime operators need.

Done means:

- `gormes` covers Hermes CLI command groups or tested divergences;
- OpenAI-compatible HTTP surfaces, Responses/Runs streaming, health, cron admin,
  dashboard-facing contracts, and disconnect/cancel snapshots are native Go;
- Bubble Tea TUI startup, provider/model overrides, status, copy policy, and
  streaming are independent of Node/Ink bundles;
- Unix/Windows installers, service units, offline doctor, version output,
  release artifacts, and docs are fixture-backed and Python-free.

### Lane 6 — Learning Loop

Goal: make Gormes improve itself through durable skills and evidence.

Done means:

- complex tasks can be detected, distilled into skills, scored, improved, and
  safely promoted;
- skill usage is logged and linked to outcomes;
- failed or stale skills become planner-visible work;
- learning loop behavior builds on the Phase 5 skills substrate instead of
  creating a second skill system.

## First Ordered Passes

These are the next planner/builder passes that should happen before expanding
the roadmap again:

1. **Parity audit: Native agent spine.** Use `gormes-parity-auditor` on Hermes
   `run_agent.py`, provider adapters, prompt/context, retry, and tool-call
   repair. Output missing/vague rows only.
2. **Planner pass: Phase 4 row readiness.** Use `gormes-planner` to split broad
   Phase 4 rows into small provider/context/kernel tracer bullets with focused
   tests.
3. **Builder pass: one provider boundary row.** Use `gormes-builder` +
   `gormes-tdd-slice` to ship exactly one provider behavior through the public
   Go interface.
4. **Parity audit: Goncho/Honcho.** Compare `../honcho` concepts and MCP/docs
   against `internal/goncho`, `internal/gonchotools`, `internal/memory`, and
   Phase 3/5.I rows.
5. **Planner pass: Goncho compatibility rows.** Ensure every public `honcho_*`
   compatibility behavior has a hermetic fixture and builder-ready row.
6. **Builder pass: one Goncho compatibility row.** Ship one request/response or
   tool contract with tests.
7. **Parity audit: tool descriptors.** Map Hermes tool registry/toolsets into
   descriptor-first Go rows before any large handler ports.
8. **Builder pass: one descriptor-to-schema slice.** Prove one tool descriptor
   drives schema, availability, audit, and doctor output.
9. **Planner pass: API/TUI/packaging dependency order.** Split Phase 5.O-5.Q
   into API contract, CLI command, TUI, installer, and service-manager slices
   with no Node/Python runtime assumptions.
10. **Release readiness pass.** Use docs, doctor, and e2e gates to identify the
    smallest Python-free operator path, then build it row by row.

## Next Skill-Routed Targets

Use this table to keep the next few passes concrete. If a row here is blocked
or too vague, update `progress.json` instead of skipping silently.

| Order | Row | Why it matters | Skill chain |
|---:|---|---|---|
| 1 | `Python-free normal agent turn e2e harness` | Defines the first honest Hermes-in-Go closure test across provider, tools, Goncho memory, final response, and audit evidence. | `gormes-builder` -> `gormes-tdd-slice` |
| 2 | `Goncho Honcho SDK compatibility e2e harness` | Proves Goncho as the Honcho-compatible Go port with SDK-style local flows. | `gormes-builder` -> `gormes-tdd-slice` |
| 3 | `Goncho empty peer-card hint contract` | Improves Honcho-compatible diagnostics and unblocks the Goncho closure harness. | `gormes-builder` -> `gormes-tdd-slice` |
| 4 | `ContextEngine compression-boundary callback vocabulary` | Gives Phase 4 a precise context/compression callback contract before kernel binding. | `gormes-builder` -> `gormes-tdd-slice` |
| 5 | `Provider-tool-memory golden transcript suite` | Turns the normal-turn harness into repeatable regression fixtures. | `gormes-builder` -> `gormes-tdd-slice` |
| 6 | `Provider image-too-large error classification` | Hardens provider failure taxonomy before image retry and multimodal rows. | `gormes-builder` -> `gormes-tdd-slice` |

Rows not listed here can still be built, but a planner pass should explain why
they outrank Lane 1/2 closure.

## Hard Dependency Order

1. **Lane 0 remains enforced at all times.** Progress validation, skill routing,
   and generated docs must stay green before runtime work expands.
2. **Lane 1 before broad tools.** Provider/context/kernel/tool-call continuity
   must be stable before porting dozens of tool handlers.
3. **Lane 2 before memory-heavy UX.** Goncho/Honcho compatibility must be
   fixture-complete before learning-loop and advanced session UX claims.
4. **Lane 3 before untrusted exposure.** Tool descriptors, trust classes,
   approval policy, and availability checks land before gateway/API exposure.
5. **Lane 4 before release polish.** Shared gateway/session/delivery behavior
   must be unified before packaging markets Gormes as multi-channel.
6. **Lane 5 before public install promises.** Installers, services, API health,
   and docs must match the real binary.
7. **Lane 6 after skill substrate maturity.** Learning loop work builds on
   reviewed skills, retrieval, and outcome evidence.

## Pass Templates

### Parity Audit Template

Use `gormes-parity-auditor`.

1. Pick one lane and one upstream surface.
2. List exact upstream paths and symbols.
3. List exact Gormes packages/tests/progress rows.
4. Classify every behavior as covered, planned, vague, missing, or owned.
5. Propose only the missing/vague rows that unblock the next builder pass.

### Planner Template

Use `gormes-planner`.

1. Read the parity audit or current lane docs.
2. Update docs or `progress.json`, not runtime code.
3. Split umbrella work into tracer-bullet rows.
4. Preserve builder-owned health blocks.
5. Validate docs/progress.
6. Report the next three builder-ready rows.

### Builder Template

Use `gormes-builder` and `gormes-tdd-slice`.

1. Select one row.
2. Write one failing public-behavior test.
3. Implement the smallest passing behavior.
4. Repeat vertically only inside the same row.
5. Run row-local and lane gates.
6. Update evidence and stop.

## Definition Of Done By Row

A row is done when:

- the behavior is observable through a public Go interface;
- row-local tests prove the behavior with no live credentials by default;
- required docs/web surfaces are updated;
- `go run ./cmd/progress validate` passes;
- the relevant focused package tests pass;
- broad shared changes also pass `go test ./... -count=1`;
- the final report names the done signal and remaining follow-up rows.

## Risk Burn-Down

| Risk | Burn-down rule |
|---|---|
| Planner token drain | Run bounded skill passes; stop after row/doc validation. |
| Broad rows that workers cannot finish | Split into tracer-bullet rows before assignment. |
| Python runtime leakage | Every lane must prove no Python dependency in the operator path it claims. |
| Node/Ink TUI leakage | Treat Gormes Bubble Tea independence as a tested divergence. |
| Live provider/platform brittleness | Use fake clients and hermetic fixtures for row proof. |
| Duplicate memory/skills substrates | Reuse Goncho and Phase 2.G skills store; do not create parallel stores. |
| Tool schema drift | Descriptor-first rows before handler ports. |
| Public messaging drift | Sync `progress.json`, generated docs, and `www.gormes.ai` when roadmap status changes. |

## Operating Cadence

Use this loop until Gormes is complete:

1. Pick one lane.
2. Audit parity for that lane.
3. Refine the smallest next rows in `progress.json`.
4. Build one row with TDD.
5. Run the relevant gates.
6. Update docs/progress evidence.
7. Repeat.

When a pass discovers a reusable agent workflow, route through
`gormes-skill-manager` and create a new repo-local skill only if the workflow
will recur and has distinct validation needs.
