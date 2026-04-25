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
| 2 / 2.F.3 | Pairing read-model schema + atomic persistence | Gateway pairing state is persisted as a Go-native XDG read model with atomic writes, deterministic pending/approved ordering, and per-platform paired/unpaired status before approval UX | gateway, operator | `internal/gateway/pairing_store_test.go` | Unblocks Pairing approval + rate-limit semantics, `gormes gateway status` read-only command. |
| 3 / 3.F | Goncho dreaming scheduler contract | Goncho models Honcho dream scheduling as local auditable work intent with cooldown, idle, threshold, and dedupe gates before any LLM dream execution | operator, system | `internal/goncho/dream_scheduler_test.go` | Unblocks Goncho dream execution worker, Goncho dream status documentation. |
| 7 / 7.E | BlueBubbles iMessage bubble formatting parity | BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes | gateway, system | `internal/channels/bluebubbles/bot_test.go` | Unblocks BlueBubbles iMessage session-context prompt guidance. |
<!-- PROGRESS:END -->
