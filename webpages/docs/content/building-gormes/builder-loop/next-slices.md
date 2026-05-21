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
| 2 / 2.B.12 | Hermes gateway platform strict-fidelity source-pair expansion | Expand strict-fidelity mappings for Hermes gateway platform implementations and TUI gateway bridge files. The pass must classify platform adapters, platform helper modules, API-server platform surface, TUI gateway websocket/render/protocol bridge, and platform docs into completed channel rows, planned adapter rows, or explicit exclusions. | operator, system | `internal/channels strict-fidelity gateway platform mapping fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.I | Hermes agent runtime strict-fidelity source-pair expansion | Expand source-pair and progress mappings for unmapped Hermes `agent` runtime files before treating them as runtime implementation gaps. The pass must classify transports, LSP helpers, context/compression helpers, prompt caching, retry/rate diagnostics, conversation loop helpers, tool dispatch helpers, and safety/redaction helpers into existing Gormes provider/runtime/tool rows or new builder rows. | operator, system | `internal/fidelity agent runtime strict-fidelity mapping fixture` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.I | Hermes plugin catalog strict-fidelity classifier | Classify Hermes first-party plugin families into Gormes plugin/provider/channel/memory rows or explicit exclusions. The classifier must cover model-provider plugin manifests, memory plugins, web/browser/image/video plugins, platform plugins, Google Meet, Teams pipeline, Spotify, and plugin docs so strict fidelity can distinguish runtime gaps from catalog-only compatibility evidence. | operator, system | `internal/plugins hermes plugin catalog strict-fidelity fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.O | Long-term plan: profile fleet supervisor and single control-plane gateway | Define Gormes' long-term profile-fleet runtime so operators get one control surface for all named profiles while preserving Hermes-compatible profile state separation. The near-term per-profile gateway services remain a compatibility bridge; the target is a fleet supervisor that can enumerate configured profiles, start/stop/restart profile-scoped workers or a proven profile-scoped in-process equivalent, validate token ownership, surface per-profile health, and coordinate update/restart-all flows without sharing config, auth, sessions, memory, tool state, or kernels across profiles. | operator, gateway, system | `internal/gateway/fleet_supervisor_test.go; cmd/gormes/gateway_fleet_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.Q | Hermes ui-tui strict-fidelity action matrix | Map the unmapped Hermes `ui-tui` source and test surface into Gormes-native TUI rows, owned-divergence notes, or explicit exclusions. The matrix must cover command dispatch, viewport/history stores, RPC/gateway client events, terminal modes, clipboard/OSC52, provider/model UI, approval actions, and state isolation before the strict-fidelity report can stop treating `ui-tui` as an undifferentiated blocker bucket. | operator, system | `internal/tui hermes-ui-tui strict-fidelity matrix fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.Q | Hermes web dashboard strict-fidelity contract map | Classify Hermes `web/src` dashboard behavior into Gormes API/TUI gateway contracts, owned public-site divergence, or explicit exclusions. The map must connect chat, sessions, profiles, plugins, OAuth/provider panels, model picker, cron/admin pages, i18n, theme/plugin slots, and gateway client event shapes to Gormes runtime rows before dashboard parity is claimed. | operator, system | `internal/apiserver dashboard contract fixtures; internal/tuigateway gateway-client fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.C | Hermes skill catalog strict-fidelity classifier | Classify Hermes bundled and optional skill catalog files into Gormes skill-store rows, catalog-copy rows, unsupported-skill exclusions, or owned-divergence notes. The classifier must preserve SKILL.md metadata, support-file/reference layout, DESCRIPTION.md category docs, optional-skill install status, and docs-site generation boundaries without blindly copying Python-only examples into runtime packages. | operator, system | `internal/skills optional skill strict-fidelity catalog fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 7 / 7.E | SimpleX Chat platform plugin parity | Port Hermes' SimpleX Chat platform plugin into Gormes behind the shared channel adapter contract: local daemon/WebSocket configuration, allowlist admission, opaque contact IDs, DM pairing, outbound delivery, command routing, and status/degraded evidence. | - | `internal/channels/simplex fake WebSocket fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
