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

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 5 / 5.G | Gormes-native MCP host runtime boundary | Gormes exposes a native MCP/tool host boundary with explicit tool declarations, filtering, audit evidence, and channel/runtime-safe execution without adopting a non-Hermes config surface. | gateway, operator, system | `internal/tools/mcp_host_boundary_test.go` | Unblocks MCP stdio transport + tool/list discovery, Managed tool gateway bridge, Tool output budget persisted artifact pointer. |
<!-- PROGRESS:END -->
