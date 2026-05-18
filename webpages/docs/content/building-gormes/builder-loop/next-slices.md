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
| 5 / 5.L | Hermes LSP write-time semantic diagnostics | After `write_file` or `patch`, Gormes runs a language-server diagnostic pass equivalent to Hermes' write-time LSP surface, shifts baseline ranges through edits, and returns new semantic errors to the agent without blocking unsupported languages. | - | `internal/tools LSP diagnostic fake-server fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | Hermes x_search tool and auth surface | Expose Hermes' first-class `x_search` tool in Gormes with a descriptor, OAuth/API-key auth status, query/result envelope, rate-limit/degraded errors, and registry/toolset visibility without requiring live X credentials in tests. | - | `internal/tools x_search fake transport fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Hermes session recap command surface | Port Hermes' session recap command as a Gormes-native read-only session summarizer over local session/transcript storage, preserving output modes, missing-session diagnostics, and provider-free degraded behavior. | - | `cmd/gormes session recap fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Long-term plan: profile fleet supervisor and single control-plane gateway | Define Gormes' long-term profile-fleet runtime so operators get one control surface for all named profiles while preserving Hermes-compatible profile state separation. The near-term per-profile gateway services remain a compatibility bridge; the target is a fleet supervisor that can enumerate configured profiles, start/stop/restart profile-scoped workers or a proven profile-scoped in-process equivalent, validate token ownership, surface per-profile health, and coordinate update/restart-all flows without sharing config, auth, sessions, memory, tool state, or kernels across profiles. | operator, gateway, system | `internal/gateway/fleet_supervisor_test.go; cmd/gormes/gateway_fleet_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.C | Hermes v0.14 optional skill catalog refresh | Refresh the Gormes skill catalog and metadata compatibility checks against Hermes v0.14 optional skills, including devops/pinggy-tunnel, research/darwinian-evolver, research/osint-investigation, and the updated Notion skill, without blindly copying unsupported Python scripts into runtime packages. | - | `internal/skills optional skill catalog fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 7 / 7.E | SimpleX Chat platform plugin parity | Port Hermes' SimpleX Chat platform plugin into Gormes behind the shared channel adapter contract: local daemon/WebSocket configuration, allowlist admission, opaque contact IDs, DM pairing, outbound delivery, command routing, and status/degraded evidence. | - | `internal/channels/simplex fake WebSocket fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
