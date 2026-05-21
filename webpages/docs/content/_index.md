---
title: "Gormes Documentation"
description: "Install, configure, operate, and extend the Go-native Gormes runtime."
weight: 0
slug: "/"
---

Install, configure, and operate Gormes from one Go-native runtime.

Use the fast path when you want first success. Use the section map when you
need platform details, gateway workflows, exact reference, or contributor
evidence.

## Fast path

Linux, macOS, WSL2, and Termux:

```bash
curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh
gormes doctor --offline
gormes setup
gormes chat
```

Prefer another path?

| Path | Start here |
|---|---|
| Windows PowerShell | [Install on Windows](install/windows/) |
| Termux on Android | [Install on Termux](install/termux/) |
| Inspect-first Unix install | [Linux and macOS install](install/linux-macos/) |
| Source build | [Build from source](install/from-source/) |

Prefer the full terminal UI? Run `gormes` after setup. Need a scriptable
one-shot turn? Use `gormes chat -q "hello from Gormes"`.

## Choose your path

| I want to... | Go to |
|---|---|
| Install and run my first turn | [Quickstart](start-here/) |
| Understand supported install paths | [Install](install/) |
| Configure providers, models, agents, workspaces, profiles, and bindings from the CLI | [Configure](configure/) |
| Run local chat, gateways, profiles, memory, and dashboards | [Operate](operate/) |
| Separate state by named profile | [Profiles and workspaces](configure/profiles-workspaces/) |
| Diagnose broken state | [Troubleshoot](troubleshooting/) |
| Look up commands, config, env vars, paths, status, or terms | [Reference](reference/) |
| Understand the runtime model | [Concepts](concepts/) |
| Contribute to Gormes itself | [Build Gormes](building-gormes/) |
| Read upstream/source background | [Archive & Research](archive/) |

## Available now

- Runtime-ready: CLI and offline TUI
- Runtime-ready: provider-backed chat
- Runtime-ready: SQLite memory and sessions
- Runtime-ready: local dashboard
- Runtime-ready: Telegram, Discord, and Slack gateways

## Expanding next

- More Hermes compatibility
- Voice/TTS
- MCP/plugin support
- Package-manager installs
- More gateways

See [Status & Roadmap](reference/status-readiness/) for the public readiness
view. Deep implementation progress remains under [Build Gormes](building-gormes/).

## How these docs are organized

The docs separate public readiness from implementation progress.

| Area | Audience | Use it for |
|---|---|---|
| **Quickstart, Install, Configure, Operate, Troubleshoot** | Users and operators | First success, setup, day-two workflows, and diagnosis. |
| **Reference** | Operators and contributors | Exact commands, config keys, env vars, paths, status labels, and glossary terms. |
| **Concepts** | Operators and builders | Stable mental models for runtime, memory, tools, gateways, and Hermes compatibility. |
| **Build Gormes** | Contributors and agents | Roadmap, parity workflow, progress rows, generated queues, and architecture evidence. |
| **Archive & Research** | Researchers and maintainers | Upstream Hermes docs and agent-system research; not Gormes setup instructions. |
