---
title: "Gormes Documentation"
description: "Install, configure, operate, and extend the Go-native Gormes runtime."
weight: 0
slug: "/"
---

<p align="center">
  <img src="/gormes-agent-logo-blue.svg" alt="GORMES-AGENT" width="720">
</p>

# Gormes

Gormes runs AI agents from one Go-native runtime.

No Python runtime. No virtualenv repair. No backend service just to open the UI.

Choose source build, release-first `install.sh` (Linux/macOS/WSL2), or release-first `install.ps1` (native Windows), prove the machine offline, then add provider and gateway credentials.

## Get started

Use one of the three promoted install paths.

Build from source:

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes
./bin/gormes doctor --offline
./bin/gormes --offline
```

Or inspect and run `install.sh` on Linux/macOS/WSL2:

```bash
curl -fsSLO https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh
less install.sh
sh install.sh
gormes doctor --offline
```

Or inspect and run `install.ps1` on native Windows:

```powershell
irm https://gormes.ai/install.ps1 -OutFile install.ps1
Get-Content .\install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
gormes doctor --offline
```

After any path, add only what you need:

```bash
gormes onboard
gormes setup provider
gormes --oneshot "hello"
gormes config show
gormes gateway status
```

## What is Gormes?

Gormes is the Go-native rewrite of the Hermes-Agent architecture. It does not require a running Hermes backend for the native smoke path, and it keeps the operator surface inspectable: one release binary, local files, explicit diagnostics, and source-backed release evidence.

Who is it for? Operators, developers, and agent builders who want a local runtime that can keep working across restarts, machines, and flaky networks.

What makes it different?

- **Go-native runtime:** native TUI, doctor, onboard/setup, provider turns, tools, memory, dashboard, logs, audits, and configured gateways from one binary.
- **Offline proof path:** `gormes --offline` and `gormes doctor --offline` work before credentials, network calls, or token spend.
- **Three install paths:** source build for maximum inspection, release-first `install.sh` (Linux/macOS/WSL2), or release-first `install.ps1` (native Windows) for a managed install that publishes the `gormes` command.
- **Local SQLite memory ("Goncho"):** sessions and durable context stay local.
- **Roadmap honesty:** Hermes parity, broad channel parity, voice/TTS, MCP/plugin parity, and release hardening stay visible as active work instead of shipped promises.

## What you can do today

- Run a local agent UI with zero runtime dependencies on the offline path: `gormes --offline`.
- Send one-shot prompts to a provider-compatible endpoint: `gormes --oneshot "..."`.
- Validate your environment before spending tokens: `gormes doctor --offline`, `gormes onboard`.
- Configure providers, models, agents, workspaces, and bindings from the CLI: `gormes setup [section]`.
- Inspect and edit the native config: `gormes config show`, `gormes config get`, `gormes config set`, `gormes config check`.
- Operate configured Telegram, Discord, or Slack agents from one binary: `gormes gateway`, `gormes gateway status`, `gormes gateway reload`.
- Isolate work by named profile: `gormes profile create`, `gormes profile use`, `gormes --profile <name>`.
- Inspect memory, sessions, skills, logs, security, and durable task boards from local operator commands.
- Browse sessions, config, skills, logs, and audits in local operator surfaces.

## Support labels

The docs use support labels so roadmap work does not look like shipped runtime behavior:

| Label | Meaning |
|---|---|
| **Runtime-ready** | Covered by current Go runtime paths and promoted for configured scout-release use. Telegram, Discord, and Slack are in this group. |
| **Adapter present / needs live validation** | Go code or fixtures exist, but the path is not promoted as a default user-ready channel yet. |
| **Planned** | Tracked in the roadmap or progress file, but not user-ready. |
| **Experimental or internal** | Kept for contributor context, not public setup guidance. |

## Trust posture

- Source build and release-first inspectable installers are the promoted scout-release paths.
- Offline doctor runs before provider credentials or token spend.
- Secrets stay local under the Gormes home.
- `install.sh` fetches the latest release binary, verifies its SHA-256, publishes `gormes`, and falls back to a managed source build when needed.
- Tagged artifacts carry checksums; release signing and package-manager hardening are still in progress.
- Progress data is generated from the canonical `progress.json` source instead of hand-edited marketing copy.

## How it works

The Gormes binary is the local source of truth for TUI sessions, provider turns, tools, local SQLite memory, dashboard views, and configured gateway status. Operator docs separate what is ready today from architecture and parity pages that track deeper Hermes/Honcho compatibility.

## What lives here?

This docs surface is the operator and developer manual for Gormes. It starts with install, first run, configuration, gateway operation, and troubleshooting. Architecture and parity pages explain how Gormes stays compatible with Hermes without making roadmap work look like shipped runtime behavior.

| Section | Audience | Use it for |
|---|---|---|
| **Getting Started**, **Operate**, **Using Gormes**, **Reference** | Users and operators | Install, first run, provider setup, gateway operation, config, CLI, environment, and troubleshooting. |
| **Architecture**, **Development**, **Parity**, **Building Gormes** | Contributors | Runtime boundaries, parity workflow, implementation progress, and source-backed roadmap rows. |
| **Upstream Hermes Archive** and **Papers** | Researchers and maintainers | Historical/source context. These pages are not Gormes setup instructions and may describe upstream-only behavior. |

## Start here

| | |
|---|---|
| **[Start here](start-here/)** | Install, authenticate, and run your first turn in under two minutes. |
| **[Install](install/)** | Pick a path: `install.sh`, `install.ps1`, or build from source. |
| **[Configure](configure/)** | `GORMES_HOME`, `config.toml`, env vars, providers, Telegram, and paths. |
| **[CLI reference](cli/)** | One page per top-level command, audited against the binary. |
| **[Recipes](recipes/)** | Copy-paste walkthroughs for the most common Gormes tasks. |
| **[Troubleshooting](troubleshooting/)** | Doctor, common errors, and log locations. |

## Learn more

- [Why Gormes](why-gormes/) for the operational philosophy behind the Go-native runtime.
- [Building Gormes](building-gormes/) for implementation plans, progress, and parity evidence.
