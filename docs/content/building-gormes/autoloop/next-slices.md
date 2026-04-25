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
| 3 / 3.E.8 | Operator-auditable search evidence | Goncho/Honcho-compatible search surfaces operator evidence for user scope, source filters, and session lineage on every widened cross-source search hit | operator, system | `cmd/gormes/goncho_search_evidence_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.D | Per-turn model selection | Kernel accepts a per-turn model override, sends it on exactly one hermes.ChatRequest, exposes the active model in RenderFrame during that turn, and falls back to the resident config model on the next turn | operator, gateway, system | `internal/kernel/per_turn_model_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 7 / 7.E | BlueBubbles iMessage bubble formatting parity | BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes | gateway, system | `internal/channels/bluebubbles/bot_test.go` | Unblocks BlueBubbles iMessage session-context prompt guidance. |
<!-- PROGRESS:END -->
