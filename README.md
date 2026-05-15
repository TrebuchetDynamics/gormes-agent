<p align="center">
  <img src="assets/gormes-agent-logo-blue.svg" alt="GORMES-AGENT" width="720">
</p>

<p align="center">
  <strong>Run Hermes-compatible agents from one Go binary.</strong><br>
  Gormes carries the 30 most-used Hermes skills into a Go-native runtime that runs on Termux, Windows-without-Python, and locked-down corp Linux — no pip, no venv, no Docker daemon.
</p>

<p align="center">
  <a href="https://docs.gormes.ai/"><img src="https://img.shields.io/badge/docs-gormes.ai-FFD700?style=flat-square" alt="Docs"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/TrebuchetDynamics/gormes-agent/ci.yml?branch=development&style=flat-square" alt="CI"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent/releases/latest"><img src="https://img.shields.io/github/v/release/TrebuchetDynamics/gormes-agent?style=flat-square" alt="Latest release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent"><img src="https://img.shields.io/github/stars/TrebuchetDynamics/gormes-agent?style=social" alt="Stars"></a>
</p>

---

<p align="center">
  <img src="webpages/docs/assets/gormes-tui-demo.gif" alt="Gormes install, setup, provider setup, first task, web tools, Termux, and gateway demo" width="960">
</p>

Install once, run `gormes setup`, configure a provider, and open chat from a normal terminal.

## Quick Install

**Linux, macOS, WSL2:**

```bash
curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh
```

**Windows (native PowerShell):**

```powershell
irm https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 | iex
```

> **Inspect first:** download the script, read it, then run it. Both installers are user-scoped — no root or admin paths.

After installation:

```bash
gormes setup
gormes chat
```

## First Setup

```bash
gormes setup                 # guided setup for provider, model, terminal, gateway, tools
gormes setup provider        # direct provider setup shortcut
gormes chat                  # provider-backed terminal chat
gormes doctor --offline      # local runtime, TUI, gateway, memory — no credentials needed
gormes --offline             # native TUI, no network
```

If `gormes chat` opens, the TUI and gateway have a model to use.

## What Works Today

| Surface | Status |
|---|---|
| Install, offline smoke, doctor, dashboard | **Supported** |
| CLI-first setup/config, TUI, scripted chat, multi-agent routing | **Supported** |
| Providers: OpenAI, Anthropic, DeepSeek, Groq, Ollama, OpenAI Codex, OpenCode, custom endpoints | **Supported** |
| Local SQLite memory (Goncho), session state | **Supported** |
| Gateways: Telegram, Discord, Slack | **Supported** |
| Profiles, local Kanban board, skills/plugins inventory, security/secret audits | **Supported** |
| Hermes / OpenClaw migration with dry-run | **Supported** |
| Gateways: WhatsApp, Teams, Yuanbao | **Experimental** |
| Learning loop, MCP/plugin parity, voice/TTS | **Roadmap** |
| Release signing, package-manager lanes | **Roadmap** |

Full Hermes-parity status by phase lives in the [roadmap](https://docs.gormes.ai/building-gormes/architecture_plan/).

## Why People Switch

| | Other agents | Gormes |
|---|---|---|
| **Install** | pip, venvs, system packages | One Go command from `install.sh` |
| **First setup** | Find and edit config files | `gormes setup` |
| **Smoke test** | Needs a live model first | `gormes doctor --offline` and `gormes --offline` |
| **State** | Redis, vector DBs, sidecars | SQLite under `~/.gormes` |
| **Channels** | Separate bot glue per platform | One gateway process with channel bindings |
| **Release trust** | Ad-hoc local environments | Tagged release assets with SHA-256 + SBOMs |

## Daily Use

```bash
gormes                          # open the native TUI
gormes setup                    # guided setup and reconfiguration
gormes setup --quick            # fill missing setup items only
gormes dashboard                # web UI at http://127.0.0.1:43827/dashboard
gormes config show              # inspect config with secrets redacted
gormes profile use <name>       # switch isolated profile homes
gormes gateway                  # run the configured messaging gateway
gormes gateway status --json    # inspect gateway runtime state
gormes logs                     # read recent gateway logs
```

### CLI vs Gateway

| Action | Local CLI | Messaging gateway |
|---|---|---|
| Start chatting | `gormes` | Run `gormes gateway`, then message the configured channel bot. |
| Start provider-backed chat | `gormes chat` | Send a normal message after provider setup. |
| Configure provider/model | `gormes setup provider` | Run setup locally, then `gormes gateway reload`. |
| Diagnose runtime issues | `gormes doctor --offline`, `gormes logs` | `gormes gateway status`, then fix config or credentials locally. |

## Migrating From Hermes Or OpenClaw

Always dry-run first and review the redacted manifest before applying.

```bash
gormes migrate hermes   --dry-run --source ~/.hermes
gormes migrate hermes   --yes     --source ~/.hermes

gormes migrate openclaw --dry-run --source ~/.openclaw
gormes migrate openclaw --yes     --source ~/.openclaw
gormes migrate openclaw --yes --secrets --source ~/.openclaw
```

OpenClaw cleanup archives old directories rather than deleting them:

```bash
gormes migrate openclaw cleanup --dry-run
gormes migrate openclaw cleanup --yes
```

## Build From Source

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
export PATH="$PWD/bin:$PATH"
gormes doctor --offline
```

## Docs

| Section | What it covers |
|---|---|
| [Installation](https://docs.gormes.ai/getting-started/installation/) | Source build, installer path, PATH, and offline verification. |
| [First run](https://docs.gormes.ai/getting-started/first-run/) | `doctor`, `setup`, provider setup, and first provider-backed turn. |
| [What you can do](https://docs.gormes.ai/guides/what-you-can-do/) | Outcome-driven recipes for CLI, config, gateway, profiles, memory, and security. |
| [CLI reference](https://docs.gormes.ai/reference/cli/) | Commands, flags, and operator workflows. |
| [Providers](https://docs.gormes.ai/reference/providers/) | Supported provider config and credential paths. |
| [Configuration](https://docs.gormes.ai/reference/config/) | `~/.gormes/config.toml`, `.env`, agents, workspaces, and bindings. |
| [Gateway](https://docs.gormes.ai/building-gormes/core-systems/gateway/) | Channel runtime, status checks, reloads, and troubleshooting evidence. |
| [Troubleshooting](https://docs.gormes.ai/getting-started/troubleshooting/) | Common install, provider, gateway, and browser-tool failures. |
| [Roadmap](https://docs.gormes.ai/building-gormes/architecture_plan/) | Hermes parity phases, status labels, and progress evidence. |

## Security & Trust

- `gormes doctor --offline` and `gormes --offline` prove local readiness before any token spend.
- Provider secrets stay local under `~/.gormes/.env`; config under `~/.gormes/config.toml`.
- Tagged releases publish a single static binary per target plus SHA-256 checksums and SBOMs (release signing and package-manager lanes are still hardening).
- The `curl … | sh` install path is validated by an end-to-end suite ([`tests/install/e2e.sh`](tests/install/e2e.sh)) covering API outage with redirect fallback, SHA-256 mismatch abort, SSH-origin update fallback to public HTTPS, hosts without Go/curl/wget/systemd, Termux detection, sudo'd root install, and `--uninstall --dry-run` preview. Runnable locally or via the [`install-e2e`](.github/workflows/install-e2e.yml) workflow on demand.

## How It's Built

Gormes is the artifact of an autonomous-porting methodology — a validation-gated planner → builder → TDD-slice loop that ports Hermes to Go one bounded vertical at a time. The runtime is the product; the methodology is the supporting evidence.

- Strategy: [Gormes Success Plan](docs/content/building-gormes/strategy/success-plan.md)
- Engineering blog: [TrebuchetDynamics Engineering](https://engineering.trebuchetdynamics.com/) ([RSS](https://engineering.trebuchetdynamics.com/feed.xml))
- Differentiator: [v1.0 differentiator](docs/content/building-gormes/strategy/v1-differentiator.md)
- Toolkit extraction (`agentic-porting-kit`): tracked as a Phase 8 row; until that public repo exists, the [repo-local development skills](docs/development-skills/) are the inspectable placeholder.

Hermes-Agent, with upstream Git history preserved for attribution, remains the parity oracle.

## Status

Latest public release: [v0.2.11](https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.2.11).

CI runs `go test ./... -count=1`, `go run ./cmd/progress validate`, and `git diff --check`. The single static binary ships for Linux, macOS, Windows, and Termux/Android. The current Linux build measures ~45.3 MB (`benchmarks.json`). WASI Whisper tiny.en runs at 3.78x realtime (`benchmarks.json`, 2026-05-10). Offline doctor peaks at ~25.1 MB RSS (`benchmarks.json`, 2026-05-15).

<details>
<summary>Roadmap phase rollup</summary>

<!-- PROGRESS:START kind=readme-rollup -->
| Phase | Status | Shipped |
|-------|--------|---------|
| Phase 1 — The Dashboard | 🔨 | 5/6 subphases |
| Phase 2 — The Gateway | 🔨 | 21/22 subphases |
| Phase 3 — The Black Box (Memory) | ✅ | 15/15 subphases |
| Phase 4 — The Brain Transplant | ✅ | 13/13 subphases |
| Phase 5 — The Final Purge | 🔨 | 22/23 subphases |
| Phase 6 — The Learning Loop (Soul) | 🔨 | 9/12 subphases |
| Phase 7 — Paused Channel Backlog | ✅ | 5/5 subphases |
| Phase 8 — Reputation & Publication | 🔨 | 3/7 subphases |
| Phase 9 — Design & Security Hardening | 🔨 | 5/6 subphases |
<!-- PROGRESS:END -->

</details>

See [CHANGELOG.md](CHANGELOG.md) and [SECURITY.md](SECURITY.md).

---

Built by [Trebuchet Dynamics](https://trebuchetdynamics.com/).
Hermes Agent lineage by [Nous Research](https://nousresearch.com).
MIT license.
