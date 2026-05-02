<p align="center">
  <img src="assets/gormes-agent-logo.png" alt="GORMES-AGENT" width="600">
</p>

# GORMES-AGENT

<p align="center">
  <strong>AI agents that do not break because Python broke.</strong><br>
  One single static binary from a Go-native runtime. No virtualenv repair. No backend service just to open the UI.
</p>

<p align="center">
  <a href="https://docs.gormes.ai/"><img src="https://img.shields.io/badge/docs-gormes.ai-FFD700?style=flat-square" alt="Docs"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/TrebuchetDynamics/gormes-agent/ci.yml?branch=development&style=flat-square" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"></a>
</p>

---

![Gormes native TUI running offline](docs/assets/gormes-tui-demo.gif)

The offline TUI starts locally with no API key, no network call, no Python,
no Node, no Docker, and no Hermes process.

## Quick Start

Build the source tree you inspected:

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes --offline
./bin/gormes doctor --offline
```

Expected result: a native terminal UI opens, then `doctor --offline` reports
local readiness checks without contacting a model provider.

## What You Get

```bash
./bin/gormes                         # interactive TUI
./bin/gormes --oneshot "hello"       # one provider-backed turn
./bin/gormes doctor --offline        # local diagnostics before token spend
./bin/gormes dashboard               # local htmx web dashboard
./bin/gormes gateway                 # configured Telegram/Discord/Slack paths
./bin/gormes goncho doctor --json    # local SQLite memory diagnostics
```

## First Provider Turn

After the offline proof works:

```bash
./bin/gormes setup provider
./bin/gormes setup model
./bin/gormes --oneshot "Summarize this repo in one sentence"
```

API keys live in Gormes-owned state such as `~/.gormes/.env`, not in upstream
Hermes state.

## Why People Switch

| | Python-stack agents | Gormes |
|---|---|---|
| **Startup** | Runtime setup first | `./bin/gormes --offline` |
| **Deploy** | Virtualenvs, Docker, sidecars | One static Go artifact |
| **Memory** | Redis/vector DB/service stores | Goncho SQLite in the binary |
| **Diagnostics** | Spread across layers | Built-in `gormes doctor` |
| **Channels** | Separate runtime surfaces | Configured Telegram/Discord; Slack with complete credentials |

Gormes is built for agents that need to stay running across restarts,
machines, and flaky networks. Start offline. Prove the machine works. Add
provider and gateway credentials later.

## Current State

Status: early-stage 0.x. Useful today, not production-stable.

Shipping now:

- Native CLI and Bubble Tea TUI, including the offline smoke path.
- Provider-compatible one-shots and TUI startup paths.
- `doctor --offline` for local tools, Goncho, gateways, Slack, and provider setup.
- Configured Telegram and Discord gateways; Slack when Socket Mode credentials are complete.
- Goncho memory on local SQLite, session diagnostics, and an htmx dashboard.
- Web/browser/search tool surfaces with visible unavailable evidence when backends are missing.

Still in progress:

- Full Hermes parity and production hardening.
- Broad channel parity beyond Telegram, Discord, and Slack.
- Voice/TTS/transcription, MCP/plugin parity, and signed/package-manager releases.

<!-- PROGRESS:START kind=readme-rollup -->
| Phase | Status | Shipped |
|-------|--------|---------|
| Phase 1 — The Dashboard | ✅ | 4/4 subphases |
| Phase 2 — The Gateway | ✅ | 21/21 subphases |
| Phase 3 — The Black Box (Memory) | ✅ | 15/15 subphases |
| Phase 4 — The Brain Transplant | 🔨 | 7/13 subphases |
| Phase 5 — The Final Purge | 🔨 | 5/22 subphases |
| Phase 6 — The Learning Loop (Soul) | 🔨 | 6/12 subphases |
| Phase 7 — Paused Channel Backlog | 🔨 | 2/5 subphases |
<!-- PROGRESS:END -->

## Install

Source-backed convenience installer:

```bash
curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.sh
less install.sh
sh install.sh
gormes --offline
gormes doctor --offline
```

Windows PowerShell:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 -OutFile install.ps1
Get-Content .\install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
gormes --offline
gormes doctor --offline
```

Prebuilt release archives exist, but source builds remain the primary trust
path until signed stable artifacts and package-manager distribution are hardened.

## Docs

[Setup guide](https://docs.gormes.ai/getting-started/first-run/) |
[CLI reference](https://docs.gormes.ai/reference/cli/) |
[Configuration](https://docs.gormes.ai/reference/config/) |
[Gateway](https://docs.gormes.ai/building-gormes/core-systems/gateway/) |
[Roadmap](https://docs.gormes.ai/building-gormes/architecture_plan/) |
[Security](SECURITY.md)

## Contributing

```bash
make build
make test
go run ./cmd/progress validate
git diff --check
```

Built by [Trebuchet Dynamics](https://trebuchetdynamics.com/).
Original lineage: Hermes-Agent, with upstream Git history preserved for attribution,
by [Nous Research](https://nousresearch.com).
License: MIT.
