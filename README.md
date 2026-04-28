<p align="center">
  <img src="assets/gormes-agent-logo.png" alt="GORMES-AGENT" width="600">
</p>

# GORMES-AGENT

Run AI agents as one Go-native runtime.

Gormes is for long-running agents that need predictable installs, stable runtime behavior, recoverable streams, and local diagnostics. It replaces Python-stack runtime drift with a single static binary that is easier to ship, inspect, and operate.

**Status: early-stage scout release. Not production-stable yet.** Use Gormes today for the native TUI, local diagnostics, provider-backed one-shots, gateway work, and Goncho memory development. Do not treat it as production-ready until the remaining brain/provider slices are complete.

Gormes does **not** require a running Hermes process. It replicated the useful Hermes-Agent architecture in Go; Hermes remains lineage, attribution, and compatibility vocabulary, not a runtime dependency.

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
- Signed binary releases, checksums, detached signatures, and package-manager manifests are release-hardening work, not current trust claims.
- No `hermes gateway start` step is required.

---

## Why Gormes

Python-stack agents are powerful, but production operation is fragile:

- Python environments drift between dev, staging, and prod.
- npm, Nix, and virtualenv setup fails on host skew.
- Multi-process orchestration crashes or hangs under load.
- SSE streams drop and kill long-running turns.
- Debugging crosses Python, Node, shell, and OS runtime boundaries.

Gormes attacks those failure modes directly:

| Feature | Python-stack agents | Gormes-Agent (Go) |
|---|---|---|
| Deployment | Virtualenvs, Docker, Nix, or host package drift | Single static binary built from source |
| Stability | Runtime drift is common across hosts | Immutable Go runtime artifact |
| Recovery | Dropped streams can hard-fail long turns | Route-B reconnect treats drops as recoverable |
| Memory | Often Redis, vector DBs, or sidecars | In-binary Goncho memory layer |
| Diagnostics | Crosses Python, Node, shell, and OS state | `gormes doctor` and Goncho diagnostics in one binary |

- **Install once** - ship a Go binary instead of reconstructing a Python stack.
- **Run the same artifact** - the binary you test is the binary you deploy.
- **Recover stream drops** - dropped SSE streams become recoverable events.
- **Validate locally** - `gormes doctor` catches config and tool problems early.
- **Keep memory in-process** - Goncho memory runs inside Gormes, not as another sidecar.

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

## What Works Today

- Native CLI and Bubble Tea TUI.
- Offline smoke test and local doctor diagnostics.
- Provider-compatible one-shot and TUI startup paths.
- Shared gateway foundation with Telegram and Discord shipped; Slack, WhatsApp, and WeChat are active.
- Isolated subagent workstreams with durable job metadata.
- Goncho memory diagnostics and Honcho-style local memory tools inside the binary.
- Progress-driven docs generated from the canonical architecture plan.

## Current Limits

- Early-stage scout release, not production-stable.
- Brain/provider runtime is active but not fully hardened.
- Gateway coverage is partial across all planned channels.
- Stable tagged releases and changelog discipline are still pending.
- Some docs still preserve Hermes/Honcho naming where compatibility or lineage matters.

---

## Core Capabilities

- **Single static binary** - current Gormes build is ~17.7 MB, stripped, static, and zero-CGO.
- **No Hermes backend dependency** - the shipped runtime is Gormes.
- **No runtime drift** - test and deploy the same Go binary.
- **Recoverable stream behavior** - a dropped connection does not have to kill the agent turn.
- **Local validation** - diagnose config and tool issues before a live run.
- **In-binary memory** - peer context, search, profiles, queue status, and diagnostics live in Gormes.
- **Multi-channel gateway path** - route agent work through chat and messaging adapters as they land.

---

## Trust Signals

- `make test` runs `go test ./...`.
- `make build` validates `progress.json`, builds `bin/gormes`, records binary metrics, and regenerates progress-driven docs.
- `make build` sets the operator-facing version with Go linker flags so release builds can stamp `main.Version`.
- The phase table below is generated from `docs/content/building-gormes/architecture_plan/progress.json`.
- Docs and the public site deploy through GitHub Actions workflows under `.github/workflows/`.
- Security reporting policy lives in [SECURITY.md](SECURITY.md).
- `gormes version` reports the current operator-facing line: `0.2.0-scout`.
- `gormes doctor --offline` reports local TUI, built-in tools, Goncho, gateway, Slack, and provider-endpoint readiness without contacting a model provider.
- `gormes goncho doctor --json` reports local Goncho config paths, memory DB paths, schema status, session catalog status, queue status, degraded modes, and provider readiness.

## Security & Transparency

Gormes is a statically linked Go runtime that performs network calls, local SQLite I/O, process supervision, and gateway work. Those traits are useful, but they can look suspicious to heuristic scanners when a project is young.

Current posture:

- Source build is the recommended install path. Convenience installers remain inspect-first from GitHub raw URLs rather than `curl | sh` or `irm | iex` as the primary path.
- The binary is zero-CGO, statically linked, and does not depend on hidden shared libraries.
- The runtime is a local binary and does not act as a self-updating dropper.
- Network calls go to configured provider or gateway endpoints; offline diagnostics do not contact a model provider.
- Local diagnostics expose what is configured before live provider calls: `gormes doctor --offline` checks the runtime surface, and `gormes goncho doctor --json` reports memory paths, schema status, queue status, degraded modes, and provider readiness.
- Goncho memory is local SQLite-backed state. The exact path is configuration-dependent and is reported by the Goncho doctor command.

Release-hardening targets before production-stable distribution:

- Homebrew formula for macOS/Linux and Scoop or Winget manifests for Windows.
- SHA-256 checksums and detached signatures for release artifacts.
- Windows code signing plus embedded version/company/icon metadata through a resource file.
- Release-candidate scanning and vendor false-positive submissions for major AV providers when needed.
- A dedicated `gormes security-audit` command that summarizes filesystem paths, configured endpoints, persistence, and network behavior in one operator-facing report.

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

## How It Works

Gormes keeps the operator-facing runtime in Go:

- `cmd/gormes` owns the CLI, TUI, doctor, gateway, memory, and Goncho commands.
- `internal/hermes` owns provider-compatible stream contracts and adapters. The package name is compatibility lineage, not a process dependency.
- `internal/goncho` and `internal/gonchotools` provide in-binary Honcho-style memory.
- `internal/gateway` and `internal/channels/*` route events across messaging adapters.
- `docs/content/building-gormes/architecture_plan/progress.json` is the canonical roadmap.

Architecture depth belongs in the docs: [Core systems](https://docs.gormes.ai/building-gormes/core-systems/) and [Architecture plan](https://docs.gormes.ai/building-gormes/architecture_plan/).

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
