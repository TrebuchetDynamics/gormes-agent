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
| 5 / 5.V | Goncho honcho_reasoning LLM-backed synthesis | HonchoReasoningTool must call a real LLM for synthesis (matching Python Honcho's .chat() multi-pass dialectic) instead of deterministic string assembly. Requires adding LLMCaller interface to goncho.Service and threading through kernel/config. Fall back to deterministic with degraded evidence when no LLM caller configured. | operator, system | `internal/gonchotools/honcho_tools_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.V | Web dashboard server shell + degraded inventory | Create the Go web dashboard server shell that matches Hermes' FastAPI dashboard startup, static SPA serving, localhost security guard, public API allowlist, and degraded inventory semantics before any React feature work lands. | operator, system | `internal/apiserver/dashboard_server_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.V | Web dashboard React/Vite scaffold + 9-page route manifest | Port the minimal Vite/React scaffold and route manifest for Hermes' 9-page dashboard without filling each page's detailed behavior yet. | operator, system | `web/src/App.test.tsx` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.F | Skill registries | Native skills hub registry providers expose source-backed, read-only metadata for the current Hermes skills hub source adapters before any write-capable install flow: OptionalSkillSource, HermesIndexSource, SkillsShSource, WellKnownSkillSource, UrlSource, GitHubSource, ClawHubSource, ClaudeMarketplaceSource, and LobeHubSource are the active upstream contract at Hermes 69d4800d. This executable slice should add only the missing remote registry read-model providers over the existing HubRegistryProvider/Search seam, with source filtering, trust normalization, centralized-index preference, stale-cache fallback, and typed degraded evidence for unavailable, malformed, timeout, empty, and rate-limited upstreams. Url direct parsing and optional bundled-skill inventory stay in their existing rows; this slice must not install, activate, quarantine, guard-scan, or mutate skills. | operator, system | `internal/skills/hub_registry_sources_test.go` | Unblocks Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC, Skill registries unavailable-network UX fixtures. |
| 5 / 5.V | Web dashboard core components + data-state fixtures | Port Hermes dashboard's reusable component set as typed React fixtures so later page slices compose parity components instead of re-inventing UI behavior per page. | operator, system | `web/src/components/dashboard-components.test.tsx` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.V | Web dashboard PTY chat + event websocket fixtures | Port the dashboard embedded-chat/PTY websocket contract and React chat integration with fakes before binding to a live terminal or provider runtime. | operator, system | `internal/apiserver/dashboard_pty_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.V | Web dashboard theme catalog + switcher parity | Port Hermes dashboard theme catalog and theme-selection API as a small independent slice, separate from TUI skin parity and page functionality. | operator, system | `internal/apiserver/dashboard_theme_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.V | Web dashboard OAuth provider flows + EN/ZH i18n | Port the dashboard OAuth-provider UI contract and EN/ZH i18n catalog as fixture-backed surfaces; do not implement real browser OAuth beyond fake-provider endpoint shapes in this slice. | operator, system | `internal/apiserver/dashboard_oauth_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
