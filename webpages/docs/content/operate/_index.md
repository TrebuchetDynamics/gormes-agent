---
title: "Operate"
description: "Run Gormes day to day: local chat, profiles, memory and sessions, gateways, routing, fallback providers, and dashboards."
aliases:
  - /guides/
  - /guides/what-you-can-do/
---

Operate covers the workflows after Gormes is installed and configured. Use this
section for local terminal work, server-side gateways, profile isolation,
provider fallback, and runtime inspection.

## Local terminal workflow

| Task | Use it for |
|---|---|
| [First chat](first-chat/) | Connect a provider and run a first turn. |
| [Local Ollama](local-ollama/) | Run Gormes against a local model endpoint. |
| [Profiles for client work](profiles-client-work/) | Keep projects, clients, and workspaces separated. |
| [Memory and sessions](memory-sessions/) | Inspect sessions, persisted memory, and local state. |

## Server and gateway workflow

| Task | Use it for |
|---|---|
| [Telegram bot](telegram-bot/) | Run a Telegram-facing Gormes agent. |
| [Multi-channel gateway](multi-channel-gateway/) | Run configured Telegram, Discord, and Slack gateways from one runtime. |
| [Channel bindings](channel-bindings/) | Route channels, users, profiles, agents, and workspaces intentionally. |
| [Fallback provider chain](fallback-providers/) | Keep runtime turns resilient when a provider fails. |
| [Dashboard, status, and logs](dashboard-status-logs/) | Inspect the runtime while it is running. |

## Related setup

- [Configure providers](../configure/providers/) before provider-backed chat.
- [Configure channel credentials](../configure/telegram/) before gateway work.
- [Troubleshoot doctor output](../troubleshooting/doctor/) when a runtime check fails.
