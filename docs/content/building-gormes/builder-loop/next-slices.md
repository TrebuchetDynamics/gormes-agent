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
| 2 / 2.F.4 | Channel-neutral native runtime turn adapter | Telegram, Slack, Discord, WhatsApp, BlueBubbles, and future channels enter the same native Gormes turn adapter so provider/runtime fixes preserve Hermes channel parity instead of hard-coding Telegram behavior. | gateway, operator, system | `internal/gateway/channel_neutral_turn_adapter_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.A | Tool output budget persisted artifact pointer | Native tool execution bounds large tool results by persisting full output as a session artifact and returning a short text pointer to the model/channel, preserving Hermes operator readability and channel safety. | gateway, operator, system | `internal/tools/result_budget_test.go` | Unblocks 61-tool registry port, Native runtime provider gateway binding, MCP stdio transport + tool/list discovery. |
| 5 / 5.G | Gormes-native MCP host runtime boundary | Gormes exposes a native MCP/tool host boundary with explicit tool declarations, filtering, audit evidence, and channel/runtime-safe execution without adopting a non-Hermes config surface. | gateway, operator, system | `internal/tools/mcp_host_boundary_test.go` | Unblocks MCP stdio transport + tool/list discovery, Managed tool gateway bridge, Tool output budget persisted artifact pointer. |
| 5 / 5.N | Goncho serialized write queue + relation candidates | Goncho serializes memory/conclusion writes and records pending relation candidates for possible conflicts or supersession without blocking the originating memory write. | operator, system | `internal/goncho/write_queue_relation_test.go` | Unblocks Goncho memory integration into normal agent turn, Goncho operator diagnostics contract. |
<!-- PROGRESS:END -->
