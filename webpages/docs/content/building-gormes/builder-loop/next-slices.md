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
| 5 / 5.O | Bubble Tea Messaging Platforms setup: Telegram-first Hermes fidelity | `gormes setup gateway` becomes the Bubble Tea-only Messaging Platforms setup surface for TTY users and the source of channel setup truth for first-run setup. The first executable slice ships Telegram end to end: token capture and validation, allowlist/pairing/open access policy, structured home_channel config, redacted review-before-write, `--plan`, non-TTY guidance, GORMES_* plus Hermes-compatible env aliases, explicit Hermes migration mapping, channel-scoped offline doctor evidence, and one consolidated gateway lifecycle recommendation after selected flows complete. Runtime paths stay Gormes-owned and normal runtime never reads Hermes config or dotenv files. | operator, gateway, system | `cmd/gormes/setup_gateway_bubbletea_test.go; internal/gateway/channel_setup_test.go; internal/config/telegram_config_test.go; internal/migrate/hermes/telegram_mapping_test.go` | P0 handoff; needs contract proof before closeout. |
| 2 / 2.A | Coding-agent delegation: Phase 1 scaffold (internal/codingagents) | Shared internal/codingagents package providing the CodingAgent interface, CodingAgentRequest/Result, mode constants, binary availability detection, workspace guard with default deny list, git snapshot/diff helper, and prompt wrapper. No tools are registered in this slice; adapters and registry exposure land in later phases. | operator, system | `internal/codingagents` | Already active; contract metadata keeps execution bounded. |
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
