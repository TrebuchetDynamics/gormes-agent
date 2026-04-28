<p align="center">
  <img src="assets/gormes-agent-logo.png" alt="GORMES-AGENT" width="600">
</p>

# GORMES-AGENT

Run AI agents as one Go-native runtime.

Gormes is for long-running agents that need predictable installs, stable runtime behavior, recoverable streams, and local diagnostics. It replaces Python-stack runtime drift with a single static binary that is easier to ship, inspect, and operate.

**Status: early-stage scout release. Not production-stable yet.** Use Gormes today for the native TUI, local diagnostics, provider-backed one-shots, gateway work, and Goncho memory development. Do not treat it as production-ready until the remaining brain/provider slices are complete.

Gormes is a standalone, Go-native rewrite of the Hermes Agent architecture. It requires no Python dependencies and no running Hermes backend.

<p align="center">
  <a href="https://docs.gormes.ai/"><img src="https://img.shields.io/badge/Docs-docs.gormes.ai-FFD700?style=for-the-badge" alt="Documentation"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent"><img src="https://img.shields.io/badge/GitHub-TrebuchetDynamics%2Fgormes--agent-181717?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License: MIT">
</p>

---

## TL;DR

- Source build is the recommended install path: clone the repository, inspect it, and run `make build`.
- `gormes --offline` opens the native TUI without network or provider setup.
- `gormes doctor --offline` checks the local runtime surface before you burn tokens.
- `gormes goncho doctor --json` reports local memory paths and Goncho readiness.
- Provider-backed turns run directly from Gormes with `GORMES_ENDPOINT`, `GORMES_API_KEY`, and `GORMES_MODEL`.
- Gateway operators can inspect configured Telegram, Discord, and Slack channels with `gormes gateway status`; deeper connector setup stays in the docs.
- Signed binary releases, checksums, detached signatures, and package-manager manifests are release-hardening work, not current trust claims.

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

## Current State

What works today:

- Native CLI and Bubble Tea TUI.
- Offline smoke test and local doctor diagnostics.
- Provider-compatible one-shot and TUI startup paths.
- Shared gateway runtime for configured Telegram, Discord, and Slack channels.
- Isolated subagent workstreams with durable job metadata.
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

## Installation

### From Source (Recommended)

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes --offline
./bin/gormes doctor --offline
./bin/gormes goncho doctor --json
```

This path is intentionally boring: you build the exact source tree you inspected, without piping a network response into a shell. It proves:

- the binary builds on your machine;
- the native TUI starts without a JS bundle or Python runtime;
- local diagnostics can inspect the tool/config/memory surface without contacting a provider.

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

If local Go is missing, the installer can download a managed Go toolchain. The Gormes runtime itself does not self-update or fetch-and-execute secondary binaries; release hardening is tracking package-manager installs and signed binary artifacts so production environments do not need source-backed bootstrapping.

### Pre-Compiled Binaries

Pre-compiled binaries are not the primary trust path yet. The release target is signed Windows, Linux, and macOS artifacts with SHA-256 checksums, detached signatures, embedded metadata, and package-manager manifests for Homebrew plus Scoop or Winget.

---

## Model-Backed Turn

Configure a provider-compatible endpoint and run Gormes directly:

```bash
export GORMES_ENDPOINT="https://your-provider.example/v1"
export GORMES_API_KEY="..."
export GORMES_MODEL="your-model"
gormes --oneshot "hello from Gormes"
```

Use `gormes` without `--oneshot` to open the TUI against the same configured runtime. If your provider needs an explicit route, pass `--provider <name>` or set `GORMES_INFERENCE_PROVIDER`.

---

## Gateway Operator Surface

Gateway setup details stay in the docs. The README-level operator contract is:

- `gormes gateway` runs every configured channel through one `gateway.Manager` and the same kernel/tool loop as the TUI.
- `gormes gateway status` reads configured channels, pairing state, persisted runtime state, and runtime PID validation without starting channel clients.
- `gormes doctor --offline` reports gateway readiness alongside local tools, provider configuration, and Goncho diagnostics.
- Telegram, Discord, and Slack are the current `gormes gateway` runtime channels. WhatsApp, WeChat, and the broader connector backlog remain progress-row work until their live transports are exposed through the gateway command.
- `gateway start`, `stop`, `restart`, `install`, and `uninstall` are intentionally not mutating service-manager commands inside the binary; run `gormes gateway` in the foreground or supervise it externally.

Deep dive: [Gateway core system](https://docs.gormes.ai/building-gormes/core-systems/gateway/).

---

## Auditability & Security

Gormes is designed for high-trust environments. Auditability comes from source-first builds, zero-CGO static binaries, local-first SQLite memory, and built-in diagnostic tooling.

- Source build is the recommended install path. Convenience installers remain inspect-first from GitHub raw URLs rather than `curl | sh` or `irm | iex` as the primary path.
- The current build is ~17.7 MB, stripped, static, zero-CGO, and does not depend on hidden shared libraries.
- `make test` runs `go test ./...`; `make build` validates `progress.json`, builds `bin/gormes`, records binary metrics, stamps `main.Version`, and regenerates progress-driven docs.
- `gormes doctor --offline` reports local TUI, built-in tools, Goncho, gateway, Slack, and provider-endpoint readiness without contacting a model provider.
- `gormes goncho doctor --json` reports local Goncho config paths, memory DB paths, schema status, session catalog status, queue status, degraded modes, and provider readiness.
- Network calls go to configured provider or gateway endpoints; offline diagnostics do not contact a model provider.
- The runtime is a local binary and does not act as a self-updating dropper.
- Docs and the public site deploy through GitHub Actions workflows under `.github/workflows/`.
- Security reporting policy lives in [SECURITY.md](SECURITY.md).

Release-hardening targets before production-stable distribution:

- Homebrew formula for macOS/Linux and Scoop or Winget manifests for Windows.
- SHA-256 checksums and detached signatures for release artifacts.
- Windows code signing plus embedded version/company/icon metadata through a resource file.
- Release-candidate scanning and vendor false-positive submissions for major AV providers when needed.
- A dedicated `gormes security-audit` command that summarizes filesystem paths, configured endpoints, persistence, and network behavior in one operator-facing report.

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
| Phase 2 — The Gateway | 🔨 | 16/20 subphases |
| Phase 3 — The Black Box (Memory) | 🔨 | 13/15 subphases |
| Phase 4 — The Brain Transplant | 🔨 | 0/9 subphases |
| Phase 5 — The Final Purge | 🔨 | 2/18 subphases |
| Phase 6 — The Learning Loop (Soul) | ⏳ | 0/6 subphases |
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
