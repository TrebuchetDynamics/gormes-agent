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
- Contract: Native skills hub registry providers expose source-backed, read-only metadata for the current Hermes skills hub source adapters before any write-capable install flow: OptionalSkillSource, HermesIndexSource, SkillsShSource, WellKnownSkillSource, UrlSource, GitHubSource, ClawHubSource, ClaudeMarketplaceSource, and LobeHubSource are the active upstream contract at Hermes 69d4800d. This executable slice should add only the missing remote registry read-model providers over the existing HubRegistryProvider/Search seam, with source filtering, trust normalization, centralized-index preference, stale-cache fallback, and typed degraded evidence for unavailable, malformed, timeout, empty, and rate-limited upstreams. Url direct parsing and optional bundled-skill inventory stay in their existing rows; this slice must not install, activate, quarantine, guard-scan, or mutate skills.
- Trust class: operator, system
- Ready when: The existing internal/skills HubRegistryProvider/Search read model remains the public seam for registry metadata., Tests inject fake HTTP clients, response fixtures, or temp cache roots for Skills.sh/GitHub, WellKnown, HermesIndex, ClawHub, ClaudeMarketplace, and LobeHub; no live network, GitHub token, gh CLI, active skill store, or quarantine directory is required., UrlSource direct SKILL.md parsing is treated as already covered by the separate complete `Skills hub direct URL candidate parser` row rather than reimplemented here.
- Not ready when: The slice downloads arbitrary bundle files, writes active/candidate skills, performs guard scans, runs install commands, or changes skill prompt injection., The slice omits current Hermes 69d4800d source adapters (HermesIndexSource, ClaudeMarketplaceSource, or LobeHubSource) from the read-model contract, or treats the centralized index preference in parallel_search_sources as out of scope without a source-backed split row., ClawHub, Skills.sh, WellKnown, or GitHub results are treated as builtin/trusted without the upstream trust rules, or malformed remote payloads panic instead of returning typed degraded evidence.
- Degraded mode: Network failures, non-200 responses, malformed JSON, expired/missing cache, empty registries, and rate limits return typed evidence such as registry_unavailable, registry_rate_limited, registry_malformed, registry_cache_stale, or registry_empty without panics and without active-store mutation.
- Fixture: `internal/skills/hub_registry_sources_test.go`
- Write scope: `internal/skills/hub_registry_sources.go`, `internal/skills/hub_registry_sources_test.go`, `internal/skills/hub_search.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/skills -run 'TestClawHubProvider\|TestHermesIndexProvider\|TestClaudeMarketplaceProvider\|TestLobeHubProvider\|TestSkillsShProvider\|TestWellKnownRegistryProvider\|TestRegistryProvider' -count=1`, `go test ./internal/skills -count=1`, `go run ./cmd/progress validate`
- Done signal: Native registry providers expose current Hermes HermesIndex, Skills.sh/GitHub, WellKnown, ClawHub, ClaudeMarketplace, and LobeHub metadata through HubRegistryProvider/Search with fixture-backed cache/degraded evidence tests, centralized-index source filtering, and no install/store mutation.
- Acceptance: TestClawHubProviderCommunityTrustAndDegradedEvidence proves ClawHub search/inspect normalizes slug/name/tags, assigns community trust, reports registry_unavailable or registry_rate_limited for failures, and never mutates the active store., TestHermesIndexProviderPrefersCachedIndex proves centralized Hermes index fixtures return metadata with zero API calls and source-router search can skip duplicate remote API sources when the index is available and source_filter=all., TestClaudeMarketplaceProviderCommunityTrustAndCacheEvidence proves marketplace.json fixtures resolve source paths, normalize trust through TRUSTED_REPOS, and report typed malformed/unavailable evidence without store writes., TestLobeHubProviderAgentMetadataAndDegradedEvidence proves LobeHub agent index fixtures convert title/identifier/tags into community-trust metadata and return typed timeout/malformed evidence instead of panicking., TestSkillsShProviderDelegatesThroughGitHubMetadata proves Skills.sh result identifiers resolve to GitHub metadata/fetch IDs while preserving source=skills-sh and deterministic source-filter behavior., TestWellKnownRegistryProviderReadsIndexMetadata proves .well-known/skills index fixtures produce community-trust metadata without network or filesystem writes beyond the temp cache., TestRegistryProviderCacheFallback proves malformed or unavailable network responses reuse a valid stale cache when present and return typed evidence when no cache exists., TestRegistryProvidersDoNotInstall proves the registry-source package has no dependency on active store mutators, quarantine install paths, gateway adapters, or provider/model clients.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:GitHubSource.search,GitHubSource.fetch,GitHubSource.inspect, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:SkillsShSource.search,SkillsShSource.fetch,SkillsShSource.inspect,_discover_identifier, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:WellKnownSkillSource.search,WellKnownSkillSource.fetch,WellKnownSkillSource.inspect, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:ClawHubSource.search,ClawHubSource.fetch,ClawHubSource.inspect,_load_catalog_index, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:HermesIndexSource.search,fetch,inspect,is_available,_load_hermes_index, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:ClaudeMarketplaceSource.search,fetch,inspect,_fetch_marketplace_index, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:LobeHubSource.search,fetch,inspect,_fetch_index,_fetch_agent, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:create_source_router,parallel_search_sources,unified_search, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tests/tools/test_skills_hub.py:TestSkillsShSource,TestWellKnownSkillSource,TestUrlSource,TestSkillSourceRouter,TestUnifiedSearch, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tests/tools/test_skills_hub_clawhub.py:TestClawHubSource, internal/skills/hub_search.go, internal/skills/hub_search_test.go, internal/skills/hub_registry_sources.go:WellKnownRegistryProvider,ClawHubRegistryProvider, internal/skills/hub_registry_sources_test.go:TestWellKnownRegistryProviderReadsIndexMetadata,TestClawHubProviderCommunityTrustAndDegradedEvidence,TestClawHubProviderDegradedEvidence, internal/skills/url_candidate.go
- Unblocks: Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC, Skill registries unavailable-network UX fixtures
- Why now: Unblocks Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC, Skill registries unavailable-network UX fixtures.

<!-- PROGRESS:END -->
