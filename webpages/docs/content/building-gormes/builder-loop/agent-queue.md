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
## 1. Hermes auth priority credential reordering

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

<!-- PROGRESS:END -->
