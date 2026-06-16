---
title: "Porting a Subsystem from Upstream"
weight: 40
---

# Porting a Subsystem from Upstream

The contribution path. Use this when you want to port a piece of Hermes into Gormes.

Start with the [Gormes Completion Plan](../architecture_plan/completion-plan/)
and [Completion Lane Roadmap](../architecture_plan/lane-roadmap/).
Every porting pass must route through a repo-local skill:

- `gormes-parity-auditor` to compare upstream behavior against current Gormes.
- `gormes-planner` to create or refine builder-ready rows.
- `gormes-interface-designer` when the Go package/API shape is not obvious.
- `gormes-builder` and `gormes-tdd-slice` to implement one row.

## 1. Pick your target

Start from the current dispatch surfaces, not from memory or an old audit:

1. Check [Agent Queue](../builder-loop/agent-queue/) and
   [Next Slices](../builder-loop/next-slices/) for builder-ready rows.
2. If those generated pages are empty, choose one planned row from
   `progress.json`, [Contract Readiness](../contract-readiness/), or the
   relevant phase page and run a `gormes-planner` pass to make it executable.
3. Use [Subsystem Inventory](../architecture_plan/subsystem-inventory/) and
   [Hermes And Honcho Feature Map](../architecture_plan/hermes-honcho-feature-map/)
   to confirm the upstream subsystem and target phase.

Pick work that:

- Is not already `complete` in `progress.json`.
- Has no unsatisfied `blocked_by` dependency.
- Can be expressed as one contract-bearing row with source refs, write scope,
  test commands, acceptance, degraded mode, and done signal.
- Fits your context and risk tolerance: voice/vision/provider rows are large
  lifts; a focused command, tool, or adapter fixture is a better first row.

Historical audit tables are evidence, not dispatch state. If an audit says a
surface is missing but `progress.json` now marks the row complete/validated,
treat it as historical unless current code or tests prove a regression.

## 2. Do the source-study checklist

Before writing implementation tasks, answer these in the spec or plan:

1. **Contract:** what upstream behavior is being ported? Name the source files
   and the external contract, not just the donor implementation.
2. **Go pattern:** if the implementation shape is unclear, which
   `gormes-references` donor file was read end to end, and what exact pattern
   is being adapted?
3. **Trust class:** who can call it: `operator`, `gateway`, `child-agent`, or
   `system`? What is rejected before handler code runs?
4. **Fixture:** what replayable fixture proves compatibility without live
   credentials, live platforms, or a real provider?
5. **Degraded mode:** how does partial capability show up in status, doctor,
   audit, or logs?
6. **Boundary:** what stays out of the kernel, gateway adapter, or trusted
   plugin surface?

Useful donor study pages:

- [Upstream Hermes Source Study](../../upstream-hermes/source-study/)
- [Upstream Lessons](../upstream-lessons/)
- [Go Donor Reference Map](../architecture_plan/go-donor-reference-map/)

## 2.5. Find a Go donor before writing code

Hermes Python defines the parity contract; Go donors under
`references/go-agent-os/` supply working Go shapes. Skip this step and you will
re-derive an HTTP client, retry loop, OAuth flow, SQLite schema, MCP queue, or
truncation policy that already exists.

Routing:

- **Provider, auth, streaming, quota, retry, OAuth, error classification** →
  use the `gormes-provider-parity` skill (`development-skills/gormes-provider-parity/SKILL.md`)
  plus the donor table in `references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md`.
  GoClaw OAuth/quota and Plandex retry/drift live there as named files.
- **Anything else (runtime, tools, memory, channels, utilities)** → use the
  `gormes-references` skill (`development-skills/gormes-references/SKILL.md`).
  Its "Donor Maps By Subsystem" section names the file to read for each
  problem class (engram store/relations/write_queue, nanobot
  truncate/tokencount/runtime/tools, trpc-agent-go await_user_reply/callbacks,
  axe budget/artifact, etc.).

GoClaw code is permitted in Gormes with provenance (2026-04-29). Other donors
stay patterns-only unless the per-donor permission map in
`references/go-agent-os/README.md` authorizes them. Always add a
`// Adapted from <donor>/...::Symbol` comment on the receiving Gormes file when
porting code.

If no donor fits, record `provenance.origin_type: gormes` on the row and write
the contract from scratch.

## 3. Decide the row shape

Before implementation, the work must exist as one or more rows in
`progress.json`. Specs and plans are useful for larger changes, but they are
not the queue. A row must name the contract, source refs, write scope, tests,
acceptance, ready/not-ready conditions, and done signal.

If the row would require several independent commits, split it before builder
execution. Use tracer-bullet rows that cut through the public interface and
tests end to end.

Use this packet when adding or refining rows:

| Packet field | Required answer |
|---|---|
| Upstream behavior | Exact files, symbols, docs, or commits. |
| Go donor pattern | Exact donor file plus the pattern being adapted, or `none` with rationale. |
| Gormes target | Package, command, tool, API, or doc surface. |
| Public contract | What the operator, gateway, tool caller, or kernel observes. |
| Degraded mode | How unavailable/partial behavior is reported. |
| Fixture | Local testdata, fake provider, SQLite fixture, or transcript. |
| Write scope | Narrow list of paths the builder may edit. |
| Gate | Row-local test plus focused package gate. |
| Done signal | Exact evidence to report when complete. |

## 4. Write a spec or plan when needed

For broad or high-risk subsystems, write:

- `webpages/docs/superpowers/specs/YYYY-MM-DD-<subsystem>-design.md`
- `webpages/docs/superpowers/plans/YYYY-MM-DD-<subsystem>.md`

Use them to explain architecture and dependency order, then copy the actual
execution slices into `progress.json`.

## 5. Implement

Use `gormes-builder` and `gormes-tdd-slice`. Bite-sized commits. Tests first
where feasible. Mirror existing Go package layout under `internal/`, but do
not copy Python monolith boundaries when a deeper Go module hides the same
behavior behind a smaller interface.

## 6. Open a PR

Target `main`. Title convention: `feat(gormes/<subsystem>): port <capability> from upstream`. Reference the spec + plan in the description.

## 7. Update the inventory

Flip your row in [Subsystem Inventory](../architecture_plan/subsystem-inventory/) from ⏳ planned to ✅ shipped, with a link to the shipped spec.
