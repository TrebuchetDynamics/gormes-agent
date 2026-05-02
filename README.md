<p align="center">
  <img src="assets/gormes-agent-logo.png" alt="GORMES-AGENT" width="600">
</p>

<p align="center">
  <strong>AI agents that don't break.</strong><br>
  One binary. No Python. No Docker.
</p>

<p align="center">
  <a href="https://docs.gormes.ai/"><img src="https://img.shields.io/badge/docs-gormes.ai-FFD700?style=flat-square" alt="Docs"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/TrebuchetDynamics/gormes-agent/ci.yml?branch=development&style=flat-square" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"></a>
  <a href="https://github.com/TrebuchetDynamics/gormes-agent"><img src="https://img.shields.io/github/stars/TrebuchetDynamics/gormes-agent?style=social" alt="Stars"></a>
</p>

---

![Gormes native TUI](docs/assets/gormes-tui-demo.gif)

```bash
curl -fsSLO https://gormes.ai/install.sh && sh install.sh
```

That's it. No Python. No pip. No Docker.

### First run

```bash
gormes onboard
```

Tells you exactly what's configured and what to do next. Missing a provider?

```bash
gormes setup provider     # picks your LLM provider interactively
gormes --oneshot "hello"  # one turn, done
```

Now you have a working agent. Three commands.

### Daily use

```bash
gormes                     # open the TUI
gormes gateway             # run as Telegram, Discord, or Slack bot
gormes dashboard           # web UI at localhost:43827
```

### Multi-agent, multi-channel

```bash
gormes setup agent         # create agents with different models
gormes setup bindings      # route Telegram → coder, Slack → assistant
```

### Why people switch

| | Other agents | Gormes |
|---|---|---|
| **Install** | pip, venv, dependency hell | `curl \| sh` |
| **Setup** | Edit config files by hand | `gormes onboard` → `setup provider` |
| **Memory** | Redis, vector DB | SQLite in the binary |
| **Recovery** | Crash = lost context | Self-restart, session persistence |
| **Channels** | One bot per process | Telegram, Discord, Slack, WhatsApp, WeChat |
| **Size** | ~500MB Python stack | ~22.2 MB static Go binary |

### Why we built it

Python agents are incredible — until they're not. The moment you need them to
stay running, the infrastructure eats your time. Gormes is what we wanted:
boring, local, and impossible to break.

### Docs

[Setup guide](https://docs.gormes.ai/getting-started/first-run/) ·
[CLI reference](https://docs.gormes.ai/reference/cli/) ·
[Configuration](https://docs.gormes.ai/reference/config/) ·
[Gateway](https://docs.gormes.ai/building-gormes/core-systems/gateway/) ·
[Roadmap](https://docs.gormes.ai/building-gormes/architecture_plan/)

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

Binary: ~22.2 MB static Go binary, CGO_ENABLED=0, stripped. 3418+ tests, 615+ Go source files, 119 dependencies. [SECURITY.md](SECURITY.md).

---

Built by [Trebuchet Dynamics](https://trebuchetdynamics.com/).
Hermes Agent lineage by [Nous Research](https://nousresearch.com).
MIT license.
