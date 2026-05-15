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
| 2 / 2.A | Coding-agent delegation: Phase 1 scaffold (internal/codingagents) | Shared internal/codingagents package providing the CodingAgent interface, CodingAgentRequest/Result, mode constants, binary availability detection, workspace guard with default deny list, git snapshot/diff helper, and prompt wrapper. No tools are registered in this slice; adapters and registry exposure land in later phases. | operator, system | `internal/codingagents` | Already active; contract metadata keeps execution bounded. |
| 9 / 9.E | Remove SSH Navivox stdio path | Delete the SSH stdio Navivox transport — `gormes navivox serve\|pair\|setup-host` CLI surface, the wire-protocol Go package (codec/frames/server/status), the Flutter dartssh2 transport and ssh_navivox_channel client, and the `dartssh2` dependency. Preserve the HTTP/WS path (`internal/channels/navivox/channel.go` and `flutter-navivox/app/lib/core/gateway/*`). Move `PlatformName = "navivox"` from `protocol.go` into the surviving channel package so deleting protocol.go does not break the gateway import. | - | `internal/channels/navivox/channel_test.go` | Unblocks Navivox VPN host enumeration helper, Navivox HTTP gateway mandatory-VPN bind, Navivox HTTP gateway connect-info command. |
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
