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
| 4 / 4.B | Gormes-owned session tree navigator over lineage and labels | Add a native `/tree` session navigator that projects Gormes' existing session lineage, fork, compression, and title metadata into an in-place tree view with search/filter modes and operator labels. Selecting a prior user turn should restore that prompt for editing when safe; selecting non-user entries should switch the visible leaf or report why the stored transcript cannot be replayed. The implementation must use Gormes session stores and lineage tables, not Pi JSONL files. | operator, system | `internal/tui/tree_selector_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.L | Per-file mutation queue for native write edit and patch tools | Serialize concurrent file mutations that target the same canonical path across native write, edit, patch, and custom file-task tools while preserving parallel execution for independent files. The queue must resolve symlink aliases for existing files, use cleaned absolute paths for new files, cover the full read-modify-write window, and compose with the existing file staleness registry and atomic writer helpers. | operator, system | `internal/tools/file_mutation_queue_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.Q | Gormes JSONL RPC mode over agent runtime events | Expose a local `gormes` JSONL RPC run mode for language-agnostic embedding. The protocol should accept prompt, steer, follow_up, abort, get_state, get_messages, session stats, model/thinking controls where existing runtime seams support them, and stream agent/tool/queue/compaction events as newline-delimited JSON with strict LF framing. It should reuse Gormes kernel/API-server event models and must not require a web server, Pi subprocess, or live provider in tests. | operator, system | `cmd/gormes/rpc_mode_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.C | Hermes integrations claim audit + source-backed plugin/skill parity map | Turn the sanitized Reddit/WebAfterAI Hermes integrations post into a source-backed parity map without accepting marketing shorthand as fact: classify each named integration as first-party bundled skill, bundled plugin/backend, gateway/platform/tool, optional skill, indirect web/browser/MCP/scraping workflow, Gormes-owned candidate, or unsupported/excluded claim. The audit must explicitly handle cases where a workflow is achievable through generic web scraping, browser automation, MCP, or Firecrawl-style extraction without being a direct Hermes plugin or tool, and it must not create implementation rows for Reddit, Stripe, InsForge, Graphiti/Zep, or Fireflies unless exact current Hermes source appears. | operator, system | `webpages/docs/content/building-gormes/architecture_plan/hermes-integrations-claim-audit.md` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.D | Gormes-owned TUI queued-message widget and busy delivery modes | Adapt Pi's visible steering/follow-up queue pattern into the native Gormes Bubble Tea chat TUI without changing Hermes-compatible slash command semantics. While a turn is active, plain Enter should honor the configured busy-input mode, queued or steering drafts should be visible in the bottom-pinned chrome, queued entries should drain after the kernel returns idle, and the UI must keep Alt/Shift+Enter newline behavior intact. | operator, system | `internal/tui/queued_messages_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.D | Gormes-owned TUI extension status widget and footer seam | Introduce a small Go-native TUI extension context that lets trusted in-process Gormes extensions add or clear footer status entries, widgets above or below the editor, and working-indicator text/frames. The seam should be typed, width-safe, scoped to the active session, and degrade to no-op evidence in non-interactive modes; it must not execute TypeScript or import Pi packages. | operator, system | `internal/tui/status_bar_ext_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
