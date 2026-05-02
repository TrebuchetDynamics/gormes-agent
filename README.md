<p align="center">
  <img src="assets/gormes-agent-logo.png" alt="GORMES-AGENT" width="600">
</p>

# GORMES-AGENT

Your agents should not crash because of a broken Python environment.

Gormes is a Go-native runtime and rewrite of the Hermes Agent operator surface.
The release path builds a single static binary for the local terminal UI,
offline diagnostics, provider-backed turns, local Goncho memory, the htmx
dashboard, and configured gateway channels. The offline path does not need
Python, a virtualenv, Node, Docker, or a running Hermes backend.

**Status: early-stage scout release.** The native TUI, `doctor --offline`,
provider one-shots, Go tool registry, Goncho memory, htmx dashboard, and
configured Telegram/Discord/Slack gateway paths have implementation and tests.
Full Hermes parity, broad channel parity, voice/TTS/transcription parity,
MCP/plugin parity, release signing, package-manager distribution, and TUI polish
are still in progress.

![Gormes native TUI running offline](docs/assets/gormes-tui-demo.gif)

<p align="center">
  <a href="https://docs.gormes.ai/"><img src="https://img.shields.io/badge/Docs-docs.gormes.ai-FFD700?style=for-the-badge" alt="Documentation"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/TrebuchetDynamics/gormes-agent/ci.yml?branch=development&style=for-the-badge" alt="CI status"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent"><img src="https://img.shields.io/badge/GitHub-TrebuchetDynamics%2Fgormes--agent-181717?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License: MIT"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> |
  <a href="#operator-commands">Commands</a> |
  <a href="#current-state">Current State</a> |
  <a href="#install-paths">Install</a> |
  <a href="#auditability--security">Security</a> |
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

## Operator Commands

Once built, `./bin/gormes` owns the current operator surface:

| Goal | Command | Notes |
|---|---|---|
| Open the local TUI | `./bin/gormes` | Uses configured provider settings. |
| Prove local startup | `./bin/gormes --offline` | No credentials or network submit. |
| Run one turn | `./bin/gormes --oneshot "hi"` | Writes final assistant output and exits. |
| Diagnose local stack | `./bin/gormes doctor --offline` | Skips provider network health. |
| Choose model/provider | `./bin/gormes model` | Interactive selector for configured providers. |
| Manage credentials | `./bin/gormes auth` | Provider credential pool commands. |
| Run configured gateways | `./bin/gormes gateway` | Telegram, Discord, Slack when configured. |
| Inspect gateway state | `./bin/gormes gateway status` | Reads configured/runtime channel state. |
| Inspect Goncho memory | `./bin/gormes goncho doctor --json` | Local SQLite diagnostics. |
| Inspect session memory | `./bin/gormes memory status` | Persisted memory and extractor state. |
| Export a transcript | `./bin/gormes session export <id>` | Persisted session transcript. |
| Start web dashboard | `./bin/gormes dashboard --no-open` | htmx dashboard at a local HTTP port. |
| Show logs | `./bin/gormes logs` | Gateway API first, local log fallback. |
| Remove artifacts safely | `./bin/gormes uninstall --dry-run` | Dry-run is the default inspection path. |

Run `./bin/gormes --help` or see [cmd/README.md](cmd/README.md) for the full
command tree.

## Configure A Provider

For an OpenAI-compatible endpoint, put provider settings in
`~/.config/gormes/config.toml`:

```toml
[hermes]
endpoint = "https://your-provider.example/v1"
api_key = "..."
model = "your-model"
```

Then run:

```bash
./bin/gormes --oneshot "Summarize this repo in one sentence"
```

Example output:

```text
Gormes runs AI agents from one Go runtime with no Python backend.
```

Use `--provider <name>` and `--model <model>` when a configured provider route
needs to be explicit. Invocation-only overrides also exist for `--endpoint` and
`--api-key`; the API key flag is not persisted.

## Why Gormes

Python-stack agents are powerful, but production operation can fail before the
agent ever reaches its model: virtualenv drift, host Python changes, wheel
issues, Node bootstraps, sidecar databases, or a missing backend process.
Gormes keeps the operator-facing runtime in one inspectable Go artifact.

| Surface | Python-stack agents | Gormes-Agent |
|---|---|---|
| Deployment | Virtualenvs, Docker, Nix, sidecars | One static Go binary from `make build` |
| Offline proof | Often crosses Python/Node/provider setup | `gormes --offline` and `gormes doctor --offline` |
| Provider turns | Backend process and SDK stack | Native provider-compatible Go client paths |
| Memory | Redis/vector DB/service sidecars | In-binary Goncho on local SQLite |
| Diagnostics | Spread across runtime layers | Built-in `gormes doctor` |
| Recovery | Dropped streams and process drift are still common | Typed retry/cancel evidence, still hardening |

The target operator story is boring on purpose: copy the binary, keep state
under the Gormes home/config paths, inspect diagnostics locally, and avoid
repairing a Python runtime before the agent can answer.

## Current State

What works today:

- Native CLI and Bubble Tea TUI, including an offline smoke path.
- `doctor --offline` for local TUI, tools, web/browser, Goncho, gateway, Slack,
  and provider-endpoint readiness checks.
- Provider-compatible one-shot turns and TUI startup paths.
- Configured Telegram and Discord gateway runtime.
- Slack Socket Mode gateway path when `bot_token`, `app_token`, and enabled
  channel settings are complete.
- Goncho memory with local SQLite persistence, session search, and
  Honcho-compatible tool names such as `honcho_search` and `honcho_context`.
- htmx dashboard for local sessions, config, skills, and logs.
- Web tools: DuckDuckGo search fallback by default; Firecrawl, Parallel,
  Tavily, Exa, Brave, SearXNG, Perplexity, and CDP-backed extraction when
  configured.
- Browser tool contracts backed by the Go browser harness bridge when present,
  with typed unavailable evidence when it is not.
- Skill read/write tools: `skills_list`, `skill_view`, and `skill_manage`.
- Gateway busy modes via `/busy interrupt`, `/busy queue`, and `/busy steer`.
- Release workflow support for static archives, SHA-256 files, SBOMs, and
  GitHub build-provenance attestations.
- Progress-driven architecture docs generated from `progress.json`.

Current limits:

- Scout release, not production-stable.
- Brain/provider runtime is active but still hardening.
- Gateway coverage is partial. Telegram and Discord are the main configured
  paths; Slack requires complete Socket Mode credentials.
- WhatsApp, WeChat/WeCom, Yuanbao, Signal, Matrix, and the longer connector
  backlog are tracked in progress docs unless an operator doc says otherwise.
- `gateway start`, `gateway restart`, `gateway install`, and
  `gateway uninstall` are registered as unavailable helper surfaces, not stable
  service-manager commands.
- Delegation is disabled by default and must be enabled in config before
  `delegate_task` is registered.
- `text_to_speech` has a tool contract, but production voice/TTS/transcription
  provider parity is not complete.
- MCP/plugin surfaces are partial; manifest, OAuth, and tool-host boundaries are
  still moving toward parity.
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
into your PATH. The README uses GitHub-hosted script URLs so operators can
inspect the script before running it.

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

Default managed source homes are `~/.gormes/gormes-agent` on Unix-like systems
and `%LOCALAPPDATA%\gormes\gormes-agent` on Windows. If Go is missing, the
installer can install or download a managed Go toolchain. The Gormes runtime
itself does not self-update or fetch and execute secondary runtime binaries.

Convenience aliases exist at `https://gormes.ai/install.sh` and
`https://gormes.ai/install.ps1`, but inspect-first source URLs remain the
recommended README path.

### Prebuilt Release Artifacts

The release workflow can produce static archives for Linux, macOS, and Windows,
with SHA-256 files, SBOMs, and GitHub build-provenance attestations. Stable
signed releases, Windows code signing, detached signatures, and package-manager
distribution are still release-hardening work. Treat source builds as the
primary trust path until those are complete.

## Gateway Operator Surface

`./bin/gormes gateway` runs every configured channel through the same
`gateway.Manager` and kernel/tool loop used by the TUI.

| Action | Native CLI / TUI | Configured gateway |
|---|---|---|
| Start chatting | `./bin/gormes` | Send a message to the paired bot |
| Run offline diagnostics | `./bin/gormes doctor --offline` | CLI only |
| Inspect runtime state | `./bin/gormes gateway status` | CLI read-model |
| Inspect Goncho memory | `./bin/gormes goncho doctor --json` | Planned gateway operator surface |
| Delegated jobs | Config-gated via `[delegation].enabled` | Same registry once enabled |

Deep dive: [Gateway core system](https://docs.gormes.ai/building-gormes/core-systems/gateway/).

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

## How It Works

Gormes keeps the operator-facing runtime in Go:

- `cmd/gormes` owns the CLI, TUI, doctor, gateway, memory, Goncho, dashboard,
  auth, setup, model, status, logs, and uninstall commands.
- `internal/hermes` owns provider-compatible stream contracts and adapters. The
  package name is compatibility lineage, not a backend process dependency.
- `internal/kernel` drives turns, tool calls, retries, admission limits, and
  render frames.
- `internal/goncho` and `internal/gonchotools` provide in-binary
  Honcho-compatible memory surfaces on local SQLite.
- `internal/gateway` and `internal/channels/*` route messaging events through
  shared channel contracts.
- `internal/tools` owns Go-native tools for files, terminal, web, browser,
  skills, TTS contracts, MCP seams, and provider-specific utilities.
- `docs/content/building-gormes/architecture_plan/progress.json` is the
  canonical roadmap and progress source.

Architecture depth belongs in the docs: [Core systems](https://docs.gormes.ai/building-gormes/core-systems/)
and [Architecture plan](https://docs.gormes.ai/building-gormes/architecture_plan/).

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
