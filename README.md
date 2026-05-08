<p align="center">
  <img src="assets/gormes-agent-logo-blue.svg" alt="GORMES-AGENT" width="720">
</p>

<p align="center">
  <strong>AI agents that don't break when your environment does.</strong><br>
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

![Gormes install and first-run onboarding demo](webpages/docs/assets/gormes-tui-demo.gif)

`gormes --offline` starts locally with no API key, no network calls, and no Python runtime.

Gormes is a Go-native runtime for AI agents, packaged as a single Go binary. It keeps the TUI, provider turns, local memory, gateways, diagnostics, and setup flows behind one command, so setup is a checklist instead of a Python environment.

Gormes is a Go-native rewrite of Hermes-Agent, with upstream Git history preserved for attribution and Hermes-compatible agent behavior carried forward in Go.

## Capability Map

Status labels are intentional: `runtime-ready` means verified in the current
Gormes runtime; `fixture-backed` means covered by local tests or fake clients;
`row-backed` means tracked in the roadmap without a live support promise yet.

| Surface | Public status | Current path |
|---|---|---|
| Install and offline smoke | `runtime-ready` | Source build or inspectable `install.sh`, then `gormes doctor --offline` and `gormes --offline`. |
| TUI and one-shot turns | `runtime-ready` | `gormes` opens the native TUI; `gormes --oneshot "hello"` sends a provider-backed turn. |
| Provider setup | `runtime-ready` | OpenAI-compatible endpoints plus OpenAI, Anthropic, DeepSeek, Groq, Ollama, OpenAI Codex, OpenCode, and custom endpoints. |
| Memory and sessions | `runtime-ready` | Local SQLite memory ("Goncho") and session state live under `~/.gormes`. |
| Gateway channels | `runtime-ready` for Telegram, Discord, and Slack | One gateway process with channel bindings; WhatsApp and long-tail adapters stay explicitly tracked until live runtime validation is complete. |
| Dashboard and operations | `runtime-ready` | `dashboard`, `doctor`, `config show`, `gateway status`, `gateway reload`, `logs`, `security audit`, and `secrets audit`. |
| Hermes/OpenClaw migration | `runtime-ready` with explicit source paths | Dry-run first, then apply with `--yes`; OpenClaw secret import requires explicit `--secrets`. |
| Learning loop, MCP/plugin parity, voice/TTS | `row-backed` / in progress | Implemented slice-by-slice through the public roadmap and progress gates. |

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | bash
```

Works on Linux, macOS, and WSL2. The installer clones a managed checkout, builds the `gormes` command from source, verifies `gormes version`, runs `gormes doctor --offline`, then starts `gormes setup` when a terminal is available. Pass `GORMES_SKIP_SETUP=1` to defer that wizard.

> **Windows:** Native Windows is not supported. Please install [WSL2](https://learn.microsoft.com/en-us/windows/wsl/install) and run the command above.

After installation:

```bash
gormes doctor --offline
gormes --offline
```

Prefer to inspect first? Download `install.sh` from the URL above, read it, then run `sh install.sh`. To build entirely from source instead, see [Build From Source](#build-from-source).

## First Run

Want the fastest proof?

```bash
gormes doctor --offline
gormes --offline
```

Then complete the provider-backed path:

```bash
gormes doctor --offline
gormes onboard
gormes setup provider
gormes --oneshot "hello"
```

| Command | What it proves |
|---|---|
| `gormes doctor --offline` | Local runtime, TUI, tools, gateway checks, and local SQLite memory ("Goncho") are wired without using credentials. |
| `gormes onboard` | Shows your config path, skills, agents, channel bindings, missing provider state, and next commands. |
| `gormes setup provider` | Interactively picks endpoint, provider, model, and API key. Secrets go to `~/.gormes/.env`; config goes to `~/.gormes/config.toml`. |
| `gormes --oneshot "hello"` | Sends one provider-backed turn and exits. If this works, the TUI and gateway have a model to use. |

## Daily Use

```bash
gormes                     # open the native TUI
gormes onboard --wizard    # guided first-run readiness plan
gormes dashboard           # web UI at http://127.0.0.1:43827/dashboard
gormes gateway             # run the configured messaging gateway
gormes gateway status      # inspect gateway runtime state
gormes gateway reload      # reload swappable gateway config without restart
gormes logs                # read recent gateway logs
```

## CLI vs Gateway Quick Reference

| Action | Local CLI | Messaging gateway |
|---|---|---|
| Start chatting | `gormes` | Run `gormes gateway`, then message the configured channel bot. |
| Prove local readiness | `gormes doctor --offline` and `gormes --offline` | `gormes gateway status` shows configured channels and runtime state. |
| Send one provider-backed turn | `gormes --oneshot "hello"` | Send a normal message after provider setup. |
| Configure provider/model | `gormes setup provider` or `gormes setup model` | Run setup locally, then `gormes gateway reload`. |
| Route channels to agents | `gormes setup bindings` and `gormes onboard` | `gormes gateway status` confirms active bindings and missing channel state. |
| Diagnose runtime issues | `gormes doctor --offline`, `gormes logs` | `gormes gateway status`, `gormes logs`, then fix config or credentials locally. |

## Multi-Agent Routing

```bash
gormes setup agent         # print agent/profile setup guidance
gormes setup workspace     # set the default workspace path
gormes setup bindings      # route channels to specific agents
gormes onboard             # review the active routes
```

The default agent is ready after install. Add more when you need separate workspaces, models, or channel ownership.

## Migrating From Hermes Or OpenClaw

Gormes can import upstream state without invoking Python at runtime. Always run
a dry-run first and review the redacted manifest before applying changes.

```bash
gormes migrate hermes --dry-run --source ~/.hermes
gormes migrate hermes --yes --source ~/.hermes

gormes migrate openclaw --dry-run --source ~/.openclaw
gormes migrate openclaw --yes --source ~/.openclaw
gormes migrate openclaw --yes --secrets --source ~/.openclaw
```

OpenClaw cleanup is separate from migration and archives old directories rather
than deleting them:

```bash
gormes migrate openclaw cleanup --dry-run
gormes migrate openclaw cleanup --yes
```

## Why People Switch

| | Other agents | Gormes |
|---|---|---|
| **Install** | pip, venvs, system packages | Source build or inspectable `install.sh` to one Go command |
| **First setup** | Find and edit config files | `gormes onboard`, optional wizard, then `gormes setup provider` |
| **Smoke test** | Needs a live model first | `gormes doctor --offline` and `gormes --offline` |
| **State** | Redis, vector DBs, sidecars | SQLite under `~/.gormes` |
| **Channels** | Separate bot glue per platform | One gateway process with channel bindings |
| **Release trust** | Ad-hoc local environments | Tagged release assets with SHA-256, SBOMs, and CI validation; signing and package-manager lanes are still hardening |

## Operator Cheatsheet

```bash
gormes config show              # redacted config
gormes config edit              # open ~/.gormes/config.toml
gormes onboard --wizard         # guided first-run readiness plan
gormes setup model              # select the default model
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
export PATH="$PWD/bin:$PATH"
gormes version
gormes doctor --offline
go test ./... -count=1
go run ./cmd/progress validate
```

## Docs

| Section | What it covers |
|---|---|
| [Installation](https://docs.gormes.ai/getting-started/installation/) | Source build, installer path, PATH, and offline verification. |
| [First run](https://docs.gormes.ai/getting-started/first-run/) | `doctor`, `onboard`, provider setup, and first provider-backed turn. |
| [CLI reference](https://docs.gormes.ai/reference/cli/) | Commands, flags, and operator workflows. |
| [Providers](https://docs.gormes.ai/reference/providers/) | Supported provider config and credential paths. |
| [Configuration](https://docs.gormes.ai/reference/config/) | `~/.gormes/config.toml`, `.env`, agents, workspaces, and bindings. |
| [Gateway](https://docs.gormes.ai/building-gormes/core-systems/gateway/) | Channel runtime, status checks, reloads, and troubleshooting evidence. |
| [Troubleshooting](https://docs.gormes.ai/getting-started/troubleshooting/) | Common install, provider, gateway, and browser-tool failures. |
| [Roadmap](https://docs.gormes.ai/building-gormes/architecture_plan/) | Hermes parity phases, status labels, and progress evidence. |

## Security & Trust

- Source build and inspectable `install.sh` are the recommended scout-release paths.
- `gormes doctor --offline` and `gormes --offline` prove local readiness before token spend.
- Provider secrets stay local under `~/.gormes/.env`; config stays under `~/.gormes/config.toml`.
- Binary size and public benchmark mirrors come from repo data, not hand-tuned marketing copy.
- Release signing and package-manager installs are still active hardening work.

## Status

Early 0.x release.

Working today:

- TUI, onboarding, setup, and offline local smoke tests
- Provider-backed one-shot turns and major provider setup paths
- Gateway paths for Telegram, Discord, and Slack
- Local SQLite memory, dashboard, logs, doctor, and security/secrets audits

In progress:

- Full Hermes runtime parity
- More channels and gateway hardening
- Deeper learning-loop, MCP/plugin, voice/TTS, and release-distribution work

Latest public release: [v0.1.05](https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.05).

Current `development` head after `v0.1.05` also includes fixture-backed work
for the OpenClaw-compatible `gormes migrate claw` alias, cron `no_agent`
script-only watchdog jobs, planned gateway-stop markers, WSL-safe service PATH
handling, and Navivox SSH/admin/key-import/tool-approval groundwork. Those
changes are merged to `development` and will move into the next public release
after the normal release lane.

<details>
<summary>Roadmap phase rollup</summary>

<!-- PROGRESS:START kind=readme-rollup -->
| Phase | Status | Shipped |
|-------|--------|---------|
| Phase 1 — The Dashboard | ✅ | 4/4 subphases |
| Phase 2 — The Gateway | ✅ | 21/21 subphases |
| Phase 3 — The Black Box (Memory) | ✅ | 15/15 subphases |
| Phase 4 — The Brain Transplant | ✅ | 13/13 subphases |
| Phase 5 — The Final Purge | 🔨 | 12/23 subphases |
| Phase 6 — The Learning Loop (Soul) | 🔨 | 8/12 subphases |
| Phase 7 — Paused Channel Backlog | 🔨 | 3/5 subphases |
| Phase 8 — Reputation & Publication | 🔨 | 1/7 subphases |
<!-- PROGRESS:END -->

</details>

Release v0.1.05 publishes static Go binaries for Linux, macOS, and Windows on amd64/arm64. The current benchmark mirror reports a Linux build at ~39.1 MB (`benchmarks.json`, 2026-05-05). CI runs `go test ./... -count=1`, `go run ./cmd/progress validate`, and `git diff --check`. See [CHANGELOG.md](CHANGELOG.md) and [SECURITY.md](SECURITY.md).

---

Built by [Trebuchet Dynamics](https://trebuchetdynamics.com/).
Hermes Agent lineage by [Nous Research](https://nousresearch.com).
MIT license.
