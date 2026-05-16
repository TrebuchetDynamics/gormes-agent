---
title: "Blocked Slices"
weight: 40
aliases:
  - /building-gormes/blocked-slices/
---

# Blocked Slices

This page is generated from canonical `progress.json` rows that declare
`blocked_by`.

Use it to avoid assigning work before the dependency chain is ready.

<!-- PROGRESS:START kind=blocked-slices -->
| Phase | Slice | Blocked by | Ready when | Unblocks |
|---|---|---|---|---|
| 5 / 5.O | Gormes update verified binary swap and rollback | Gormes update release planner and dry-run contract | `Gormes update release planner and dry-run contract` is complete and exposes a typed release update plan., The builder can fixture release artifacts, checksums, signatures/provenance states, staged binary smoke results, and filesystem rename failures without network access., Snapshot retention policy is documented in the plan row and can be implemented without touching user config/session/memory state. | Gormes update bundled assets and skills sync |
| 5 / 5.O | Gormes update bundled assets and skills sync | Gormes update verified binary swap and rollback | `Gormes update verified binary swap and rollback` is complete and exposes snapshot/rollback primitives for release installs., Release artifacts can include or point to a manifest with checksums for bundled assets and skills., Tests can fixture bundled skill roots, user-modified skill copies, removed manifest entries, asset checksum failures, and rollback without live network. | Gormes update managed service drain and restart |
| 5 / 5.O | Gormes update managed service drain and restart | Gormes update bundled assets and skills sync | `Gormes update bundled assets and skills sync` is complete and release updates can roll back binary/assets/skills coherently., Service restart and gateway status helpers are already tested through fake service/runtime seams., Tests can inject process tables, lock files, service states, drain timeouts, restart failures, health-check results, and rollback state without real systemd/launchd/sc.exe or live gateway. | Installed-runtime self-healing smoke, Gormes release operator trust report |
| 5 / 5.Q | Kernel cross-provider client swap for in-session model switch | Kernel in-session model-switch seam for the native TUI | Prerequisite row 'Kernel in-session model-switch seam for the native TUI' is complete (PlatformEventSetModel + k.sessionProvider exist)., A provider+model -> hermes.ModelRoute resolver path is identified in internal/hermes (BaseURL/APIMode/KeyEnv obtainable without a new catalog). | Native TUI /model slash command binding over the existing model picker |
| 8 / 8.A | TD social presence connected to blog feed | TD engineering blog scaffolded and live | TD blog (8.A row 1) is live and emitting a feed., Operator has chosen a social platform and created the account. | - |
| 8 / 8.C | Engineering writeup #1: autonomous Hermes-porting loop | TD engineering blog scaffolded and live, Loop $/iteration cost metric in status file | TD blog (8.A row 1) is live., Loop $/iteration cost telemetry (8.F) has at least one week of data., Operator has decided the publication date and platform (HN/Lobsters/Reddit). | Engineering writeup #2: validation-gated agentic engineering, Engineering writeup #3: Gormes vs Hermes-Python benchmarks, HN launch post for Gormes 1.0 |
| 8 / 8.D | Single-binary cross-platform release pipeline | Sharp v1.0 differentiator decision | Sharp v1.0 differentiator (8.D row 1) is decided., go build ./cmd/gormes succeeds on all seven target GOOS/GOARCH combinations from CI or a Linux runner where cross-compilation is supported. | - |
| 8 / 8.D | Gormes streaming tool-trail status + spinner cadence wiring | Gormes-owned streaming feedback uplift | Gormes-owned streaming feedback uplift (R3) is complete (thinking indicator + collapsible output shipped). | - |
<!-- PROGRESS:END -->
