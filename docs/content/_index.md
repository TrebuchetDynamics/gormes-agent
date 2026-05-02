---
title: "Gormes Documentation"
description: "Install, configure, operate, and extend the Go-native Gormes runtime."
weight: 0
slug: "/"
---

# Gormes

Gormes is a Go-native agent runtime for local TUI work, offline diagnostics, provider-backed turns, tools, Goncho memory, dashboard inspection, and configured chat gateways.

## Get started

Build from source, verify the local stack, then open the offline TUI:

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes doctor --offline
./bin/gormes --offline
```

## What is Gormes?

Gormes is the Go-native rewrite of the Hermes-Agent architecture. It does not require a running Hermes backend for the native smoke path, and it keeps the operator surface inspectable: one release binary, local files, explicit diagnostics, and source-backed release evidence.

Who is it for? Operators, developers, and agent builders who want a local runtime that can keep working across restarts, machines, and flaky networks.

What makes it different?

- **Go-native runtime:** native TUI, doctor, provider turns, tools, memory, dashboard, and configured gateways from one binary.
- **Offline proof path:** `./bin/gormes --offline` and `./bin/gormes doctor --offline` work before credentials, network calls, or token spend.
- **Local memory:** Goncho keeps sessions and durable context in local SQLite.
- **Roadmap honesty:** Hermes parity, broad channel parity, voice/TTS, MCP/plugin parity, and release hardening stay visible as active work instead of shipped promises.

## How it works

The Gormes runtime is the local source of truth for TUI sessions, provider-backed turns, tools, Goncho memory, dashboard views, and configured gateway status. Operator docs separate what is ready today from architecture and parity pages that track deeper Hermes/Honcho compatibility.

## What lives here?

This docs surface is the operator and developer manual for Gormes. It starts with install, first run, configuration, gateway operation, and troubleshooting. Architecture and parity pages explain how Gormes stays compatible with Hermes without making roadmap work look like shipped runtime behavior.

## Start here

| | |
|---|---|
| **[Getting Started](getting-started/)** | Build or install Gormes, run doctor, and complete a local smoke test |
| **[Configuration](getting-started/configuration/)** | Native `GORMES_HOME`, `config.toml`, `.env`, and runtime paths |
| **[Gateway Operations](guides/gateway-operations/)** | Run and inspect persistent messaging channels |
| **[CLI Reference](reference/cli/)** | Current top-level commands and operator subcommands |
| **[Architecture](architecture/)** | Runtime model, gateway pipeline, tool execution, and memory boundaries |
| **[Roadmap & Parity](parity/)** | Current status, known gaps, and Hermes/Honcho compatibility tracking |

## Learn more

- [Operate Gormes](guides/) for provider setup, gateway operations, web tools, and debugging.
- [Reference](reference/) for exact CLI, config, environment, provider, path, and log details.
- [Building Gormes](building-gormes/) for implementation plans, progress, and parity evidence.
