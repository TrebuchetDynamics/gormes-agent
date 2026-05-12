---
title: "CLI Commands"
description: "Complete Gormes command reference: getting started, configuration, gateway, multi-agent, memory, tools, and maintenance."
weight: 10
---

# CLI Commands

Gormes is a single binary. Run `gormes --help` for the live command tree.

## Getting Started

| Command | Purpose |
|---|---|
| `gormes onboard` | Show what's configured and what to do next. Detects missing provider, lists agents, workspaces, and channel bindings. |
| `gormes setup provider` | Interactive guided setup for your LLM provider. Select from OpenAI, Anthropic, DeepSeek, Codex, OpenCode, Groq, or Ollama. API keys stored in `.env`, endpoints in `config.toml`. |
| `gormes setup model` | Interactive model picker. Select your default model from the provider's available models. |
| `gormes --oneshot "prompt"` | Send a single turn and exit. Great for testing provider configuration. |
| `gormes --offline` | Smoke test the TUI without provider calls or credentials. |
| `gormes doctor --offline` | Verify runtime readiness: TUI, tools, gateway, Goncho memory, web backend. |

## Configuration

| Command | Purpose |
|---|---|
| `gormes config edit` | Open `config.toml` in your editor (`$EDITOR` / `$VISUAL`). |
| `gormes config show` | Print resolved configuration with secrets redacted. |
| `gormes config set <key> <value>` | Set a config value. Dotted keys route to TOML sections (e.g. `hermes.endpoint`). Secret keys (`*_API_KEY`, `*_TOKEN`) route to `.env`. |
| `gormes config check` | Validate `config.toml` schema without writing. |
| `gormes config path` | Print the TOML config file path. |
| `gormes config env-path` | Print the `.env` secrets file path. |
| `gormes config migrate` | Apply native `config.toml` schema migrations. |
| `gormes auth add <provider>` | Add a provider credential. Supports `--type api-key` and `--type oauth`. |
| `gormes auth list [provider]` | List credentials with secrets redacted. |
| `gormes auth status <provider>` | Show redacted provider auth status. |
| `gormes auth remove <provider> <target>` | Remove a credential by index, ID, or label. |
| `gormes auth reset <provider>` | Reset credential cooldown/exhaustion state. |
| `gormes auth logout <provider>` | Clear all credentials for a provider. |
| `gormes logout --provider <provider>` | Top-level logout shortcut (nous, openai-codex, spotify). |

### Codex OAuth Login

```bash
gormes auth add openai-codex --type oauth
```

Launches device code flow. You'll get a URL and code — visit `https://auth.openai.com/codex/device`, enter the code, and tokens are stored in the Gormes credential pool.

**Emergency import** (if you already have a Codex CLI auth.json):

```bash
gormes auth add openai-codex --type oauth \
  --emergency-import-from-codex-cli ~/.codex/auth.json
```

### OpenCode as a Provider

```bash
gormes auth add opencode --api-key <your-key>
gormes auth add opencode-go --api-key <your-key>
```

OpenCode Zen (`opencode`) and OpenCode Go (`opencode-go`) are registered as OpenAI-compatible providers. They appear in `gormes setup provider` and `gormes setup model`.

## Gateway — Multi-Channel Messaging

| Command | Purpose |
|---|---|
| `gormes gateway` | Start the configured multi-channel gateway. Runtime-ready channels are Telegram, Discord, and Slack; WhatsApp is row-backed/fixture-backed until the live bridge bundle and gateway registration land. |
| `gormes gateway status` | Check gateway runtime state, connected platforms, active agents. |
| `gormes gateway reload` | Reload swappable config in the live gateway without restarting; invalid config keeps the last-good runtime config. |
| `gormes gateway stop` | Stop a running gateway. |
| `gormes telegram` | Start Telegram-only mode. |
| `gormes whatsapp` | Set up WhatsApp pairing through the row-backed/fixture-backed Baileys bridge. |
| `gormes logs` | View recent gateway logs. |

### Assigning Channels to Specific Agents

In `config.toml`, use bindings to route messages from a channel to a specific agent:

```toml
[[bindings]]
agent_id = "alerts"
[bindings.match]
channel = "telegram"
account_id = "my-bot"
```

Run `gormes setup bindings` for interactive guidance, or `gormes onboard` to see current bindings.

## Multi-Agent & Workspaces

| Command | Purpose |
|---|---|
| `gormes agent reset` | Seed default agent context templates. |
| `gormes setup agent` | Shows how to create additional agents. Each agent has its own workspace, model, and skills. |
| `gormes setup workspace` | Configure the default workspace path. |
| `gormes setup bindings` | Interactive channel→agent binding setup. |
| `gormes profile list` | List available profiles. |
| `gormes profile use <name>` | Switch active profile. |

The default agent is `main` with workspace at `~/.gormes/workspace`. Add more agents in `config.toml`:

```toml
[[agents.list]]
id = "coder"
name = "Coder"
workspace = "/home/xel/projects"
model = "claude-sonnet-4-20250514"
```

## Memory & Sessions

| Command | Purpose |
|---|---|
| `gormes memory status` | Inspect memory store and extractor state. |
| `gormes session list` | List past sessions. |
| `gormes session export <id>` | Export a session transcript. |
| `gormes goncho doctor --json` | Inspect Goncho memory storage (SQLite, FTS5, graph). |

## Tools & Skills

| Command | Purpose |
|---|---|
| `gormes skills list` | List installed runtime and bundled skills. |
| `gormes skills install <url>` | Install a skill from a URL or file path. |
| `gormes mcp list` | List configured MCP servers. |
| `gormes mcp add <name> <command>` | Add an MCP server (stdio transport). |
| `gormes dashboard` | Start the web dashboard at `http://127.0.0.1:43827`. |

## Maintenance

| Command | Purpose |
|---|---|
| `gormes status` | Show runtime status and progress blockers. |
| `gormes usage` | Show provider account usage. |
| `gormes migrate hermes` | Import state from Hermes (dry-run). |
| `gormes migrate openclaw` | Import state from OpenClaw. |
| `gormes version` | Print version and build info. |
| `gormes uninstall` | Remove Gormes artifacts from the system. |

## Global Flags

| Flag | Purpose |
|---|---|
| `--offline` | Run without provider health checks or network submits. |
| `-z, --oneshot <prompt>` | Single-turn mode. Send one prompt and exit. |
| `-m, --model <model>` | Model override for `--oneshot` or TUI. Also settable via `GORMES_INFERENCE_MODEL`. |
| `--provider <name>` | Provider override. Also settable via `GORMES_INFERENCE_PROVIDER`. |
| `--endpoint <url>` | Provider endpoint override. Also settable via `GORMES_ENDPOINT`. |
| `--api-key <key>` | API key override (never persisted). |
| `--remote <url>` | Connect TUI to a remote Gormes gateway over SSE. |
| `--resume <id>` | Override persisted session ID for TUI. |

## Environment Variables

| Variable | Purpose |
|---|---|
| `GORMES_HOME` | Runtime home directory (default `~/.gormes`). |
| `GORMES_API_KEY` | Provider API key. |
| `GORMES_ENDPOINT` | Provider endpoint URL. |
| `GORMES_INFERENCE_MODEL` | Default model name. |
| `GORMES_INFERENCE_PROVIDER` | Default provider name. |
| `GORMES_SKILLS_ROOT` | Custom skills directory. |
| `GORMES_BRANCH` | Installer managed-checkout branch (default `main`). |
| `GORMES_RESTART_GATEWAY` | Restart policy for install.sh (`auto`, `always`, `never`). |
| `GORMES_SKIP_SETUP` | Set to `1`, `true`, `yes`, or `on` to skip the post-install setup wizard. |

## Installer

```bash
curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh
less install.sh
sh install.sh
```

Key installer options:

| Flag | Purpose |
|---|---|
| `--branch <name>` | Git branch cloned or updated in the managed checkout (default `main`). |
| `--home <dir>` | Managed install home (default `~/.gormes`). |
| `--local` | Build from the current checkout instead of the managed installer checkout. |
| `--dry-run` | Preview without making changes. |
| `--skip-setup` | Install and verify without starting `gormes setup`. |
| `--restart-gateway auto\|always\|never` | Gateway restart policy. |
| `--no-restart` | Skip gateway restart. |

By default, `install.sh` clones or updates a managed source checkout, builds
`./cmd/gormes`, publishes the command, verifies it, and starts `gormes setup`
when a terminal is available. For development, use `sh install.sh --local` from
the repo root to build and install from the current checkout.

## Self-Restart

Use `gormes gateway reload` or gateway `/reload` first for allowlists, first-run discovery flags, display/tool-progress settings, provider/model routing, skills root, and agent bindings. Restart remains the path for binary updates, database path changes, or channel transport changes that require reconnecting clients.

When no service manager (systemd/launchd) is detected, the gateway supports self-restart via `/restart`. The process re-executes itself with `syscall.Exec`, preserving all arguments and environment. A takeover marker ensures no duplicate message delivery across the restart boundary.

The installer can also set up automatic startup:
- **systemd** (Linux): `systemctl --user enable gormes-gateway.service`
- **launchd** (macOS): `~/Library/LaunchAgents/com.gormes.gateway.plist`
