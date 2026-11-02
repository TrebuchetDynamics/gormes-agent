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
| 4 / 4.I | Advance Hermes checkout and source pairs to latest upstream | Fetch NousResearch/hermes-agent origin/main, advance the in-repo hermes-agent gitlink to that exact SHA, update hermes-source-pairs.json to the same SHA, revalidate every source pair, and regenerate the contract inventory. The planning seed is fbf6c6c91d (17,175 commits ahead of d0288be5b); the builder must use the latest origin/main at execution time. | operator, system | `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json; webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json; webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md` | P0 handoff; needs contract proof before closeout. |
| 4 / 4.I | Rolling upstream baseline freshness gate | Implement the always-upstream freshness step for the strict parity gate. The gate reads the recorded baseline SHA, resolves the latest fetched NousResearch/hermes-agent origin/main, and fails closed when they differ. A matching baseline proceeds to behavior evaluation and emits the source delta from the recorded SHA to origin/main. | operator, system | `internal/planning/progress/repoctl/inventory/hermes_contract_inventory_test.go; fake git revision resolver fixtures` | P0 handoff; needs contract proof before closeout. |
| 2 / 2.F.5 | Gateway goal judge and subgoal command | Wire a production goal judge into the gateway manager and add the missing /goal subgoal add\|remove\|list\|clear command. The judge uses the auxiliary model slot, fails open to continue on transport or parse errors, and marks done only on an explicit done verdict. Subgoals persist in GoalState.Subgoals and appear in /goal status and the continuation prompt. | operator, system | `internal/gateway/goal_loop_test.go; internal/persistence/session/goal_state_test.go; internal/app/gateway/service_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Hermes auth priority credential reordering | Implement `gormes auth priority <provider> <target> <priority>` for an existing credential pool. Resolve target by 1-based displayed index, entry ID, or unique case-insensitive exact label; reject missing or ambiguous targets without writing. Move the selected credential to the requested zero-based position, clamp to the pool bounds, persist contiguous priorities atomically, preserve every non-priority field including cooldown state, and reproduce Hermes's Anthropic manual-before-seeded ordering. Report the effective priority and the upstream clamp/provider-strategy notes without exposing credential values. This row excludes `auth add --priority` and unrelated auth/documentation parity work. | operator | `internal/platform/cli/gormescli/modules/providers/auth_command_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.Q | TUI goal auto-continuation and judge | Finish the in-flight native TUI goal slice. On each idle frame after a goal turn, increment the turn count once, evaluate an injected goal judge, pause at the budget, and submit the continuation prompt exactly once. Paused, done, cleared, cancelled, queued, and failed turns must not continue. The judge fails open on missing or malformed output. | operator, system | `internal/tui/goal/goal_test.go; internal/tui/goal_continuation_test.go; internal/tui/slash_goal_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
