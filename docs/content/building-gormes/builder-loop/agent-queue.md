---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Multi-account auth

- Phase: 4 / 4.G
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: Gormes exposes a provider-neutral multi-account credential-pool helper that can load persisted credential records, select an available credential by Hermes-compatible strategy, mark an active credential exhausted with redacted evidence, and rotate without reading live provider tokens or contacting provider/keychain services.
- Trust class: operator, system
- Ready when: Token vault is complete and provides XDG/profile-scoped credential-file safety; this slice may share internal/config profile/path helpers but must not mount or read live secrets., The builder can model credential records with neutral placeholder tokens in temp HOME/HERMES_HOME/XDG fixtures only., The first public seam is a pure Go helper for pool persistence, selection, exhaustion cooldown, and lease accounting; provider-specific refresh flows stay in later rows.
- Not ready when: The slice imports, launches, shells out to, or calls hermes-agent runtime services instead of porting the contract natively in Go., Tests read Juan's live ~/.hermes auth.json/config.yaml, platform keychains, Claude/Codex credential files, or make provider/network calls., The implementation wires Anthropic, Codex, Nous, OpenRouter, or Google refresh/auth flows instead of only creating provider-neutral pool semantics., Error/status evidence prints access_token, refresh_token, API key bytes, host secret paths, or raw provider payloads.
- Degraded mode: Empty pools, exhausted credentials, invalid strategy names, corrupt records, and unsafe credential payloads return structured redacted evidence while preserving the persisted pool for operator recovery.
- Fixture: `internal/config/credential_pool_test.go`
- Write scope: `internal/config/credential_pool.go`, `internal/config/credential_pool_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/config -run '^TestCredentialPool' -count=1`, `go test ./internal/config -count=1`, `go run ./cmd/progress validate`
- Done signal: Credential-pool fixtures prove native provider-neutral load/select/exhaust/rotate/lease behavior with redacted evidence and no live credential/provider access.
- Acceptance: TestCredentialPoolLoadRoundTrip stores and reloads neutral credential records under a temp Hermes profile while redacting token-bearing fields from status evidence., TestCredentialPoolSelectStrategies proves fill_first, round_robin, least_used, and random strategies select only non-exhausted records with deterministic fixtures for non-random paths., TestCredentialPoolExhaustedCooldownAndRotate marks the current credential exhausted with reason/code/reset_at metadata, skips it until the cooldown expires, then rotates to the next available credential., TestCredentialPoolLeaseAccounting proves acquire/release prefers the least-leased available credential and never exceeds the soft cap when another credential is below cap., TestCredentialPoolCorruptStoreEvidence preserves the corrupt pool file and returns credential_pool_corrupt or credential_pool_empty evidence without leaking file contents or secret-looking fields.
- Source refs: ../hermes-agent/agent/credential_pool.py:PooledCredential,CredentialPool.select,mark_exhausted_and_rotate,acquire_lease,release_lease,reset_statuses, ../hermes-agent/agent/credential_pool.py:get_pool_strategy,_exhausted_until,_normalize_error_context, ../hermes-agent/hermes_cli/auth.py:read_credential_pool,write_credential_pool, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:provider routing and credential parity, internal/config/token_vault.go
- Unblocks: Anthropic OAuth/keychain credential discovery, Codex OAuth state + stale-token relogin, Provider auth selection for native gateway turns
- Why now: P0 handoff; needs contract proof before closeout.

<!-- PROGRESS:END -->
