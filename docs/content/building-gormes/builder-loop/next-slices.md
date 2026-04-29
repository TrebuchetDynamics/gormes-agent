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
| 4 / 4.C | Context-file discovery + injection scan | Native prompt assembly exposes a pure context-file discovery helper that mirrors Hermes project-context precedence: load SOUL.md from the Hermes/Gormes profile unless skipped, then load exactly one project context source in order .hermes.md/HERMES.md walking up to git root, AGENTS.md/agents.md in cwd, CLAUDE.md/claude.md in cwd, then .cursorrules plus sorted .cursor/rules/*.mdc in cwd. Each loaded source is scanned for Hermes-compatible injection/invisible-character patterns and head/tail truncated to the context-file budget before being rendered into a deterministic prompt block. | system, operator | `internal/hermes/context_files_test.go::TestContextFiles*` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
