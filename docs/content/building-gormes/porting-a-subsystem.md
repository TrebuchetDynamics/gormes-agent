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

Open [Subsystem Inventory](../architecture_plan/subsystem-inventory/). Every row is a Hermes subsystem with a target Gormes sub-phase. Pick one that:

- Carries a ⏳ planned status (not already shipped)
- Has no hard dependency on a later phase (check the "Target phase" column)
- You have context on (voice/vision are big lifts; a platform adapter is a reasonable first PR)

## 2. Do the source-study checklist

Before writing implementation tasks, answer these in the spec or plan:

1. **Contract:** what upstream behavior is being ported? Name the source files
   and the external contract, not just the donor implementation.
2. **Trust class:** who can call it: `operator`, `gateway`, `child-agent`, or
   `system`? What is rejected before handler code runs?
3. **Fixture:** what replayable fixture proves compatibility without live
   credentials, live platforms, or a real provider?
4. **Degraded mode:** how does partial capability show up in status, doctor,
   audit, or logs?
5. **Boundary:** what stays out of the kernel, gateway adapter, or trusted
   plugin surface?

Useful donor study pages:

- [Upstream Hermes Source Study](../../upstream-hermes/source-study/)
- [Upstream GBrain Architecture](../../upstream-gbrain/architecture/)
- [Upstream Lessons](../upstream-lessons/)

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
| Gormes target | Package, command, tool, API, or doc surface. |
| Public contract | What the operator, gateway, tool caller, or kernel observes. |
| Degraded mode | How unavailable/partial behavior is reported. |
| Fixture | Local testdata, fake provider, SQLite fixture, or transcript. |
| Write scope | Narrow list of paths the builder may edit. |
| Gate | Row-local test plus focused package gate. |
| Done signal | Exact evidence to report when complete. |

## 4. Write a spec or plan when needed

For broad or high-risk subsystems, write:

- `docs/superpowers/specs/YYYY-MM-DD-<subsystem>-design.md`
- `docs/superpowers/plans/YYYY-MM-DD-<subsystem>.md`

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
