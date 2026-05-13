---
title: "Start here"
description: "Install, authenticate, and run your first Gormes turn in under two minutes."
aliases:
  - /getting-started/
  - /getting-started/first-run/
  - /using-gormes/
  - /using-gormes/quickstart/
---

# Start here

Gormes is a Go-native AI agent runtime: one static binary, no Python, no Docker, no Hermes process. Run it in your terminal as a TUI or as a persistent Telegram, Discord, or Slack gateway. Configuration and state live under `~/.gormes/`. Secrets stay local.

## 60-second install

Pick one. Full details for each path live in [Install](../install/).

Linux, macOS, WSL2:

```bash
curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh
less install.sh
sh install.sh
```

Native Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 -OutFile install.ps1
Get-Content .\install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

From source (requires Go 1.26+):

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
export PATH="$PWD/bin:$PATH"
```

Verify the binary is on `PATH`:

```bash
gormes version
gormes doctor --offline
```

## Your first turn

Add a provider credential, then send a prompt.

```bash
gormes auth add openai --api-key sk-...
gormes --oneshot "hello from Gormes"
```

Or open the interactive TUI:

```bash
gormes
```

`gormes auth add` also accepts `--type oauth` for providers that support OAuth (Codex, Anthropic), plus `--label`, `--inference-url`, and credential type overrides. See `gormes auth add --help` for the full flag set.

`--oneshot` (alias `-z`) sends a single prompt and exits without starting the TUI. Provider, model, endpoint, and API key can each be overridden for a single invocation via `--provider`, `--model`, `--endpoint`, and `--api-key`.

## Now what?

| | |
|---|---|
| **[Install](../install/)** | Linux/macOS, Windows, and source-build details |
| **[Configure](../configure/)** | `config.toml`, environment variables, providers, Telegram, paths and logs |
| **[CLI reference](../cli/)** | Every top-level command and subcommand |
| **[Recipes](../recipes/)** | First turn, Telegram bot, profiles, fallback chains, local Ollama |
| **[Troubleshooting](../troubleshooting/)** | Doctor, common errors, log locations |
