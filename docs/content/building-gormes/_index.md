---
title: "Building Gormes"
weight: 200
---

# Building Gormes

Contributor-facing documentation for the Go runtime, roadmap, skill-driven work
queue, and upstream-porting research. If you want to **use** Gormes, start at
[Using Gormes](../using-gormes/).

## Runtime thesis

**Gormes is the production runtime for self-improving agents.** Four core systems live inside the binary:

- **Learning Loop** — detect complex tasks, distill reusable skills, improve them over time ([Phase 6](./architecture_plan/phase-6-learning-loop/)).
- **Memory** — SQLite + FTS5 + ontological graph, with a human-readable USER.md mirror ([Phase 3](./architecture_plan/phase-3-memory/)).
- **Tool Execution** — typed Go interfaces, in-process registry, no Python bounce ([Phase 2.A](./architecture_plan/phase-2-gateway/)).
- **Gateway** — one runtime, many interfaces: TUI plus shipped Telegram/Discord, with Slack and long-tail adapters advancing as contract-first Phase 2 slices ([Phase 2.B](./architecture_plan/phase-2-gateway/)).

Gormes ports upstream contracts, not upstream monoliths. The default rule is
Hermes parity for almost every operator-visible surface: config, command tree,
slash commands, gateway handlers, provider routing/auth/usage, tool
continuations, plugins, skills, cron, status, TUI/API behavior, packaging, and
recovery. Go-native divergence is allowed only when it is explicit, tested, and
visible to operators. GBrain proves the value of contract-first operations,
durable jobs, graph provenance, and skills as auditable runtime knowledge.
Gormes absorbs those durable contracts into a small Go runtime instead of
copying Python mega-files or TypeScript database gravity.

## Section map

| Need | Start with | Then use |
|---|---|---|
| Understand the finish line | [Gormes Completion Plan](./architecture_plan/completion-plan/) | [Hermes/Honcho To Gormes Go Runtime Plan](./architecture_plan/hermes-honcho-go-runtime-plan/), [Hermes And Honcho Feature Map](./architecture_plan/hermes-honcho-feature-map/), [Upstream Coverage Ledger](./architecture_plan/upstream-coverage-ledger/), [Swarm Feature Parity Audit](./architecture_plan/swarm-feature-parity-audit/), [Completion Lane Roadmap](./architecture_plan/lane-roadmap/), [Agent Operating Model](./architecture_plan/agent-operating-model/) |
| Understand the runtime shape | [Core Systems](./core-systems/) | [Architecture Plan](./architecture_plan/), [Why Go](./architecture_plan/why-go/) |
| Choose implementation work | [Agent Queue](./builder-loop/agent-queue/) | [Next Slices](./builder-loop/next-slices/), [Blocked Slices](./builder-loop/blocked-slices/), [Umbrella Cleanup](./builder-loop/umbrella-cleanup/) |
| Prepare a skill-builder handoff | [Contract Readiness](./contract-readiness/) | [Progress Schema](./builder-loop/progress-schema/), [Skill Builder Handoff](./builder-loop/builder-loop-handoff/) |
| Port an upstream subsystem | [Hermes/Honcho To Gormes Go Runtime Plan](./architecture_plan/hermes-honcho-go-runtime-plan/) | [Hermes And Honcho Feature Map](./architecture_plan/hermes-honcho-feature-map/), [Upstream Coverage Ledger](./architecture_plan/upstream-coverage-ledger/), [Swarm Feature Parity Audit](./architecture_plan/swarm-feature-parity-audit/), [Porting a Subsystem](./porting-a-subsystem/), [Upstream Lessons](./upstream-lessons/), [Testing](./testing/) |
| Reuse gateway adapter ideas | [Gateway Donor Map](./gateway-donor-map/) | [Shared Adapter Patterns](./gateway-donor-map/shared-adapter-patterns/), then the channel dossier |
| Find a Go donor for a non-gateway subsystem (runtime, tools, memory, utilities) before writing code | `gormes-references` skill (`docs/development-skills/gormes-references/SKILL.md`) | `references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md` for provider/auth/streaming, then `gormes-tdd-slice` |
| Continue Goncho/Honcho memory work | [Goncho Honcho Memory](./goncho_honcho_memory/) | [Prompts](./goncho_honcho_memory/01-prompts/), [Tool Schemas](./goncho_honcho_memory/02-tool-schemas/) |

## Planning rules

Every subsystem plan answers four questions before implementation:

1. What upstream contract are we porting?
2. Which trust class can call it: operator, gateway, child-agent, or system?
3. How does degraded mode show up in `gormes doctor`, status, audit, or logs?
4. What fixture proves compatibility without a live provider or platform?

For builder agents, the canonical progress row must also name the
`execution_owner`, `slice_size`, `ready_when`, `not_ready_when`, `write_scope`,
`test_commands`, and `done_signal` conditions. Assignable slices are
small/medium/large rows; `umbrella` rows stay inventory until split.

## Skill-driven execution contract

Repo-local skills are the executor for this roadmap, not a separate backlog.
`gormes-builder` reads
`docs/content/building-gormes/architecture_plan/progress.json`, uses the
generated `docs/content/building-gormes/` pages as the human-readable handoff
surface, selects one eligible row, and develops the full `gormes-agent` toward
the architecture plan with tests.

The building-gormes docs are therefore part of the control plane. When a phase,
subphase, or task is unclear to an agent, fix the canonical progress row and
regenerate the derived docs instead of adding private instructions elsewhere.
Skills consume the same source of truth contributors read: progress rows for
selection, generated pages for operator review, and row metadata for handoff
prompts.

The old `cmd/planner-loop` and `cmd/builder-loop` executables are removed.
Use `go run ./cmd/progress validate` and `go run ./cmd/progress write` for
progress maintenance, and use `go run ./cmd/repoctl ...` for repo metadata.

## Current queue rule

The generated [Agent Queue](./builder-loop/agent-queue/) and
[Next Slices](./builder-loop/next-slices/) can be empty even while
`progress.json` still contains planned rows. Empty means no unblocked,
non-umbrella row currently satisfies the builder handoff contract; it does not
mean Gormes is done.

When the queue is empty, run a `gormes-planner` pass against the relevant phase
or subsystem. The planner should sharpen a planned/draft row until it has
source refs, write scope, acceptance, degraded mode, and test commands. Only
then should `gormes-builder` and `gormes-tdd-slice` implement it. Historical
audits and parity matrices are evidence surfaces; `progress.json` is the
dispatch queue.

## Contributor path

Use the planning docs in this order:

1. Read [Upstream Lessons](./upstream-lessons/) to understand which contracts
   Gormes absorbs from Hermes and GBrain.
2. Check [Skill Builder Handoff](./builder-loop/builder-loop-handoff/) for the
   skill entrypoint, candidate source, generated docs, tests, and candidate
   policy.
3. Pick work from [Agent Queue](./builder-loop/agent-queue/) for a
   builder-ready handoff, then use [Next Slices](./builder-loop/next-slices/)
   for the shorter ranking.
4. Use `docs/development-skills/gormes-builder/SKILL.md` and
   `docs/development-skills/gormes-tdd-slice/SKILL.md` to implement one row;
   `.agents/skills/`, `.claude/skills/`, and `.codex/skills/` are symlink
   loader views.
5. Check [Contract Readiness](./contract-readiness/) before implementation; an
   active or P0 row must name its contract, trust class, degraded mode, fixture,
   source references, and acceptance checks.
6. Check [Blocked Slices](./builder-loop/blocked-slices/) and
   [Umbrella Cleanup](./builder-loop/umbrella-cleanup/) before assigning a row.
7. Use [Progress Schema](./builder-loop/progress-schema/) when editing canonical progress.
8. Write the spec/plan from [Porting a Subsystem](./porting-a-subsystem/),
   then implement with the fixture classes in [Testing](./testing/).

## Reference groups

**Architecture:** [Architecture Plan](./architecture_plan/), [Hermes/Honcho To Gormes Go Runtime Plan](./architecture_plan/hermes-honcho-go-runtime-plan/), [Hermes And Honcho Feature Map](./architecture_plan/hermes-honcho-feature-map/), [Upstream Coverage Ledger](./architecture_plan/upstream-coverage-ledger/), [Swarm Feature Parity Audit](./architecture_plan/swarm-feature-parity-audit/), [Core Systems](./core-systems/), [What Hermes Gets Wrong](./what-hermes-gets-wrong/), [Upstream Lessons](./upstream-lessons/).

**Execution queue:** [Contract Readiness](./contract-readiness/), [Skill Builder Handoff](./builder-loop/builder-loop-handoff/), [Agent Queue](./builder-loop/agent-queue/), [Next Slices](./builder-loop/next-slices/), [Blocked Slices](./builder-loop/blocked-slices/), [Umbrella Cleanup](./builder-loop/umbrella-cleanup/), [Progress Schema](./builder-loop/progress-schema/).

**Implementation help:** [Porting a Subsystem](./porting-a-subsystem/), [Testing](./testing/), [Gateway Donor Map](./gateway-donor-map/), [Goncho Honcho Memory](./goncho_honcho_memory/).
