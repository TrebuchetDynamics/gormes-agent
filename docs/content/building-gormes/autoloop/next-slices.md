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
| 2 / 2.E.3 | Durable pause/resume intent contract | Gormes durable jobs record pause and resume as explicit operator/system control intents over the SQLite ledger without starting a GBrain worker daemon | operator, system | `internal/subagent/durable_lifecycle_test.go` | Unblocks Durable replay and inbox message contract. |
| 3 / 3.E.7 | SillyTavern persona and group-chat mapping fixtures | Goncho maps Honcho SillyTavern peer modes, session naming, enrichment modes, and group-chat participants without widening recall or leaking the internal goncho name | operator, system | `internal/goncho/sillytavern_mapping_test.go` | Unblocks Cross-chat operator evidence. |
| 5 / 5.I | Plugin SDK | Gormes loads plugin manifests, capability metadata, version constraints, and disabled-state evidence without executing plugin runtime code | operator, system | `internal/plugins/manifest_test.go` | Unblocks Dashboard theme/plugin extension status contract, First-party Spotify plugin fixture. |
| 7 / 7.E | BlueBubbles iMessage bubble formatting parity | BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes | gateway, system | `internal/channels/bluebubbles/bot_test.go` | Unblocks BlueBubbles iMessage session-context prompt guidance. |
<!-- PROGRESS:END -->
