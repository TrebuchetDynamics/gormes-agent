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
| 8 / 8.F | Internal session search tool package rehome | Move the model-facing session_search adapter from internal/sessionsearchtool to internal/tools/sessionsearch while preserving descriptor text, JSON schema, argument normalization, memory/session catalog execution, degraded evidence, and cmd/gormes registry wiring. This is the next behavior-preserving Tool Adapter Enclave package move after compact and trace. | - | `internal/tools/sessionsearch/session_search_tool_schema_test.go and internal/tools/sessionsearch/session_search_tool_execution_test.go after the move, plus cmd/gormes registry tests.` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
