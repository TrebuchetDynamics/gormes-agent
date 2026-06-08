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
| 5 / 5.O | Bitwarden disk cache parity | Gormes ports Hermes' two-layer Bitwarden Secrets Manager cache for startup/config external-secret loading: `ApplyBitwarden`/the underlying fetch path first reuses a fresh in-process entry keyed by token fingerprint + project_id + server_url, then reads `$GORMES_HOME/cache/bws_cache.json` when `cache_ttl_seconds > 0`, promotes a fresh disk hit back into the process cache, and writes successful fetched secret maps back to disk atomically with mode 0600. Cache files must contain only the fingerprint key, fetched_at timestamp, and fetched Bitwarden secret map; they must never contain the raw access token, raw command env, stderr, project-list output, or unrelated provider config. Expired, malformed, wrong-key, unreadable, or unwritable cache state is best-effort: ignore and re-fetch through fake `bws` without blocking startup. | operator, system | `internal/config/externalsecrets/bitwarden_cache_test.go with temp GORMES_HOME, fake clock, fake bws runner, malformed/wrong-key/stale cache fixtures, and permission/mode assertions; no live Bitwarden, GitHub, network, or real bws binary.` | Unblocks Credential-pool Bitwarden borrowed-source persistence, Provider setup/status secrets provenance display. |
<!-- PROGRESS:END -->
