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
| 7 / 7.E | BlueBubbles iMessage bubble formatting parity | BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes | gateway, system | `internal/channels/bluebubbles/bot_test.go` | Unblocks BlueBubbles iMessage session-context prompt guidance. |
| 5 / 5.N | Cron schedule parser + repeat state fixtures | internal/cron adds a pure Hermes-compatible schedule parser and repeat-state read model before any public cronjob tool handler is exposed. ParseCronSchedule(input string, now time.Time) returns a typed ParsedSchedule for one-shot durations (`30m`, `2h`, `1d`), recurring intervals (`every 30m`, `every 2h`), 5-field cron expressions, and ISO timestamps. CronNextRunDecision(parsed, lastRunUnix, repeatCompleted, now) reports whether a one-shot is still recoverable inside the 120s grace window, whether recurring jobs should fast-forward stale next-run times, and whether finite repeat counts are exhausted. | operator, system | `internal/cron/schedule_parser_test.go` | Unblocks Cron prompt/script safety + pre-run script contract, Cronjob tool action envelope over native store. |
<!-- PROGRESS:END -->
