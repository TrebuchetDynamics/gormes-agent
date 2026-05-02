<p align="center">
  <img src="assets/gormes-agent-logo.png" alt="GORMES-AGENT" width="600">
</p>

# GORMES-AGENT

Your agents should not crash because of a broken Python environment.

Gormes runs AI agents as a single static binary.

No Python runtime. No virtualenv repair. No backend service just to open the UI.

Start offline. Prove the machine works. Add provider and gateway credentials
later.

Under the hood, Gormes is a Go-native runtime for local TUI work, offline
diagnostics, provider-backed turns, Goncho memory, a dashboard, and messaging
gateways.

![Gormes native TUI running offline](docs/assets/gormes-tui-demo.gif)

The offline TUI starts instantly: no API key, no network calls, no Python, and
no backend services.

<p align="center">
  <a href="https://docs.gormes.ai/"><img src="https://img.shields.io/badge/Docs-docs.gormes.ai-FFD700?style=for-the-badge" alt="Documentation"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/TrebuchetDynamics/gormes-agent/ci.yml?branch=development&style=for-the-badge" alt="CI status"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent"><img src="https://img.shields.io/badge/GitHub-TrebuchetDynamics%2Fgormes--agent-181717?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License: MIT"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> |
  <a href="#what-you-can-do-today">What Works</a> |
  <a href="#why-gormes">Why</a> |
  <a href="#example-run-a-provider-turn">Example</a> |
  <a href="#operator-commands">Commands</a> |
  <a href="#current-state">Current State</a> |
  <a href="#install-paths">Install</a> |
  <a href="#documentation">Docs</a>
</p>

---

## Quick Start

Build from source and start the offline TUI:

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes --offline
```

Expected result: a native terminal UI opens locally. No API key, model call,
Python runtime, Node runtime, Docker daemon, or Hermes process is required.

Then check the local runtime:

```bash
./bin/gormes doctor --offline
```

Expected result: Gormes prints local readiness checks for the TUI, built-in
tools, web/browser configuration, Goncho, gateways, Slack, and provider endpoint
setup. Provider health is skipped in offline mode. Local failures are explicit
and exit non-zero when they block the checked surface.

Prerequisites for source builds: Git, Go 1.25+, and Make.

## What You Can Do Today

- Run a local agent UI with zero runtime dependencies on the offline path.
- Send one-shot prompts to a provider-compatible endpoint.
- Validate your environment before spending tokens.
- Operate configured Telegram, Discord, or Slack agents from one binary.
- Inspect and debug agent memory locally with Goncho.
- Browse sessions, config, skills, and logs in the local dashboard.

## Why Gormes

Python-stack agents are powerful. Operating them is the fragile part.

Gormes removes the boring failure class: broken virtualenvs, host Python drift,
Node bootstraps, sidecar memory stores, and missing backend processes.

| Surface | Python-stack agents | Gormes |
|---|---|---|
| Startup | Runtime and service setup first | `./bin/gormes --offline` |
| Deployment | Virtualenvs, Docker, Nix, sidecars | One Go artifact from `make build` |
| Diagnostics | Spread across layers | Built-in `gormes doctor` |
| Memory | Redis/vector DB/service sidecars | Local Goncho SQLite |

The target operator story is boring on purpose: copy the binary, keep state in
Gormes paths, inspect diagnostics locally, and avoid repairing a language
runtime before the agent can answer.

## Example: Run A Provider Turn

After the offline path works, configure a provider-compatible endpoint in your
local config:

```toml
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
Gormes runs AI agents from one Go runtime with no Python backend.
```

Use `--provider <name>` and `--model <model>` when the route needs to be
explicit. Full provider setup lives in the
[configuration docs](https://docs.gormes.ai/using-gormes/configuration/).

## Operator Commands

Once built, `./bin/gormes` owns the current operator surface.

<details>
<summary>Show common commands</summary>

| Goal | Command | Notes |
|---|---|---|
| Open the local TUI | `./bin/gormes` | Uses configured provider settings. |
| Prove local startup | `./bin/gormes --offline` | No credentials or network submit. |
| Run one turn | `./bin/gormes --oneshot "hi"` | Writes final assistant output and exits. |
| Diagnose local stack | `./bin/gormes doctor --offline` | Skips provider network health. |
| Run configured gateways | `./bin/gormes gateway` | Telegram, Discord, Slack when configured. |
| Inspect gateway state | `./bin/gormes gateway status` | Reads configured/runtime channel state. |
| Inspect Goncho memory | `./bin/gormes goncho doctor --json` | Local SQLite diagnostics. |
| Start web dashboard | `./bin/gormes dashboard --no-open` | htmx dashboard at a local HTTP port. |
| Show logs | `./bin/gormes logs` | Gateway API first, local log fallback. |
| Remove artifacts safely | `./bin/gormes uninstall --dry-run` | Dry-run is the default inspection path. |

Run `./bin/gormes --help` or see [cmd/README.md](cmd/README.md) for the full
command tree.

</details>

## Current State

**Status: early-stage 0.x release.** Useful today, not production-stable yet.

What works:

- Native CLI and Bubble Tea TUI, including the offline smoke path.
- `doctor --offline` for local TUI, tools, web/browser, Goncho, gateways,
  Slack, and provider-endpoint readiness.
- Provider-compatible one-shots and TUI startup paths.
- Configured Telegram and Discord gateways; Slack when Socket Mode credentials
  are complete.
- Goncho memory on local SQLite, plus session search and diagnostic commands.
- htmx dashboard for sessions, config, skills, and logs.
- Web/browser/search tools with typed unavailable evidence when backends are
  missing.

What is still in progress:

- Full Hermes parity and production hardening.
- Gateway coverage is partial beyond Telegram, Discord, and Slack.
- Service-manager helper commands are not stable operator paths yet.
- Voice/TTS/transcription, MCP/plugin parity, and broad channel parity are not
  complete.
- Prebuilt release artifacts are not the primary trust path yet.
- Some docs and package names still preserve Hermes/Honcho wording for
  compatibility and lineage.

## Install Paths

### From Source

The recommended path is to build the exact source tree you inspected:

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes --offline
./bin/gormes doctor --offline
```

### Source-Backed Installer

The installer manages a source checkout, builds `gormes`, and links the command
into your PATH. Inspect the script first, then run it.

Unix, Linux, macOS, WSL, and Termux:

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

Installers, managed Go toolchain behavior, update controls, and rollback
details live in the [install docs](https://docs.gormes.ai/using-gormes/install/).
Convenience aliases exist at `https://gormes.ai/install.sh` and
`https://gormes.ai/install.ps1`, but inspect-first source URLs remain the README
path.

### Prebuilt Release Artifacts

Static release archives, SHA-256 files, SBOMs, and GitHub build-provenance
attestations exist in the release workflow. Signed stable releases and
package-manager distribution are still hardening work, so source builds remain
the primary trust path.

## Auditability & Security

Gormes is designed for high-trust environments where the operator can inspect
what will run.

- Source build is the recommended install path.
- Convenience installers are inspect-first scripts, not `curl | sh` or
  `irm | iex` README commands.
- `make build` uses `CGO_ENABLED=0`, `-trimpath`, and stripped linker flags; the
  current recorded release-path benchmark is ~22.0 MB in `benchmarks.json`.
- `./bin/gormes doctor --offline` does not contact a model provider.
- Network calls go to configured provider, web, browser, or gateway endpoints.
- Local memory and sessions live in Gormes-owned state paths, not in upstream
  Hermes state.
- The runtime is a local binary and does not act as a self-updating dropper.
- Private vulnerability reporting is documented in [SECURITY.md](SECURITY.md).

Release integrity details and remaining production-stable hardening targets also
live in [SECURITY.md](SECURITY.md).

## Coming From Python Hermes

Gormes is a standalone Go-native rewrite of the Hermes Agent architecture. It
does not require a running Hermes process, and it does not read `~/.hermes`
state on startup.

- Use Gormes' own `config.toml` with `[hermes]` provider settings for endpoint,
  API key, provider, and model.
- Hermes config migration is tracked as a parity surface. Dry-run migration
  work exists, but automatic state import is not a production README path yet.
- SOUL.md, context-file, plugin, MCP, ACP, and full Honcho compatibility remain
  deeper roadmap surfaces until operator docs say otherwise.

## Build State

Gormes is a Go-native implementation of the Hermes-Agent architecture.
Original lineage: Hermes-Agent, with upstream Git history preserved for attribution.

| Area | Current state |
|---|---|
| Dashboard / TUI | Shipping |
| Gateway | Partial |
| Memory / Goncho | Active |
| Brain/provider runtime | Active, not production-complete |
| Hermes runtime dependency | None |
| Release artifacts | Static archives workflow exists; signed/package releases pending |

Full progress: [docs.gormes.ai/building-gormes/architecture_plan](https://docs.gormes.ai/building-gormes/architecture_plan/)

<details>
<summary>Generated phase rollup</summary>

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
- `git diff --check` - match the whitespace gate used by CI.

CI for pull requests to `main` and pushes to `development` runs:

```bash
go test ./... -count=1
go run ./cmd/progress validate
git diff --check
```

## Documentation

| Goal | Link |
|---|---|
| First run | [Quickstart](https://docs.gormes.ai/using-gormes/quickstart/) |
| Install details | [Install](https://docs.gormes.ai/using-gormes/install/) |
| Provider and runtime config | [Configuration](https://docs.gormes.ai/using-gormes/configuration/) |
| Command reference | [cmd/README.md](cmd/README.md) |
| Gateway internals | [Gateway core system](https://docs.gormes.ai/building-gormes/core-systems/gateway/) |
| Architecture | [Core systems](https://docs.gormes.ai/building-gormes/core-systems/) |
| Roadmap and progress | [Architecture plan](https://docs.gormes.ai/building-gormes/architecture_plan/) |
| Memory design | [Goncho Honcho Memory](https://docs.gormes.ai/building-gormes/goncho_honcho_memory/) |
| Security reporting | [SECURITY.md](SECURITY.md) |
| Release notes | [CHANGELOG.md](CHANGELOG.md) |

## Community & Support

- [Issues](https://github.com/TrebuchetDynamics/gormes-agent/issues) for bugs,
  missing docs, and feature requests.
- [Security policy](SECURITY.md) for private vulnerability reporting.
- [Docs](https://docs.gormes.ai/) for install, configuration, gateway, and
  architecture details.

## Contributing

Contributions are welcome. Start with the same proof path maintainers use:

```bash
make build
make test
go run ./cmd/progress validate
git diff --check
```

Contributor roadmap: [Building Gormes](https://docs.gormes.ai/building-gormes/).
Use the repo-local skills under `docs/development-skills/` for planner,
builder, TDD, parity audit, interface design, and README refresh work.

---

Built by [Trebuchet Dynamics](https://trebuchetdynamics.com/). Original Hermes
Agent lineage by [Nous Research](https://nousresearch.com). License: MIT.
