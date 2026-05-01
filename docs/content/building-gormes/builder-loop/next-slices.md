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
| 5 / 5.V | Web dashboard core components + data-state fixtures | Port Hermes dashboard's reusable component set as typed React fixtures so later page slices compose parity components instead of re-inventing UI behavior per page. | operator, system | `web/src/components/dashboard-components.test.tsx` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.V | Web dashboard PTY chat + event websocket fixtures | Port the dashboard embedded-chat/PTY websocket contract and React chat integration with fakes before binding to a live terminal or provider runtime. | operator, system | `internal/apiserver/dashboard_pty_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.V | Web dashboard theme catalog + switcher parity | Port Hermes dashboard theme catalog and theme-selection API as a small independent slice, separate from TUI skin parity and page functionality. | operator, system | `internal/apiserver/dashboard_theme_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.V | Web dashboard OAuth provider flows + EN/ZH i18n | Port the dashboard OAuth-provider UI contract and EN/ZH i18n catalog as fixture-backed surfaces; do not implement real browser OAuth beyond fake-provider endpoint shapes in this slice. | operator, system | `internal/apiserver/dashboard_oauth_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
