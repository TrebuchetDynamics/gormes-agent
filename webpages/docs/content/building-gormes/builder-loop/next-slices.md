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
| 5 / 5.L | Terminal background process registry | Add a bounded in-process registry for background terminal executions. Start with terminal background=true, return a stable process id, list/poll running status and bounded output, terminate by id, and clean up on session end. Reuse the terminal tool's approval and command-safety path; no PTY, WSL-specific, or remote process ownership behavior in this slice. | operator, system | `internal/tools/processregistry/registry_test.go; internal/tools/terminal_tool_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.Q | TUI goal auto-continuation and judge | Finish the in-flight native TUI goal slice. On each idle frame after a goal turn, increment the turn count once, evaluate an injected goal judge, pause at the budget, and submit the continuation prompt exactly once. Paused, done, cleared, cancelled, queued, and failed turns must not continue. The judge fails open on missing or malformed output. | operator, system | `internal/tui/goal/goal_test.go; internal/tui/goal_continuation_test.go; internal/tui/slash_goal_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
