---
title: "Status & Roadmap"
description: "Public readiness labels for Gormes capabilities and where to find deeper implementation evidence."
aliases:
  - /roadmap/
  - /status/
  - /reference/status/
---

This page is the public readiness view: it answers what you can use today and
what is still expanding. Contributor implementation progress lives under
[Build Gormes](../../building-gormes/).

## Readiness labels

| Label | Meaning |
|---|---|
| **Runtime-ready** | Promoted for configured local or server-side use in the current Go runtime. |
| **Adapter present / needs live validation** | Code or fixtures exist, but the path is not promoted as a default user-ready workflow. |
| **Planned** | Tracked in roadmap/progress docs, but not user-ready. |
| **Experimental or internal** | Useful for contributor context, not public setup guidance. |

## Available now

| Capability | Readiness | Where to start |
|---|---|---|
| CLI and offline TUI | Runtime-ready | [Quickstart](../../start-here/) |
| Provider-backed chat | Runtime-ready | [Configure providers](../../configure/providers/) |
| SQLite memory and sessions | Runtime-ready | [Memory and sessions](../../operate/memory-sessions/) |
| Local dashboard | Runtime-ready | [Dashboard, status, and logs](../../operate/dashboard-status-logs/) |
| Telegram, Discord, and Slack gateways | Runtime-ready | [Operate gateways](../../operate/) |

## Expanding next

| Capability | Readiness | Evidence |
|---|---|---|
| More Hermes compatibility | Planned | [Build Gormes](../../building-gormes/) |
| Voice/TTS | Planned | [Implementation roadmap](../../building-gormes/implementation-roadmap/) |
| MCP/plugin support | Planned | [CLI reference](../../cli/) and [Build Gormes](../../building-gormes/) |
| Package-manager installs | Planned | [Install](../../install/) |
| More gateways | Planned | [Gateway donor map](../../building-gormes/gateway-donor-map/) |
| NaviWogs/NaviVox control plane | Experimental or internal | [Build Gormes](../../building-gormes/) |

## Implementation evidence

Use [Build Gormes](../../building-gormes/) for progress rows, parity workflow,
builder queue, generated architecture archives, and contributor handoff
material. Use this page when you need public readiness without the deep
implementation inventory.
