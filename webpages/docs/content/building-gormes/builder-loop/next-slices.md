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
| 5 / 5.N | Hermes toolset distribution manifest and deterministic sampler | Port Hermes' toolset_distributions.py contract as a hermetic Go manifest and deterministic sampler: expose the named distribution definitions, descriptions, and percentage weights; validate referenced toolsets through the existing Gormes toolset catalog; sample each toolset independently from an injectable RNG; and guarantee the highest-probability valid toolset is selected when all rolls miss. This row does not run batch/datagen jobs or change operator toolset config persistence. | operator, system | `internal/platform/cli/toolsets/distribution_test.go with deterministic RNG fixtures for default, image_gen, safe, terminal_only, unknown distribution, invalid toolset skip, and highest-probability fallback cases` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
