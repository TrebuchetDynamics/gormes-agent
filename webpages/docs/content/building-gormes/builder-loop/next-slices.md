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
| 5 / 5.M | Hermes Kanban slash/gateway/dashboard surfaces | Gormes ports the operator surfaces around Kanban: /kanban routes in TUI/gateway use the same parser/output as gormes kanban, gateway status exposes dispatcher state and nudge capability, and the dashboard shows live Kanban tasks, lanes, filters, worker runs, and dispatcher nudges over authenticated Gormes dashboard routes. | operator, gateway, system | `internal/apiserver/dashboard_kanban_test.go; internal/gateway/kanban_command.go; internal/kanbantools/kanban_tools_test.go` | Already active; contract metadata keeps execution bounded. |
| 8 / 8.A | TD engineering blog scaffolded and live | TrebuchetDynamics has a publicly reachable engineering blog with a working Atom/RSS feed, an /about page that names the org and the methodology, and a deploy pipeline so a markdown commit becomes a published post without manual intervention. Hosting choice is owner's call (Astro/Hugo/Eleventy + Cloudflare/Vercel/GitHub Pages); the row is done when a stranger can subscribe to a feed and read one published post. | operator | `webpages/blog/ (or chosen blog repo path)` | Unblocks Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline. |
| 6 / 6.K | Behavioral pattern extraction from session logs | Mine session logs and tool execution audits for behavioral patterns: which tool sequences succeed vs fail, which reasoning patterns precede good outcomes, which response styles correlate with user satisfaction. Patterns feed into the self-evolution loop as candidate mutations. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.D | Release archive 30 MB size gate | The release workflow enforces the v1 differentiator's single 30 MB binary promise before release artifacts are uploaded: every built archive is checked against a 31,457,280-byte cap after packaging and checksum generation, and any oversize archive fails the build with a clear target-specific message. | operator, system | `webpages/docs/install/release_workflow_test.go::TestReleaseWorkflowEnforcesMaxArchiveSize` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.E | Agentic-porting-kit repo scaffold | The gormes-* skill set (gormes-planner, gormes-builder, gormes-tdd-slice, gormes-parity-auditor, gormes-references, gormes-skill-manager) is extracted into a separate public TrebuchetDynamics repo (`agentic-porting-kit` or equivalent), with a README that frames the kit as a generic Python→Go porting toolkit, a worked example using a small non-Hermes target, and a clear license. The kit must work standalone — its rows must be loadable by Codex or Claude Code in any repo, not just Gormes. | operator | `(separate repo: TrebuchetDynamics/agentic-porting-kit)` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.G | Built-with-Gormes page scaffold | A page at gormes.ai/built-with (or equivalent path on the docs site) lists real production deployments of Gormes, even if there is initially only one entry (the operator's own). The page has a documented submission process (PR-based) and a template entry shape. The point is to make the slot exist so it can be filled, not to fake usage. | operator | `webpages/landing/src/pages/built-with.astro (or equivalent)` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
