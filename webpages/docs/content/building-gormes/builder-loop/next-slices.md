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
| 5 / 5.N | Image generation managed-gateway provider binding | Bind the existing image_generate runner/provider registry to the existing ManagedGatewayBridge with hermetic fake HTTP MCP gateway fixtures, so a configured managed image provider can generate the standard redacted image artifact envelope without live FAL/API credentials. | operator, system | `internal/tools/managed_tool_gateway_test.go fake HTTP MCP gateway plus internal/tools/imagegen/generation_test.go artifact-envelope fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
