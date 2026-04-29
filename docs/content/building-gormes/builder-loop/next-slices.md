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
| 2 / 2.F.5 | Mid-run steer injection between tool calls | Gateway /steer guidance can be delivered into an in-flight native Gormes turn after the current tool batch, preserving provider message-role alternation by appending a clear user-guidance marker to the last tool-result message before the next provider request; no Telegram-only path, hermes-agent runtime call, or next-turn duplicate is introduced. | operator, gateway, system | `internal/kernel/tool_interrupt_test.go and internal/gateway/steer_queue_test.go` | P0 handoff; needs contract proof before closeout. |
<!-- PROGRESS:END -->
