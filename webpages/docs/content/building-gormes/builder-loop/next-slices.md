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
| 1 / 1.E | Shared Bubble Tea wizard step chassis under internal/tui/wizard | internal/tui/wizard exposes a small Wizard interface (Run(ctx, steps...) (Result, error)) that drives a sequence of Step values — Text, MultiLine, Password, Pick (single-select), Confirm — under a Bubble Tea program. The chassis owns: (a) TTY detection (refuse to start when stdin is not a terminal, return a typed ErrRequiresTTY so callers emit *_requires_tty evidence), (b) bypass-when-fully-specified (callers compose 'if all inputs already supplied via flags, do not run the wizard'), (c) Ctrl-C / escape returning ErrAbort, (d) golden-snapshot testability via charmbracelet/x/exp/teatest. The chassis must not import any cmd/gormes package; admin-TUI screens (1.E.3+) compose it from their screen models, and stand-alone command callers can compose it independently if needed. | operator | `internal/tui/wizard/wizard_test.go teatest scripts` | Unblocks Unified admin TUI shell with tab navigation, Admin TUI: Setup health screen with missing-config callouts, Admin TUI: Chat tab with keybinding to jump in from any screen, Admin TUI: Agents screen wired to the 2.H dynamic registry. |
| 8 / 8.A | TD engineering blog scaffolded and live | TrebuchetDynamics has a publicly reachable engineering blog with a working Atom/RSS feed, an /about page that names the org and the methodology, and a deploy pipeline so a markdown commit becomes a published post without manual intervention. Hosting choice is owner's call (Astro/Hugo/Eleventy + Cloudflare/Vercel/GitHub Pages); the row is done when a stranger can subscribe to a feed and read one published post. | operator | `webpages/blog/ (or chosen blog repo path)` | Unblocks Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline. |
| 8 / 8.E | Agentic-porting-kit repo scaffold | The gormes-* skill set (gormes-planner, gormes-builder, gormes-tdd-slice, gormes-parity-auditor, gormes-references, gormes-skill-manager) is extracted into a separate public TrebuchetDynamics repo (`agentic-porting-kit` or equivalent), with a README that frames the kit as a generic Python→Go porting toolkit, a worked example using a small non-Hermes target, and a clear license. The kit must work standalone — its rows must be loadable by Codex or Claude Code in any repo, not just Gormes. | operator | `(separate repo: TrebuchetDynamics/agentic-porting-kit)` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
