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
| 2 / 2.F.4 | Home channel ownership resolver fixtures | Add a channel-neutral home-channel ownership resolver for platform-only delivery targets. The resolver must prefer an explicit target chat/thread, then a Hermes-compatible per-platform home_channel.chat_id/thread/name setting bridged through Gormes config, then a discovery/pairing-owned source only when discovery is explicitly enabled for that platform. It must be callable by delivery routing without Telegram-specific branches and must preserve explicit endpoint/source routing semantics. | operator, gateway | `internal/gateway/home_channel_resolver_test.go with temp config structs and fake SessionSource records only; no live platform SDK or Hermes runtime service.` | P0 handoff; needs contract proof before closeout. |
<!-- PROGRESS:END -->
