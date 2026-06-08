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
## 1. Bitwarden disk cache parity

- Phase: 5 / 5.O
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Gormes ports Hermes' two-layer Bitwarden Secrets Manager cache for startup/config external-secret loading: `ApplyBitwarden`/the underlying fetch path first reuses a fresh in-process entry keyed by token fingerprint + project_id + server_url, then reads `$GORMES_HOME/cache/bws_cache.json` when `cache_ttl_seconds > 0`, promotes a fresh disk hit back into the process cache, and writes successful fetched secret maps back to disk atomically with mode 0600. Cache files must contain only the fingerprint key, fetched_at timestamp, and fetched Bitwarden secret map; they must never contain the raw access token, raw command env, stderr, project-list output, or unrelated provider config. Expired, malformed, wrong-key, unreadable, or unwritable cache state is best-effort: ignore and re-fetch through fake `bws` without blocking startup.
- Trust class: operator, system
- Ready when: Bitwarden status/sync/disable, managed install, and setup wizard rows are complete, so the remaining source-backed cache behavior can be added under the existing external-secret loader seam., Builder can add cache helpers and tests in `internal/config/externalsecrets` using temp homes, fake runner output, and optional fake time; no CLI command, live Bitwarden, network, or real bws binary is required., Scope is limited to startup/fetch cache semantics. Credential-pool borrowed-source persistence remains a separate row because it changes auth metadata, not fetch caching.
- Not ready when: The slice attempts credential-pool borrowed-source persistence, setup wizard changes, project picker changes, or provider credential UI in the same pass., The implementation writes the bootstrap access token, command env, stderr, raw .env contents, or unrelated provider config into `bws_cache.json`., Tests require live Bitwarden credentials, GitHub/network access, wall-clock sleeps, a real bws install, or operator-specific paths.
- Degraded mode: If the cache is disabled with `cache_ttl_seconds <= 0`, stale, malformed, unreadable, has a key mismatch, or cannot be written atomically, Gormes falls back to the existing fakeable `bws secret list` path and returns the normal redacted Bitwarden report. Startup must remain non-fatal and secret-safe.
- Fixture: `internal/config/externalsecrets/bitwarden_cache_test.go with temp GORMES_HOME, fake clock, fake bws runner, malformed/wrong-key/stale cache fixtures, and permission/mode assertions; no live Bitwarden, GitHub, network, or real bws binary.`
- Write scope: `internal/config/externalsecrets/`, `internal/app/secrets/`, `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json/modules/providers.json`
- Test commands: `go test ./internal/config/externalsecrets -run Bitwarden -count=1`, `go test ./internal/app/secrets -run Bitwarden -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Hermetic cache tests prove first-fetch write, fresh disk-cache hit without bws invocation, in-process cache hit, cache disabled path, stale/wrong-key/malformed cache fallback, server-url key isolation, mode 0600, and token/secret redaction.
- Acceptance: A first successful fake `bws secret list` with `cache_ttl_seconds=300` writes `$GORMES_HOME/cache/bws_cache.json` atomically with mode 0600, containing a stable non-token key, `fetched_at`, and fetched env-var secrets only., A second ApplyBitwarden call in a fresh process-cache state with the same token/project/server and fresh disk cache uses the disk cache without invoking fake bws, applies/skips env vars exactly like a live fetch, and records Bitwarden source labels for applied keys., Fresh in-process cache avoids repeated fake bws calls within one process; disabling cache with `cache_ttl_seconds <= 0` always calls fake bws and does not read/write disk cache., Wrong cache key, stale fetched_at, malformed JSON, non-object secrets, invalid env-var names, unreadable files, and write failures are ignored or warned best-effort, then Gormes re-fetches through fake bws without leaking token or secret values., Server URL participates in the cache key so US/EU/self-hosted cache entries do not bleed across regions.
- Source refs: ./hermes-agent/agent/secret_sources/bitwarden.py:72 _DISK_CACHE_BASENAME, ./hermes-agent/agent/secret_sources/bitwarden.py:75 _disk_cache_path, ./hermes-agent/agent/secret_sources/bitwarden.py:87 _cache_key_str, ./hermes-agent/agent/secret_sources/bitwarden.py:93 _read_disk_cache, ./hermes-agent/agent/secret_sources/bitwarden.py:123 _write_disk_cache, ./hermes-agent/agent/secret_sources/bitwarden.py:155 _CachedFetch.is_fresh, ./hermes-agent/agent/secret_sources/bitwarden.py:438 fetch_bitwarden_secrets, ./hermes-agent/hermes_cli/env_loader.py:250 _apply_external_secret_sources, webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md:Bitwarden Secrets Manager source, webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Providers, Models, And Credentials / Credential pool, token vault, and auth commands, internal/config/externalsecrets/bitwarden.go:ApplyBitwarden, internal/config/config.go:applyExternalSecretSources, internal/app/secrets/service.go:BitwardenSetup/BitwardenSync
- Unblocks: Credential-pool Bitwarden borrowed-source persistence, Provider setup/status secrets provenance display
- Why now: Unblocks Credential-pool Bitwarden borrowed-source persistence, Provider setup/status secrets provenance display.

<!-- PROGRESS:END -->
