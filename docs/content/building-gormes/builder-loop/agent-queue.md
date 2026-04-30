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

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Skill registries

- Phase: 5 / 5.F
- Owner: `skills`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Native skills hub registry providers expose source-backed, read-only metadata for HermesIndex, ClawHub, and LobeHub before any write-capable install flow: a centralized Hermes index cache is preferred for all-source search when available; ClawHub and LobeHub are community-trust fallback providers with deterministic search, inspect, fetch-metadata, stale-cache fallback, and typed degraded evidence for unavailable, malformed, timeout, and rate-limited upstreams. The slice must not install, activate, quarantine, or mutate skills; it only feeds the existing HubRegistryProvider/Search read model and future install rows.
- Trust class: operator, system
- Ready when: The existing internal/skills HubRegistryProvider/Search read model remains the public seam for registry metadata., Tests inject fake HTTP clients or response fixtures for Hermes index, ClawHub, and LobeHub; no live network, GitHub token, gh CLI, active skill store, or quarantine directory is required., The implementation can cache registry JSON under a temp root in tests and can prove stale-cache fallback without touching ~/.gormes or ~/.hermes.
- Not ready when: The slice downloads arbitrary bundle files, writes active/candidate skills, performs guard scans, runs install commands, or changes skill prompt injection., The all-source search path calls ClawHub, LobeHub, GitHub, or skills.sh when a valid Hermes centralized index fixture is available and source_filter is all., ClawHub or LobeHub results are treated as trusted/builtin, or malformed remote payloads panic instead of returning typed degraded evidence.
- Degraded mode: Network failures, non-200 responses, malformed JSON, expired/missing cache, and rate limits return typed evidence such as registry_unavailable, registry_rate_limited, registry_malformed, registry_cache_stale, or registry_empty without panics and without active-store mutation.
- Fixture: `internal/skills/hub_registry_sources_test.go`
- Write scope: `internal/skills/hub_registry_sources.go`, `internal/skills/hub_registry_sources_test.go`, `internal/skills/hub_search.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/skills -run 'TestHermesIndexProvider\|TestClawHubProvider\|TestLobeHubProvider\|TestRegistryProvider' -count=1`, `go test ./internal/skills -count=1`, `go run ./cmd/progress validate`
- Done signal: Native registry providers expose HermesIndex, ClawHub, and LobeHub metadata through HubRegistryProvider/Search with fixture-backed cache/degraded evidence tests and no install/store mutation.
- Acceptance: TestHermesIndexProviderPrefersCentralCache proves the provider returns featured/search metadata from a valid cached index, uses resolved_github_id for fetch metadata, and suppresses external API providers for source_filter=all., TestClawHubProviderCommunityTrustAndDegradedEvidence proves ClawHub search/inspect normalizes slug/name/tags, assigns community trust, reports registry_unavailable or registry_rate_limited for failures, and never mutates the active store., TestLobeHubProviderConvertsAgentToSkillMetadata proves LobeHub JSON produces a community-trust SKILL.md metadata preview with frontmatter name/description/tags and no filesystem writes., TestRegistryProviderCacheFallback proves malformed or unavailable network responses reuse a valid stale cache when present and return typed evidence when no cache exists., TestRegistryProvidersDoNotInstall proves the registry-source package has no dependency on active store mutators, quarantine install paths, gateway adapters, or provider/model clients.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:HermesIndexSource,_load_hermes_index,parallel_search_sources, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:ClawHubSource.search,ClawHubSource.fetch,ClawHubSource.inspect, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:LobeHubSource.search,LobeHubSource.fetch,LobeHubSource._convert_to_skill_md, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tests/tools/test_skills_hub.py, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tests/tools/test_skills_hub_clawhub.py, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tests/tools/test_skills_sync.py, internal/skills/hub_search.go, internal/skills/hub_search_test.go, internal/skills/url_candidate.go
- Unblocks: Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC, Skill registries unavailable-network UX fixtures
- Why now: Unblocks Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC, Skill registries unavailable-network UX fixtures.

<!-- PROGRESS:END -->
