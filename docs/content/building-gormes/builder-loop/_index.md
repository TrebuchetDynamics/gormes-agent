---
title: "Builder Loop Control Plane"
weight: 30
aliases:
  - /building-gormes/autoloop/
---

# Builder Loop Control Plane

The builder loop is the unattended execution side of the
Planner-Builder Loop architecture (see `AGENTS.md` at the repo root). It executes the
building-gormes roadmap by selecting work from `progress.json`, running
backend workers in isolated worktrees, and feeding promotion outcomes back
into `runs.jsonl` for the planner loop to read.

These pages mirror the structured rows in
`docs/content/building-gormes/architecture_plan/progress.json` so operators,
contributors, and worker agents use the same queue.

## Start Here

- [Builder Loop Handoff](builder-loop-handoff/) explains the shared
  entrypoint, queue source, generated docs, tests, and candidate policy.
- [Agent Queue](agent-queue/) lists rows that are ready for autonomous
  worker execution.
- [Next Slices](next-slices/) shows the short ranking of high-leverage
  work.
- [Blocked Slices](blocked-slices/) keeps blocked rows visible without
  making them assignable.
- [Umbrella Cleanup](umbrella-cleanup/) lists broad rows that need to be
  split before assignment.
- [Progress Schema](progress-schema/) defines the row fields the builder
  loop expects.
