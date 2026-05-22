---
title: "Quickstart"
description: "Install, run offline doctor, configure a provider, and start your first Gormes chat."
aliases:
  - /getting-started/
  - /getting-started/first-run/
  - /using-gormes/
  - /using-gormes/quickstart/
---

# Quickstart

Gormes is a Go-native AI agent runtime: one static binary, no Python, no Docker, no Hermes process. Run it in your terminal as a TUI or as a persistent Telegram, Discord, or Slack gateway. Configuration and state live under `~/.gormes/`. Secrets stay local.

## Fast path

Linux, macOS, WSL2, and Termux:

```bash
curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh
gormes version
gormes doctor --offline
gormes setup
gormes chat
```

Prefer inspect-first? Use [Install](../install/) to download and review `install.sh` or `install.ps1` before running it. In every path, `gormes doctor --offline` is the local proof before credentials.

Native Windows uses PowerShell:

```powershell
irm https://gormes.ai/install.ps1 -OutFile install.ps1
Get-Content .\install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
gormes version
gormes doctor --offline
gormes setup
gormes chat
```

From source (requires Go 1.26+):

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes
./bin/gormes --version
./bin/gormes doctor --offline
```

Verify the installer-published binary is on `PATH` before entering credentials:

```bash
gormes version
gormes doctor --offline
gormes setup
```

For source builds, keep using `./bin/gormes` from the checkout, or add `export PATH="$PWD/bin:$PATH"` temporarily while testing that build.

## Your first chat

The primary setup path is provider-neutral:

```bash
gormes setup
gormes chat
```

During setup, choose your provider. For API-key providers, paste the key when prompted. For OAuth providers, follow the browser flow. The first-run proof stays local and no-stack: no pip, no venv, no Docker daemon, and no provider credentials until after `gormes doctor --offline` passes.

If you already know the provider credential, you can add it directly:

```bash
gormes auth add openai --api-key sk-...
gormes chat
```

Prefer the full terminal UI?

```bash
gormes
```

`gormes auth add` also accepts `--type oauth` for providers that support OAuth (Codex, Anthropic), plus `--label`, `--inference-url`, and credential type overrides. See `gormes auth add --help` for the full flag set.

For scripts, `gormes chat -q "hello from Gormes"` sends one query and exits without starting the TUI.

## Now what?

| | |
|---|---|
| **[Install](../install/)** | Linux/macOS, Windows, and source-build details |
| **[Why Gormes](../why-gormes/#public-comparison-matrix)** | Compare Gormes with Hermes, OpenClaw, and hosted agent services; Gormes is local software, not hosted zero-infrastructure SaaS |
| **[Configure](../configure/)** | `config.toml`, environment variables, providers, Telegram, paths and logs |
| **[Operate](../operate/)** | First chat, Telegram bot, profiles, fallback chains, local Ollama, memory and sessions |
| **[Troubleshoot](../troubleshooting/)** | Doctor, common errors, log locations |
| **[Reference](../reference/)** | CLI commands, config, environment, paths, status, and glossary |
