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
| 5 / 5.O | Hermes auth priority credential reordering | Implement `gormes auth priority <provider> <target> <priority>` for an existing credential pool. Resolve target by 1-based displayed index, entry ID, or unique case-insensitive exact label; reject missing or ambiguous targets without writing. Move the selected credential to the requested zero-based position, clamp to the pool bounds, persist contiguous priorities atomically, preserve every non-priority field including cooldown state, and reproduce Hermes's Anthropic manual-before-seeded ordering. Report the effective priority and the upstream clamp/provider-strategy notes without exposing credential values. This row excludes `auth add --priority` and unrelated auth/documentation parity work. | operator | `internal/platform/cli/gormescli/modules/providers/auth_command_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
