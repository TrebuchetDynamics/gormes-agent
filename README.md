<p align="center">
  <img src="assets/gormes-agent-logo.png" alt="GORMES-AGENT" width="600">
</p>

<p align="center">
  <strong>AI agents that don't break when the environment changes.</strong><br>
  A single static binary. No Python. No pip. No Docker daemon.
</p>

<p align="center">
  <a href="https://docs.gormes.ai/"><img src="https://img.shields.io/badge/docs-gormes.ai-FFD700?style=flat-square" alt="Docs"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/TrebuchetDynamics/gormes-agent/ci.yml?branch=development&style=flat-square" alt="CI"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent/releases/latest"><img src="https://img.shields.io/github/v/release/TrebuchetDynamics/gormes-agent?style=flat-square" alt="Latest release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent"><img src="https://img.shields.io/github/stars/TrebuchetDynamics/gormes-agent?style=social" alt="Stars"></a>
</p>

---

![Gormes install and first-run onboarding demo](docs/assets/gormes-tui-demo.gif)

Gormes is a Go-native runtime for Hermes-Agent, with upstream Git history preserved for attribution. It keeps the agent, TUI, gateway, tools, sessions, and local memory in one Go binary so setup is a checklist, not a Python environment.

## Install

```bash
curl -fsSLO https://gormes.ai/install.sh && sh install.sh
```

The installer downloads the latest release asset for your machine, publishes the `gormes` command, verifies `gormes version`, runs `gormes doctor --offline`, and prints the exact PATH fix if your shell needs one.

Prefer to inspect first?

```bash
curl -fsSLO https://gormes.ai/install.sh
less install.sh
sh install.sh
```

## First Run

Run these in order:

```bash
gormes doctor --offline
gormes onboard
gormes setup provider
gormes --oneshot "hello"
```

| Command | What it proves |
|---|---|
| `gormes doctor --offline` | Local runtime, TUI, tools, gateway checks, and Goncho memory are wired without using credentials. |
| `gormes onboard` | Shows your config path, skills, agents, channel bindings, missing provider state, and next commands. |
| `gormes setup provider` | Interactively picks endpoint, provider, model, and API key. Secrets go to `~/.gormes/.env`; config goes to `~/.gormes/config.toml`. |
| `gormes --oneshot "hello"` | Sends one provider-backed turn and exits. If this works, the TUI and gateway have a model to use. |

No provider yet? You can still smoke-test the interface:

```bash
gormes --offline
```

## Daily Use

```bash
gormes                     # open the native TUI
gormes dashboard           # web UI at http://127.0.0.1:43827/dashboard
gormes gateway             # run the configured messaging gateway
gormes gateway status      # inspect gateway runtime state
gormes logs                # read recent gateway logs
```

## Multi-Agent Routing

```bash
gormes setup agent         # print agent/profile setup guidance
gormes setup workspace     # set the default workspace path
gormes setup bindings      # route channels to specific agents
gormes onboard             # review the active routes
```

The default agent is ready after install. Add more when you need separate workspaces, models, or channel ownership.

## What You Get

| Surface | Current path |
|---|---|
| Install | Release-first `curl \| sh`, with source fallback and no Python runtime. |
| Setup | `gormes onboard` for state, `gormes setup provider` for the missing model/API key path. |
| Providers | OpenAI-compatible providers, Anthropic, DeepSeek, Groq, Ollama, OpenAI Codex, OpenCode, and custom endpoints. |
| Memory | Goncho local memory and sessions backed by SQLite inside `~/.gormes`. |
| Gateway | One gateway process with Telegram, Discord, and Slack setup paths; additional adapters are tracked in the roadmap. |
| Operations | `doctor`, `config show`, `gateway status`, `logs`, `security audit`, and `secrets audit` commands. |

## Why People Switch

| | Other agents | Gormes |
|---|---|---|
| **Install** | pip, venvs, system packages | `curl \| sh` to one Go binary |
| **First setup** | Find and edit config files | `gormes onboard` then `gormes setup provider` |
| **Smoke test** | Needs a live model first | `gormes doctor --offline` and `gormes --offline` |
| **State** | Redis, vector DBs, sidecars | SQLite under `~/.gormes` |
| **Channels** | Separate bot glue per platform | One gateway process with channel bindings |
| **Release trust** | Ad-hoc local environments | Tagged release assets with SHA-256, SBOMs, and CI validation |

## Operator Cheatsheet

```bash
gormes config show              # redacted config
gormes config edit              # open ~/.gormes/config.toml
gormes auth add <provider>      # add or refresh credentials
gormes security audit --deep    # inspect gateway, tool, and state security
gormes secrets audit --plan file
gormes uninstall                # remove local Gormes artifacts
```

## Build From Source

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes version
./bin/gormes doctor --offline
go test ./... -count=1
go run ./cmd/progress validate
```

## Docs

[Installation](https://docs.gormes.ai/getting-started/installation/) ·
[First run](https://docs.gormes.ai/getting-started/first-run/) ·
[CLI reference](https://docs.gormes.ai/reference/cli/) ·
[Providers](https://docs.gormes.ai/reference/providers/) ·
[Configuration](https://docs.gormes.ai/reference/config/) ·
[Gateway](https://docs.gormes.ai/building-gormes/core-systems/gateway/) ·
[Troubleshooting](https://docs.gormes.ai/getting-started/troubleshooting/) ·
[Roadmap](https://docs.gormes.ai/building-gormes/architecture_plan/)

## Status

Latest public release: [v0.1.01](https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.01). Gormes is early 0.x software: dashboard, gateway, and memory phases are shipped; provider/runtime parity and the learning loop are still moving through the roadmap below.

<!-- PROGRESS:START kind=readme-rollup -->
| Phase | Status | Shipped |
|-------|--------|---------|
| Phase 1 — The Dashboard | ✅ | 4/4 subphases |
| Phase 2 — The Gateway | ✅ | 21/21 subphases |
| Phase 3 — The Black Box (Memory) | ✅ | 15/15 subphases |
| Phase 4 — The Brain Transplant | 🔨 | 9/13 subphases |
| Phase 5 — The Final Purge | 🔨 | 6/22 subphases |
| Phase 6 — The Learning Loop (Soul) | 🔨 | 6/12 subphases |
| Phase 7 — Paused Channel Backlog | 🔨 | 2/5 subphases |
<!-- PROGRESS:END -->

Release v0.1.01 publishes static Go binaries for Linux, macOS, and Windows on amd64/arm64. Local Linux amd64 release smoke: `gormes 0.1.01`, 25 MB on disk. CI runs `go test ./... -count=1`, `go run ./cmd/progress validate`, and `git diff --check`. See [CHANGELOG.md](CHANGELOG.md) and [SECURITY.md](SECURITY.md).

---

Built by [Trebuchet Dynamics](https://trebuchetdynamics.com/).
Hermes Agent lineage by [Nous Research](https://nousresearch.com).
MIT license.
