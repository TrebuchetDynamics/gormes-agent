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
| 5 / 5.F | Skill registries | Native skills hub registry providers expose source-backed, read-only metadata for the current Hermes skills hub source adapters before any write-capable install flow: OptionalSkillSource, HermesIndexSource, SkillsShSource, WellKnownSkillSource, UrlSource, GitHubSource, ClawHubSource, ClaudeMarketplaceSource, and LobeHubSource are the active upstream contract at Hermes 69d4800d. This executable slice should add only the missing remote registry read-model providers over the existing HubRegistryProvider/Search seam, with source filtering, trust normalization, centralized-index preference, stale-cache fallback, and typed degraded evidence for unavailable, malformed, timeout, empty, and rate-limited upstreams. Url direct parsing and optional bundled-skill inventory stay in their existing rows; this slice must not install, activate, quarantine, guard-scan, or mutate skills. | operator, system | `internal/skills/hub_registry_sources_test.go` | Unblocks Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC, Skill registries unavailable-network UX fixtures. |
| 5 / 5.F | Hermes index provider cache + source-router preference | internal/skills adds a read-only HermesIndexRegistryProvider and source-router preference helper that consume cached centralized Hermes skills index fixtures before consulting duplicate remote API providers, mirroring HermesIndexSource plus create_source_router/parallel_search_sources without installing or mutating active skills. | operator, system | `internal/skills/hub_registry_sources_test.go::TestHermesIndexProviderPrefersCachedIndex` | Unblocks Marketplace/GitHub registry metadata providers, Skill registries unavailable-network UX fixtures. |
| 5 / 5.F | Marketplace/GitHub registry metadata providers | internal/skills adds read-only GitHub/Skills.sh, Claude Marketplace, and LobeHub registry providers over the existing HubRegistryProvider/Search seam, normalizing source, install IDs, tags, and community/trusted trust evidence from Hermes source adapters while keeping install, quarantine, guard scanning, and active-store mutation out of scope. | operator, system | `internal/skills/hub_registry_sources_test.go::TestSkillsShProviderDelegatesThroughGitHubMetadata+TestClaudeMarketplaceProviderCommunityTrustAndCacheEvidence+TestLobeHubProviderAgentMetadataAndDegradedEvidence` | Unblocks Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC. |
<!-- PROGRESS:END -->
