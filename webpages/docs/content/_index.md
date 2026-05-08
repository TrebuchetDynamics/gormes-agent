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

Choose source build, `install.sh` (Linux/macOS/WSL2), or `install.ps1` (native Windows), prove the machine offline, then add provider and gateway credentials.

## Get started

Use one of the three promoted install paths.

Build from source:

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
export PATH="$PWD/bin:$PATH"
gormes doctor --offline
gormes --offline
```

Or inspect and run `install.sh` on Linux/macOS/WSL2:

```bash
curl -fsSLO https://gormes.ai/install.sh
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
gormes setup provider
gormes --oneshot "hello"
gormes gateway status
```

## What is Gormes?

Gormes is the Go-native rewrite of the Hermes-Agent architecture. It does not require a running Hermes backend for the native smoke path, and it keeps the operator surface inspectable: one release binary, local files, explicit diagnostics, and source-backed release evidence.

Who is it for? Operators, developers, and agent builders who want a local runtime that can keep working across restarts, machines, and flaky networks.

What makes it different?

- **Go-native runtime:** native TUI, doctor, onboard/setup, provider turns, tools, memory, dashboard, logs, audits, and configured gateways from one binary.
- **Offline proof path:** `gormes --offline` and `gormes doctor --offline` work before credentials, network calls, or token spend.
- **Three install paths:** source build for maximum inspection, source-backed `install.sh` (Linux/macOS/WSL2), or `install.ps1` (native Windows) for a managed checkout that publishes the `gormes` command.
- **Local SQLite memory ("Goncho"):** sessions and durable context stay local.
- **Roadmap honesty:** Hermes parity, broad channel parity, voice/TTS, MCP/plugin parity, and release hardening stay visible as active work instead of shipped promises.

## What you can do today

- Run a local agent UI with zero runtime dependencies on the offline path.
- Send one-shot prompts to a provider-compatible endpoint.
- Validate your environment before spending tokens.
- Run onboard/setup flows that surface config, providers, skills, agents, and channel bindings.
- Operate configured Telegram, Discord, or Slack agents from one binary.
- Inspect and debug agent memory locally with Goncho.
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

- Source build and inspectable `install.sh` are the two promoted scout-release paths.
- Offline doctor runs before provider credentials or token spend.
- Secrets stay local under the Gormes home.
- `install.sh` clones or updates a managed source checkout, builds `gormes`, verifies the command, and can hand off to setup.
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
| **[Getting Started](getting-started/)** | Build or install Gormes, run doctor, and complete a local smoke test |
| **[First Run](getting-started/first-run/)** | Run offline diagnostics, configure a provider, and complete the first model turn |
| **[Configuration](getting-started/configuration/)** | Native `GORMES_HOME`, `config.toml`, `.env`, and runtime paths |
| **[Provider Setup](guides/provider-setup/)** | Configure provider/model credentials and understand readiness labels |
| **[Gateway Operations](guides/gateway-operations/)** | Run and inspect persistent messaging channels |
| **[Telegram Bot](guides/telegram-bot/)** | Configure and debug the runtime-ready Telegram channel |
| **[CLI Reference](reference/cli/)** | Current top-level commands and operator subcommands |
| **[Config Reference](reference/config/)** | Native config sections, setup commands, and credential storage |
| **[Environment](reference/environment/)** | Environment variables for config, providers, gateway, and install |
| **[Providers](reference/providers/)** | Provider support taxonomy and upstream Hermes parity context |
| **[Architecture](architecture/)** | Runtime model, gateway pipeline, tool execution, and memory boundaries |
| **[Roadmap & Parity](parity/)** | Current status, known gaps, and Hermes/Honcho compatibility tracking |

## Learn more

- [Operate Gormes](guides/) for provider setup, gateway operations, web tools, and debugging.
- [Reference](reference/) for exact CLI, config, environment, provider, path, and log details.
- [Building Gormes](building-gormes/) for implementation plans, progress, and parity evidence.
