---
title: "Skill Builder Control Plane"
weight: 30
aliases:
  - /building-gormes/autoloop/
---

# Skill Builder Control Plane

The old autonomous loop binaries have been removed. Skill-driven planner and
builder passes now execute the building-gormes roadmap by selecting work from
`progress.json`, running tests locally, and feeding evidence back into the
canonical row.

These pages mirror the structured rows in
`docs/content/building-gormes/architecture_plan/progress.json` so operators,
contributors, and agents use the same queue.

Before assigning work, read the [Completion Plan](../architecture_plan/completion-plan/)
and [Agent Operating Model](../architecture_plan/agent-operating-model/).
The skill builder workflow exists to finish Hermes-in-Go parity, not to
maximize worker count or churn through vague rows.

## Selection Order

1. Validate the roadmap with `go run ./cmd/progress validate`.
2. Read the [Completion Plan](../architecture_plan/completion-plan/) and
   [Completion Lane Roadmap](../architecture_plan/lane-roadmap/).
3. Pick one row from [Agent Queue](agent-queue/) or from a user-named row.
4. Reject umbrella, blocked, or testless rows unless the row explicitly has
   `no_test_required`.
5. Use `gormes-builder` and `gormes-tdd-slice` for implementation.
6. Update evidence only for the selected row.

## Start Here

- [Skill Builder Handoff](builder-loop-handoff/) explains the shared skill
  entrypoint, queue source, generated docs, tests, and candidate policy.
- [Agent Queue](agent-queue/) lists rows that are ready for focused
  implementation.
- [Next Slices](next-slices/) shows the short ranking of high-leverage
  work.
- [Blocked Slices](blocked-slices/) keeps blocked rows visible without
  making them assignable.
- [Umbrella Cleanup](umbrella-cleanup/) lists broad rows that need to be
  split before assignment.
- [Progress Schema](progress-schema/) defines the row fields builder skills
  expect.
