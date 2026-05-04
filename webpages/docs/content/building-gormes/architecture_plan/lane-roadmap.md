---
title: "Completion Lane Roadmap"
weight: 16
---

# Completion Lane Roadmap

The phase pages describe historical delivery order. Completion lanes describe
the remaining product work required for Gormes to become Hermes in Go. The
default rule is parity with Hermes across runtime behavior, config, commands,
providers, and operator experience; any Go-native divergence must be explicit,
tested, and visible to operators.

Use this page when deciding what to audit, plan, or build next. Use
[`progress.json`](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/docs/content/building-gormes/architecture_plan/progress.json)
for the actual queue, and use [Hermes And Honcho Feature Map](../hermes-honcho-feature-map/)
when deciding where an upstream feature belongs in Go.

## Lane Crosswalk

| Lane | Owns | Primary phases | Main packages/docs |
|---|---|---|---|
| 0 — Control Plane Discipline | skill routing, row quality, progress validation gates | 1.C, 1.D, generated handoff docs | `docs/development-skills/`, `.agents/skills/`, `.claude/skills/`, `.codex/skills/`, `cmd/progress`, `cmd/repoctl`, `internal/progress`, `AGENTS.md`, `CLAUDE.md` |
| 1 — Native Agent Spine | provider adapters, prompt/context, kernel loop, retry, compression, model routing, telemetry, normal-turn e2e | 4.A-4.I, parts of 2.E, 5.Q | `internal/hermes`, `internal/kernel`, `internal/e2e`, `internal/gateway`, `internal/telemetry`, `docs/content/building-gormes/architecture_plan/phase-4-brain-transplant.md` |
| 2 — Goncho Memory And Honcho Compatibility | local memory, Honcho-compatible public contracts, session lineage, scoped recall, SDK-style local harness | 3.E, 3.F, 3.G, 5.I, 5.N, 4.B | `internal/goncho`, `internal/gonchotools`, `internal/memory`, `internal/session`, `docs/content/building-gormes/goncho_honcho_memory/` |
| 3 — Tool Surface, Security, And Skills | tool descriptors, toolsets, plugins, MCP/ACP, file/shell/browser/media tools, approvals, skill plumbing | 2.A, 2.G, 5.A-5.M | `internal/tools`, `internal/plugins`, `internal/skills`, `internal/doctor`, `docs/content/building-gormes/architecture_plan/phase-5-final-purge.md` |
| 4 — Gateway, Channels, Cron, And Delivery | shared gateway, adapters, active-turn policy, home channel, pairing, restart, cron, delivery routing | 2.B-2.F, 7, 5.N, 5.Q | `internal/gateway`, `internal/cron`, `internal/channels/*`, `internal/slack`, `internal/discord`, `docs/content/building-gormes/gateway-donor-map/` |
| 5 — CLI, API, TUI, Packaging, And Release | Hermes CLI parity, OpenAI-compatible API, TUI gateway, installers, services, release docs | 5.O-5.Q, 5.P, Phase 1 TUI divergence | `cmd/gormes`, `internal/cli`, `internal/apiserver`, `internal/tui`, `internal/tuigateway`, `www.gormes.ai/` |
| 6 — Learning Loop | skill detection, extraction, review, retrieval, feedback, promotion | 6.A-6.F | `internal/skills`, `internal/memory`, `internal/kernel`, `docs/development-skills/`, Phase 6 docs |

## Lane Exit Gates

## Lane Dependency Graph

```text
Lane 0 Control Plane
  -> Lane 1 Native Agent Spine
  -> Lane 2 Goncho Memory
  -> Lane 3 Tools / Security / Skills
  -> Lane 4 Gateway / Cron / Delivery
  -> Lane 5 CLI / API / TUI / Release
  -> Lane 6 Learning Loop
```

The graph is directional for planning priority, not a total freeze. A row may
advance out of order when it is fixture-ready and either unblocks an earlier
lane, closes an operator-control parity gap, or is clearly isolated from the
normal agent turn. In particular, CLI/config/provider/TUI parity rows are not
late polish just because some of them live in Lane 5; they define how operators
configure, run, steer, inspect, and recover Gormes.

## Lane Pressure

| Lane | Current pressure | Planner action |
|---|---|---|
| 0 | Command loops removed; skills/progress commands are the control plane, with Phase 1.D now tracking skill-era control work. | Keep docs/tests aligned and prevent old loop references from returning. |
| 1 | Phase 4 now has an explicit 4.I normal-turn closure subphase. | Build the Python-free e2e harness, then split broad provider/context rows around failing fixture evidence. |
| 2 | Phase 3 now has explicit 3.G Goncho drop-in closure rows. | Close peer-card hints, then build the SDK-style harness and normal-turn memory integration. |
| 3 | Phase 5 has many tool/security umbrellas. | Convert umbrellas into descriptor-first rows before handler ports. |
| 4 | Phase 2/7 channel backlog remains large. | Keep long-tail adapters paused unless fixture-ready or dependency-unblocking. |
| 5 | API/TUI/packaging rows are mixed with helper umbrellas, but CLI/config/TUI experience parity is part of the core Hermes contract. | Separate operator-visible CLI/API/TUI behavior from helper-file inventory, and keep command/config/provider-control rows eligible whenever they unblock dogfood or reduce operator drift from Hermes. |
| 6 | All learning rows remain planned. | Build only after skill storage/retrieval rows have fixture proof. |

### Lane 0 — Control Plane Discipline

Exit when:

- every active/P0 row has `test_commands` or `no_test_required`;
- invalid progress blocks builder selection until a planner pass fixes it;
- repo-local skill routing is documented in `AGENTS.md` and `CLAUDE.md`;
- `docs/development-skills/` is the canonical skill source and loader
  directories are symlinks;
- generated queue pages expose sane next work.

Gate:

```sh
go test ./internal/progress -count=1
go run ./cmd/progress validate
go test ./docs -count=1
```

### Lane 1 — Native Agent Spine

Exit when:

- a normal CLI/API/gateway agent turn can run entirely through Go provider,
  prompt/context, memory, tool-call, retry, and finalization paths;
- Phase 4.I `Python-free normal agent turn e2e harness` is validated or every
  failing step has a narrower progress row;
- provider-specific behavior is isolated behind adapters and transcript
  fixtures;
- cancellation and retry behavior are observable and audited.

Gate:

```sh
go test ./internal/hermes ./internal/kernel ./internal/gateway ./internal/telemetry -count=1
# After Phase 4.I creates internal/e2e, include: go test ./internal/e2e -count=1
```

### Lane 2 — Goncho Memory And Honcho Compatibility

Exit when:

- local Goncho covers Honcho sessions/messages/memory/search/provenance
  contracts required by Gormes;
- public compatibility names and fixtures exist for required `honcho_*`
  behavior;
- Phase 3.G proves local SDK-style compatibility and normal-turn memory
  integration without hosted Honcho;
- scoped recall and session lineage are deterministic.

Gate:

```sh
go test ./internal/goncho ./internal/gonchotools ./internal/memory ./internal/session ./internal/store -count=1
```

### Lane 3 — Tool Surface, Security, And Skills

Exit when:

- descriptors drive schemas, toolsets, availability, audit, doctor, and
  prompt-visible exposure;
- dangerous operations are guarded before untrusted callers can reach them;
- Phase 2.G skills substrate supports the remaining Phase 5.F mechanics.

Gate:

```sh
go test ./internal/tools ./internal/plugins ./internal/skills ./internal/doctor -count=1
```

### Lane 4 — Gateway, Channels, Cron, And Delivery

Exit when:

- shared gateway policy owns active-turn behavior, command dispatch, delivery,
  pairing, home channel, status, and restart;
- priority channels are stable and paused channels have explicit rows;
- cron, API, CLI, and tool control surfaces share one scheduler/store.

Gate:

```sh
go test ./internal/gateway ./internal/cron ./internal/discord ./internal/slack ./internal/channels/... -count=1
```

### Lane 5 — CLI, API, TUI, Packaging, And Release

Exit when:

- the `gormes` binary covers Hermes-compatible operator CLI, API, TUI, service,
  installer, and release workflows without Python or Node runtime requirements;
- command names, aliases, root flags, slash commands, gateway handlers,
  dynamic plugin commands, config precedence, profiles, provider/model
  selection, logs/status/doctor/backup/update flows, and secret redaction have
  Go-native equivalents or explicit tested divergences;
- the local TUI and gateway surfaces preserve the Hermes operator experience
  for long coding turns: slash completion, busy-turn steering, status/footer
  evidence, tool progress, approvals, interrupts, edits, and recoverable
  failure messages;
- public docs and `www.gormes.ai` match the shipped install/runtime behavior.

Gate:

```sh
go test ./cmd/gormes ./internal/cli ./internal/apiserver ./internal/tui ./internal/tuigateway -count=1
go test ./docs -count=1
(cd webpages/landing && go test ./... -count=1)
```

### Lane 6 — Learning Loop

Exit when:

- complex tasks can become reviewed, versioned skills;
- skill use is tied to outcomes and operator feedback;
- generated or stale skills never enter prompt injection without review.

Gate:

```sh
go test ./internal/skills ./internal/memory ./internal/kernel -count=1
```

## Next-Lane Priority

Until Lane 1 and Lane 2 are closed, prefer work that removes Python from the
normal agent turn, makes Goncho/Honcho compatibility testable, or closes a
Hermes operator-control gap needed to run that work safely. The current closure
rows are Phase 4.I and Phase 3.G plus the Phase 5.O/5.Q command, config,
provider-control, and TUI-experience rows they depend on. Channel expansion,
browser/media tools, plugins, and release packaging should not outrun those
lanes unless a dependency row explicitly requires them.

## Planner Split Rules

Use these when a lane row is too broad:

| Broad row shape | Split into |
|---|---|
| Provider adapter umbrella | request normalization, stream events, tool-call replay, usage/cost, retry/error taxonomy, credential source. |
| Context/compression umbrella | budget accounting, protected head/tail, tool-result pruning, callback vocabulary, kernel binding, operator status. |
| Tool registry umbrella | descriptor schema, argument validation, availability/degraded mode, trust class, handler, audit/doctor binding. |
| CLI tree umbrella | one command group plus config/profile fixtures, not dozens of files at once. |
| Channel adapter umbrella | inbound normalization, command passthrough, outbound send, identity/self-filter, bootstrap, live-smoke docs. |
| Goncho/Honcho umbrella | one public `honcho_*` tool/API behavior with SQLite fixtures and no live Honcho dependency. |

## Go Donor Preflight

Before a split row names a new Go runtime shape, run the `gormes-references`
workflow, use the [Go Donor Reference Map](../go-donor-reference-map/), and
cite the exact donor file in `source_refs`. The planner should still use
Hermes/Honcho/GBrain as the behavior contract; Go donors only answer
implementation shape questions.

| Lane | Donor preflight |
|---|---|
| 1 — Native Agent Spine | Use the provider pattern references for OAuth, retry, stream errors, empty endpoint guards, and prompt-cache failure visibility; use Nanobot/Axe only for token-budget and output-budget helpers. |
| 2 — Goncho Memory | Use Engram for SQLite/FTS5, relation verbs, deterministic write queues, and redacted activity logging; preserve Honcho-compatible public names. |
| 3 — Tools / Security / Skills | Use Nanobot for tool-host boundaries and result truncation, trpc-agent-go for callback separation, and Axe for small artifact/budget utilities. |
| 4 — Gateway / Cron / Delivery | Use trpc-agent-go await-user-reply and agentcontrolplane state-machine patterns for pause/resume, cancellation, and visible stuck-turn state; do not import their orchestration stack. |
| 5 — CLI / API / TUI / Release | Use donors only for runtime mechanics behind the operator surface; Hermes remains the source for command names, output, config, and TUI behavior. |
| 6 — Learning Loop | Use GBrain/Hermes for skill lifecycle behavior first, then Go donors only for storage, audit, and workflow mechanics. |
