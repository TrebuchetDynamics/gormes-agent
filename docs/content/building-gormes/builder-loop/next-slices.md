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
| 5 / 5.A | Tool descriptor layer (OperationSpec) | Every tool in the registry carries a declarative descriptor (OperationSpec) that generates model schemas, CLI commands, gateway slash commands, doctor checks, and audit taxonomy from one source | operator, gateway, child-agent, system | `internal/tools/operation_spec_test.go` | P0 handoff; needs contract proof before closeout. |
| 6 / 6.I | Regex-based auto-link extraction + brain-first lookup | Markdown links, wikilinks, qualified wikilinks auto-extracted; typed inference; brain-first 5-step lookup | operator, system | `internal/goncho/auto_link_test.go` | Unblocks Compiled truth pattern, Tiered enrichment. |
| 6 / 6.H | SKILL.md metadata.when/loaded/placement schema | YAML frontmatter supports metadata.when (conditional activation), metadata.loaded (auto-load), metadata.placement (system/onscreen/admin); hierarchical routing | operator, system | `internal/skills/metadata_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
