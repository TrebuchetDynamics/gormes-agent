<p align="center">
  <img src="assets/gormes-agent-logo.png" alt="GORMES-AGENT" width="600">
</p>

# GORMES-AGENT

Your agents shouldn't crash because of a broken Python environment.

Run AI agents as a single static binary: no Python, no environment drift, no running Hermes backend.

Gormes is built for agents that need to stay running across restarts, machines, and flaky networks. It gives you a local terminal UI, offline diagnostics, provider-backed turns, and configured gateway channels from one Go-native runtime.

**Status: early-stage.** The TUI, offline doctor, and provider-backed one-shots work today; the full agent runtime is still in progress.

![Gormes native TUI running offline](docs/assets/gormes-tui-demo.gif)

The offline TUI starts without credentials, provider calls, Python, Node, Docker, or a Hermes process.

<p align="center">
  <a href="https://docs.gormes.ai/"><img src="https://img.shields.io/badge/Docs-docs.gormes.ai-FFD700?style=for-the-badge" alt="Documentation"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent"><img src="https://img.shields.io/badge/GitHub-TrebuchetDynamics%2Fgormes--agent-181717?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License: MIT">
</p>

---

## TL;DR

- One static Go binary runs the TUI, diagnostics, provider turns, and configured gateways.
- `./bin/gormes --offline` proves the runtime works before credentials, network calls, or token spend.

---

## Quick Start

Build from source and open the local UI:

Prerequisites: Git, Go, and Make.

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes --offline
```

Expected result: a native terminal UI opens locally. No API key, network call, Python runtime, or Hermes process is required.

Then verify the local stack:

```bash
./bin/gormes doctor --offline
```

Expected result: Gormes prints local readiness checks for the TUI, tools, gateway configuration, Goncho, and provider endpoint setup. Failures are explicit and exit non-zero.

---

### Getting Started

Once built, the `gormes` binary contains the current operator surface:

```bash
./bin/gormes                         # Interactive TUI
./bin/gormes --oneshot "hi"          # Run one turn and exit
./bin/gormes doctor --offline        # Diagnose local config, tools, and memory
./bin/gormes goncho doctor --json    # Inspect local SQLite memory paths and schema
./bin/gormes gateway status          # Check Discord, Telegram, and Slack runtime state
./bin/gormes memory status           # Inspect persisted memory and extractor queues
./bin/gormes session export <id>     # Export a persisted session transcript
```

---

## Why Gormes

Python-stack agents are powerful, but production operation is fragile. Gormes moves the runtime surface into one inspectable Go artifact.

| Feature | Python-stack agents | Gormes-Agent (Go) |
|---|---|---|
| **Deployment** | Virtualenvs, Docker, Nix | **Single static binary** |
| **Stability** | Runtime drift across hosts | **Immutable Go artifact** |
| **Recovery** | Dropped streams kill turns | **Route-B reconnect** |
| **Memory** | Redis, vector DBs, sidecars | **In-binary Goncho SQLite** |
| **Diagnostics** | Crosses Node/Python/OS bounds | **Built-in `gormes doctor`** |

That means you can drop the binary onto a small VM, a Raspberry Pi, or a managed server without rebuilding a Python environment first. No `pip install`, no virtualenv repair, no Node bootstrap just to reach the terminal UI.

It also means memory can stay boring and local. Goncho keeps session history, peer profiles, and diagnostics inside local SQLite instead of requiring Redis or an external vector database just to make the agent remember what happened.

## What You Can Do Today

- Run a local agent UI without installing Python, Node, or Docker at runtime.
- Send one-shot prompts to any provider-compatible endpoint from one binary.
- Validate your runtime before spending tokens with `./bin/gormes doctor --offline`.
- Operate Telegram, Discord, or Slack agents through one gateway runtime.
- Develop and debug Goncho memory entirely offline with `./bin/gormes goncho doctor`.

## Example: Run A One-Shot Turn

Configure a provider-compatible endpoint in your local config:

```toml
# ~/.config/gormes/config.toml
[hermes]
endpoint = "https://your-provider.example/v1"
api_key = "..."
model = "your-model"
```

Then run one prompt:

```bash
./bin/gormes --oneshot "Summarize this repo in one sentence"
```

Example output:

```text
Gormes runs AI agents as a single static Go runtime with no Python backend.
```

Use `./bin/gormes` without `--oneshot` to open the TUI against the same configured runtime. If your provider needs an explicit route, pass `--provider <name>` with an explicit `--model <model>`.

## Current State

What works today:

- Native CLI, Bubble Tea TUI, offline smoke test, and doctor diagnostics.
- Provider-compatible one-shots and TUI startup paths.
- Configured Telegram, Discord, and Slack gateway runtime.
- Isolated subagents with durable job metadata.
- Goncho memory diagnostics and Honcho-style local memory tools inside the binary.
- Progress-driven docs generated from the canonical architecture plan.

Current limits:

- Early-stage scout release, not production-stable.
- Brain/provider runtime is active but not fully hardened.
- Gateway coverage is partial across all planned channels.
- WhatsApp, WeChat, and the longer connector backlog are tracked in progress docs, not exposed as production-ready README setup paths.
- Stable tagged releases and changelog discipline are still pending.
- Some docs still preserve Hermes/Honcho naming where compatibility or lineage matters.

---

## Install Paths

### From Source

The Quick Start path above is the recommended install path. It is intentionally boring: you build the exact source tree you inspected, without piping a network response into a shell.

### Source-Backed Installer (Convenience)

Unix (Linux / macOS / Termux):

```bash
curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.sh
less install.sh
sh install.sh
gormes --offline
gormes doctor --offline
```

Windows (PowerShell):

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 -OutFile install.ps1
Get-Content .\install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
gormes --offline
gormes doctor --offline
```

The installer manages a source checkout under `~/.gormes/gormes-agent` or `%LOCALAPPDATA%\gormes\gormes-agent`, builds `gormes`, and updates in place on rerun. Convenience aliases still exist at `https://gormes.ai/install.sh` and `https://gormes.ai/install.ps1`, but the README uses GitHub-hosted source so operators can inspect the script before running it.

If local Go is missing, the installer can download a managed Go toolchain. The Gormes runtime itself does not self-update or fetch-and-execute secondary binaries.

### Pre-Compiled Binaries

Pre-compiled binaries are not the primary trust path yet. The release target is signed Windows, Linux, and macOS artifacts with SHA-256 checksums, detached signatures, embedded metadata, and package-manager manifests for Homebrew plus Scoop or Winget.

---

## Gateway Operator Surface

Gateway setup details stay in the docs. The README-level operator contract is:

- `./bin/gormes gateway` runs configured Telegram, Discord, and Slack channels through the same kernel/tool loop as the TUI.
- `./bin/gormes gateway status` reads configured channel state without starting channel clients.
- `./bin/gormes doctor --offline` reports gateway readiness alongside local tools, provider configuration, and Goncho diagnostics.

| Action | Native CLI / TUI | Configured gateway |
|---|---|---|
| **Start chatting** | `./bin/gormes` | Send a message to the paired bot |
| **Run offline diagnostics** | `./bin/gormes doctor --offline` | CLI only |
| **Inspect Goncho memory** | `./bin/gormes goncho doctor --json` | Planned |
| **Persist subagent jobs** | Native execution | Routed through gateway |

Deep dive: [Gateway core system](https://docs.gormes.ai/building-gormes/core-systems/gateway/).

---

## Auditability & Security

Gormes is designed for high-trust environments. Auditability comes from source-first builds, zero-CGO static binaries, local-first SQLite memory, and built-in diagnostic tooling.

- Source build is the recommended install path. Convenience installers remain inspect-first from GitHub raw URLs rather than `curl | sh` or `irm | iex`.
- The current build is ~17.7 MB, stripped, static, zero-CGO, and has no hidden shared library dependency.
- `./bin/gormes doctor --offline` reports local TUI, built-in tools, Goncho, gateway, Slack, and provider-endpoint readiness without contacting a model provider.
- Network calls go to configured provider or gateway endpoints; offline diagnostics do not contact a model provider.
- The runtime is a local binary and does not act as a self-updating dropper.
- Security reporting policy lives in [SECURITY.md](SECURITY.md).

Release integrity details and production-stable hardening targets live in [SECURITY.md](SECURITY.md).

---

## How It Works

Gormes keeps the operator-facing runtime in Go:

- `cmd/gormes` owns the CLI, TUI, doctor, gateway, memory, and Goncho commands.
- `internal/hermes` owns provider-compatible stream contracts and adapters. The package name is compatibility lineage, not a process dependency.
- `internal/goncho` and `internal/gonchotools` provide in-binary Honcho-style memory.
- `internal/gateway` and `internal/channels/*` route events across messaging adapters.
- `docs/content/building-gormes/architecture_plan/progress.json` is the canonical roadmap.

Architecture depth belongs in the docs: [Core systems](https://docs.gormes.ai/building-gormes/core-systems/) and [Architecture plan](https://docs.gormes.ai/building-gormes/architecture_plan/).

---

## Coming From Python Hermes

Gormes is a standalone Go-native rewrite of the Hermes Agent architecture. It does not require a running Hermes process, and it does not read `~/.hermes` state on startup.

- Use Gormes' own `config.toml` with `[hermes]` provider settings for endpoint, API key, and model.
- Hermes config migration is tracked under Phase 5.O; dry-run manifest work exists, but automatic state import is not a production README path yet.
- SOUL.md, context-file, plugin, MCP, and ACP compatibility remain deeper roadmap surfaces until their operator docs say otherwise.

---

## Build State

Gormes is a Go-native implementation of the Hermes-Agent architecture. Original lineage: Hermes-Agent, with upstream Git history preserved for attribution.

| Area | Current state |
|---|---|
| Dashboard / TUI | Shipping |
| Gateway | Partial |
| Memory / Goncho | Active |
| Brain/provider runtime | Active, not production-complete |
| Hermes runtime dependency | None |

Full progress: [docs.gormes.ai/building-gormes/architecture_plan](https://docs.gormes.ai/building-gormes/architecture_plan/)

<details>
<summary>Generated phase rollup</summary>

<!-- PROGRESS:START kind=readme-rollup -->
| Phase | Status | Shipped |
|-------|--------|---------|
| Phase 1 — The Dashboard | ✅ | 4/4 subphases |
| Phase 2 — The Gateway | 🔨 | 20/21 subphases |
| Phase 3 — The Black Box (Memory) | ✅ | 15/15 subphases |
| Phase 4 — The Brain Transplant | 🔨 | 4/11 subphases |
| Phase 5 — The Final Purge | 🔨 | 3/20 subphases |
| Phase 6 — The Learning Loop (Soul) | 🔨 | 0/9 subphases |
| Phase 7 — Paused Channel Backlog | 🔨 | 2/5 subphases |
<!-- PROGRESS:END -->

</details>

---

## Developer Workflow

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
make test
./bin/gormes --offline
./bin/gormes doctor --offline
```

Useful commands:

- `make validate-progress` - validate the canonical progress file.
- `make generate-progress` - regenerate progress-driven Markdown and site data.
- `go run ./cmd/progress validate` - run the validation command directly.
- `go run ./cmd/repoctl readme update` - refresh README benchmark text from `benchmarks.json`.

---

## Documentation

- [Quickstart](https://docs.gormes.ai/using-gormes/quickstart/)
- [Install](https://docs.gormes.ai/using-gormes/install/)
- [Configuration](https://docs.gormes.ai/using-gormes/configuration/)
- [Gateway](https://docs.gormes.ai/building-gormes/core-systems/gateway/)
- [Core systems](https://docs.gormes.ai/building-gormes/core-systems/)
- [Architecture plan](https://docs.gormes.ai/building-gormes/architecture_plan/)
- [Goncho Honcho Memory](https://docs.gormes.ai/building-gormes/goncho_honcho_memory/)
- [Command reference](cmd/README.md)

---

## Community & Support

- [Issues](https://github.com/TrebuchetDynamics/gormes-agent/issues) for bugs, missing docs, and feature requests.
- [Security policy](SECURITY.md) for private vulnerability reporting.
- [Docs](https://docs.gormes.ai/) for install, configuration, gateway, and architecture details.

---

## Contributing

Contributions are welcome. Start with the same proof path maintainers use:

```bash
make build
make test
go run ./cmd/progress validate
```

Contributor roadmap: [Building Gormes](https://docs.gormes.ai/building-gormes/). Use the repo-local skills under `docs/development-skills/` for planner, builder, TDD, parity audit, interface design, and README refresh work.

---

Built by [Trebuchet Dynamics](https://trebuchetdynamics.com/). Original Hermes Agent lineage by [Nous Research](https://nousresearch.com). License: MIT.
