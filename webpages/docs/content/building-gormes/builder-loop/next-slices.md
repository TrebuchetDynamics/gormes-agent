---
title: "Next Slices"
weight: 30
aliases:
  - /building-gormes/next-slices/
---

# Next Slices

This page is generated from the canonical progress file and lists the highest
leverage contract-bearing roadmap rows to execute next.

The ordering is:

1. unblocked `P0` handoffs;
2. active `in_progress` rows;
3. `fixture_ready` rows;
4. unblocked rows that unblock other slices;
5. remaining `draft` contract rows.

Use this page when choosing implementation work. If a row is too broad, split
the row in `progress.json` before assigning it.

If no slices are listed, the next correct action is planner work: choose one
planned row from `progress.json` or a phase page and add enough contract detail
for it to appear here. Do not infer that an empty generated list means the
roadmap is complete.

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.F | Backlog split C1: lossless multi-file loader/writer behind the single-file API | Child 1 of the module-split umbrella — the smallest NON-behavior-changing first step. In internal/progress, add the ability to load AND write a split layout (a directory of per-module files, or index + per-module files) BEHIND the existing single-file public API: internal/progress.Load(path) (progress.go:245) must transparently accept EITHER the monolithic progress.json OR the split layout and return the identical in-memory model; add a round-trip pair (e.g. `go run ./cmd/progress split` / `... merge`, or internal Split()/Merge()) that is BYTE-STABLE through the existing stable marshal (internal/progress/progress_marshal.go) — merge(split(x)) == x and validate output identical. Do NOT move any real rows, do NOT change any consumer (cmd/progress, plannerloop, builderloop, status, docs/landing generators), do NOT change validate semantics. This is purely a back-compat shim + a lossless round-trip proven by tests, so a later child can flip the on-disk layout with zero behavior change. Owned Gormes infra. | system | `internal/progress/split_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
