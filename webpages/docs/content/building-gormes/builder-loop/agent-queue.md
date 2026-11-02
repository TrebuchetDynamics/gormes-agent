---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`webpages/docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Advance Hermes checkout and source pairs to latest upstream

- Phase: 4 / 4.I
- Owner: `docs`
- Size: `large`
- Status: `planned`
- Priority: `P0`
- Contract: Fetch NousResearch/hermes-agent origin/main, advance the in-repo hermes-agent gitlink to that exact SHA, update hermes-source-pairs.json to the same SHA, revalidate every source pair, and regenerate the contract inventory. The planning seed is fbf6c6c91d (17,175 commits ahead of d0288be5b); the builder must use the latest origin/main at execution time.
- Trust class: operator, system
- Ready when: Origin/main is reachable from the in-repo hermes-agent checkout., The source-pairs manifest uses the existing schema and SHA field., The source-pair validate, sync-sha, and inventory commands exist and pass on a matching baseline.
- Not ready when: The submodule is advanced without updating the source-pairs SHA., A source pair is deleted or left pointing at a path that no longer exists., An old d0288be5b SHA remains in the inventory or atom evidence.
- Degraded mode: If upstream cannot be fetched, the submodule cannot be advanced, or any source pair lacks its upstream file at the new SHA, stop with the exact unresolved paths and keep the old baseline evidence unchanged.
- Fixture: `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json; webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json; webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md`
- Write scope: `hermes-agent`, `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json`, `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json`, `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go run ./cmd/gormes-repo hermes-source-pairs validate`, `go run ./cmd/gormes-repo hermes-contract-inventory`, `go test ./internal/planning/progress/repoctl/inventory/... -count=1`, `go test ./webpages/docs -run TestUpstreamCoverage -count=1`, `go run ./cmd/progress validate`
- Done signal: The checkout, source-pairs manifest, and contract inventory all record the same latest origin/main SHA with no stale d0288be5b reference.
- Acceptance: TestHermesCheckoutMatchesLatestUpstream proves the hermes-agent gitlink SHA equals the fetched origin/main SHA recorded in the manifest., TestSourcePairsValidateAtLatestUpstream runs the source-pair validator at the new SHA with zero missing upstream files., TestContractInventoryRecordsLatestSHA proves the generated inventory JSON and Markdown record the new SHA and current counts.
- Source refs: https://github.com/NousResearch/hermes-agent/commit/fbf6c6c91d, ./hermes-agent/agent/, ./hermes-agent/tools/, ./hermes-agent/gateway/, ./hermes-agent/hermes_cli/, ./hermes-agent/ui-tui/, ./hermes-agent/web/, ./hermes-agent/plugins/, ./hermes-agent/skills/, webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json
- Unblocks: Classify the upstream delta by feature family, Plan round 2: split upstream delta into builder rows
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Rolling upstream baseline freshness gate

- Phase: 4 / 4.I
- Owner: `docs`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Implement the always-upstream freshness step for the strict parity gate. The gate reads the recorded baseline SHA, resolves the latest fetched NousResearch/hermes-agent origin/main, and fails closed when they differ. A matching baseline proceeds to behavior evaluation and emits the source delta from the recorded SHA to origin/main.
- Trust class: operator, system
- Ready when: The source-pairs manifest and contract inventory both carry an explicit baseline SHA., The gate can resolve origin/main through an injected git revision function., The failure evidence codes baseline_stale, baseline_unverified, and baseline_match are defined.
- Not ready when: The gate passes when the recorded SHA differs from origin/main., The gate silently skips an unreachable or missing origin/main., The gate emits no delta from the recorded SHA to origin/main.
- Degraded mode: A missing or unreachable origin/main, a missing recorded SHA, or a revision-resolution error produces baseline_unverified and no parity claim. A stale baseline produces baseline_stale and no parity claim.
- Fixture: `internal/planning/progress/repoctl/inventory/hermes_contract_inventory_test.go; fake git revision resolver fixtures`
- Write scope: `internal/planning/progress/repoctl/inventory/`, `cmd/gormes-repo/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/planning/progress/repoctl/inventory/... -run 'Baseline\|Drift' -count=1`, `go test ./cmd/gormes-repo -count=1`, `go run ./cmd/progress validate`
- Done signal: The gate rejects a stale or unverified baseline, accepts a matching latest baseline, and prints the delta for the current pass.
- Acceptance: TestBaselineFreshnessRejectsStale records a SHA older than origin/main and exits non-zero with baseline_stale plus both SHAs., TestBaselineFreshnessAcceptsLatest records the origin/main SHA and passes the freshness step with baseline_match., TestBaselineFreshnessFailsClosedOnUnresolvableOrigin records a revision error and returns baseline_unverified without a parity claim., TestBaselineDeltaListsChangedPaths emits the added, modified, and deleted paths from the recorded SHA to origin/main.
- Source refs: ./hermes-agent@fbf6c6c91d, webpages/docs/content/building-gormes/architecture_plan/progress.json, webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json, webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json, internal/planning/progress/repoctl/inventory/hermes_contract_inventory.go, internal/planning/progress/fidelity/report.go
- Unblocks: Classify the upstream delta by feature family, Plan round 2: split upstream delta into builder rows
- Why now: P0 handoff; needs contract proof before closeout.

## 3. Gateway goal judge and subgoal command

- Phase: 2 / 2.F.5
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Wire a production goal judge into the gateway manager and add the missing /goal subgoal add\|remove\|list\|clear command. The judge uses the auxiliary model slot, fails open to continue on transport or parse errors, and marks done only on an explicit done verdict. Subgoals persist in GoalState.Subgoals and appear in /goal status and the continuation prompt.
- Trust class: operator, system
- Ready when: The goal loop accepts an injected GoalJudge and the manager config carries it., The auxiliary model slot can be resolved without live credentials in tests., The gateway goal command parser exposes a subgoal branch.
- Not ready when: The judge marks done on missing or malformed output., Subgoals are stored outside GoalState.Subgoals., A judge error stops the loop instead of failing open.
- Degraded mode: A missing auxiliary client, malformed judge output, or transport error continues the goal and records typed evidence. A missing goal store reports goal_metadata_unavailable. A subgoal command with no goal reports No goal set.
- Fixture: `internal/gateway/goal_loop_test.go; internal/persistence/session/goal_state_test.go; internal/app/gateway/service_test.go`
- Write scope: `internal/gateway/`, `internal/app/gateway/`, `internal/app/gormescmd/`, `internal/persistence/session/`, `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway -run 'Goal' -count=1`, `go test ./internal/persistence/session -run 'Goal' -count=1`, `go test ./internal/app/gateway -count=1`, `go run ./cmd/progress validate`
- Done signal: The gateway judge marks only explicit done results, fails open otherwise, and subgoals round-trip through status and continuation.
- Acceptance: TestGatewayGoalJudgeDoneMarksDone proves an explicit done verdict sets GoalStatusDone and sends no continuation., TestGatewayGoalJudgeFailsOpen proves missing, malformed, and transport-error judge output returns continue evidence., TestGatewayGoalContinuationUsesSubgoals proves the continuation prompt includes all stored subgoals., TestGatewayGoalSubgoalCommandAddRemoveList proves add, remove, and list persist in GoalState.Subgoals and appear in status.
- Source refs: ./hermes-agent/hermes_cli/goals.py@fbf6c6c91d:JUDGE_SYSTEM_PROMPT, ./hermes-agent/hermes_cli/goals.py@fbf6c6c91d:make_goal_judge_callback, ./hermes-agent/hermes_cli/goals.py@fbf6c6c91d:cmd_subgoal, ./hermes-agent/tests/hermes_cli/test_goals.py@fbf6c6c91d, internal/gateway/goal_loop.go, internal/gateway/goalloop/goal.go, internal/persistence/session/goal_state.go, internal/app/gateway/service.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Hermes auth priority credential reordering

- Phase: 5 / 5.O
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Implement `gormes auth priority <provider> <target> <priority>` for an existing credential pool. Resolve target by 1-based displayed index, entry ID, or unique case-insensitive exact label; reject missing or ambiguous targets without writing. Move the selected credential to the requested zero-based position, clamp to the pool bounds, persist contiguous priorities atomically, preserve every non-priority field including cooldown state, and reproduce Hermes's Anthropic manual-before-seeded ordering. Report the effective priority and the upstream clamp/provider-strategy notes without exposing credential values. This row excludes `auth add --priority` and unrelated auth/documentation parity work.
- Trust class: operator
- Ready when: At implementation start, confirm the Hermes submodule still resolves to d0288be5b3330d2442e3907185b8e9d0958297bb; do not refresh the checkout as part of this slice., CLI tests seed only synthetic pooled credentials under t.TempDir and inject the GORMES_HOME/HERMES_HOME test roots; no developer home, provider credentials, or network access is used., Use the existing atomic credential-store write path and verify invalid/ambiguous targets leave the fixture auth.json unchanged.
- Not ready when: The test harness would read or mutate a live home, use real credentials, or contact an OAuth/provider endpoint., The command cannot operate on the same injected home/profile scope used by existing auth-list tests; keep the row draft until that scope is resolved rather than writing a different pool., The pinned Hermes source behavior or SHA has changed; reconcile source evidence before implementation rather than silently changing this contract.
- Degraded mode: Fail closed on corrupt stores, invalid/ambiguous targets, and write errors; do not expose credential values or alter the stored pool on a rejected operation.
- Fixture: `internal/platform/cli/gormescli/modules/providers/auth_command_test.go`
- Write scope: `internal/config/credentials/pool.go`, `internal/config/credentials/pool_test.go`, `internal/platform/cli/gormescli/modules/providers/auth.go`, `internal/platform/cli/gormescli/modules/providers/auth_runtime.go`, `internal/platform/cli/gormescli/modules/providers/auth_command_test.go`, `internal/platform/cli/gormescli/modules/providers/auth_runtime_test.go`
- Test commands: `go test ./internal/config/credentials -run '^TestCredentialPoolMoveEntry' -count=1`, `go test ./internal/platform/cli/gormescli/modules/providers -run '^TestAuthPriority' -count=1`
- Done signal: Focused pool and CLI tests prove target matching, clamping, persistence after reload, cooldown/state preservation, Anthropic manual-first behavior, output redaction, and no write on invalid targets using only temporary fixtures., `go run ./cmd/progress validate` passes after the row is reconciled with implementation evidence; no parity claim is made beyond this command behavior.
- Acceptance: `gormes auth priority <provider> <target> <priority>` is registered and accepts 1-based index, entry ID, and unique case-insensitive exact label targets; missing and ambiguous targets fail without mutation., Requested priority is clamped to [0, pool size-1], all entries receive contiguous priorities, and reload/list shows the persisted order; negative and oversized requests are covered., Moving an entry preserves all non-priority fields, including request counters and cooldown/error fields; invalid targets preserve the exact prior store., Anthropic manual-source entries remain ahead of seeded entries per the pinned normalization rule; other providers retain the requested ordering., Output reports the effective position, clamp/Anthropic reason, and non-fill-first strategy note as applicable, uses Gormes command spelling, and never prints token or secret values.
- Source refs: NousResearch/hermes-agent@d0288be5b3330d2442e3907185b8e9d0958297bb:hermes_cli/subcommands/auth.py:51-55 build_auth_parser priority command, NousResearch/hermes-agent@d0288be5b3330d2442e3907185b8e9d0958297bb:hermes_cli/auth_commands.py:466-495 _report_priority and auth_priority_command, NousResearch/hermes-agent@d0288be5b3330d2442e3907185b8e9d0958297bb:agent/credential_pool_admin.py:67-108 move_entry and resolve_target, NousResearch/hermes-agent@d0288be5b3330d2442e3907185b8e9d0958297bb:agent/credential_pool.py:2487-2508 _normalize_pool_priorities, NousResearch/hermes-agent@d0288be5b3330d2442e3907185b8e9d0958297bb:tests/agent/test_credential_pool_operations.py:53-70 test_priority_persists_contiguous_order_without_clearing_cooldown and test_priority_honors_anthropic_manual_first, Gormes:internal/platform/cli/gormescli/modules/providers/auth.go:162-184 NewAuthCommandWithSeams registers add/list/remove/reset/status/logout/spotify but no priority command, Gormes:internal/platform/cli/gormescli/modules/providers/auth_runtime.go:914-935 loadAuthEntries and findCredentialIndex, Gormes:internal/config/credentials/pool.go:347-350 CredentialPool.Entries; 585-599 persistUnlocked; 658-684 normalizeCredentialEntries; 872-916 writeCredentialPoolAuthStore
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. TUI goal auto-continuation and judge

- Phase: 5 / 5.Q
- Owner: `tui`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Finish the in-flight native TUI goal slice. On each idle frame after a goal turn, increment the turn count once, evaluate an injected goal judge, pause at the budget, and submit the continuation prompt exactly once. Paused, done, cleared, cancelled, queued, and failed turns must not continue. The judge fails open on missing or malformed output.
- Trust class: operator, system
- Ready when: The in-flight TUI goal control and bar slice is present., The TUI idle frame path exposes a single post-turn hook., The judge and continuation prompt are injected interfaces.
- Not ready when: Continuation fires more than once for one idle frame., Continuation ignores pause, done, clear, cancel, or queue state., The judge is called with live credentials in a fixture.
- Degraded mode: A missing judge continues with continue evidence. A missing goal store disables continuation and keeps the bar empty. A session without an ID does not continue.
- Fixture: `internal/tui/goal/goal_test.go; internal/tui/goal_continuation_test.go; internal/tui/slash_goal_test.go`
- Write scope: `internal/tui/goal/`, `internal/tui/`, `internal/adapters/tuiadapter/`, `internal/platform/cli/gormescli/tuiapp/`, `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui/... ./internal/tui/goal -count=1`, `go test ./internal/adapters/tuiadapter -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: The TUI continues an active goal until the judge says done or the budget pauses it, with one submission per idle frame.
- Acceptance: TestGoalAfterIdleSubmitsOnce proves one active-goal idle frame submits exactly one continuation prompt and increments TurnsUsed once., TestGoalAfterIdleBudgetPauses proves reaching MaxTurns pauses the goal and submits nothing., TestGoalAfterIdleJudgeDone proves a done verdict marks the goal done and submits nothing., TestGoalAfterIdleSkipsPausedQueuedCancelled proves paused, queued, cancelled, and failed turns do not continue.
- Source refs: ./hermes-agent/hermes_cli/goals.py@fbf6c6c91d, ./hermes-agent/ui-tui/src/app/goalStatus.ts@fbf6c6c91d, ./hermes-agent/ui-tui/src/components/goalBar.tsx@fbf6c6c91d, internal/tui/goal/goal.go, internal/tui/slash_goal.go, internal/tui/update.go, internal/persistence/session/goal_state.go, internal/gateway/goalloop/goal.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
