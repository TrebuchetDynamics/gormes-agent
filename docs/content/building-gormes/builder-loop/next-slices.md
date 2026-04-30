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
| 5 / 5.F | Skill registries | Native skills hub registry providers expose source-backed, read-only metadata for HermesIndex, ClawHub, and LobeHub before any write-capable install flow: a centralized Hermes index cache is preferred for all-source search when available; ClawHub and LobeHub are community-trust fallback providers with deterministic search, inspect, fetch-metadata, stale-cache fallback, and typed degraded evidence for unavailable, malformed, timeout, and rate-limited upstreams. The slice must not install, activate, quarantine, or mutate skills; it only feeds the existing HubRegistryProvider/Search read model and future install rows. | operator, system | `internal/skills/hub_registry_sources_test.go` | Unblocks Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC, Skill registries unavailable-network UX fixtures. |
<!-- PROGRESS:END -->
