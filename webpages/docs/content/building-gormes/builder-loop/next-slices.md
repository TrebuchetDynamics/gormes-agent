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
| 5 / 5.G | Gormes MCP server enable/disable persistence | Add noninteractive `gormes mcp enable <name>` and `gormes mcp disable <name>` commands that atomically update only an existing active-profile `mcp_servers.<name>.enabled` field, preserve every other server field including tool filters and secret references, emit redacted text or JSON evidence, and state that a runtime reload is required. This slice performs no probe, network request, process launch, secret lookup, or live registry mutation. | operator | `t.TempDir active-profile config.toml with HTTP and stdio entries, synthetic secret-reference/header fields, explicit tool filters, injected MCPConfigPath, and no environment, process, network, OAuth, or live MCP server access` | Unblocks Hermes MCP catalog install/configure lifecycle. |
<!-- PROGRESS:END -->
