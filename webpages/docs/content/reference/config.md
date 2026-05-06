---
title: "Configuration"
description: "Native Gormes config sections, setup wizard, and where to inspect the active config."
weight: 20
---

# Configuration

Gormes configuration lives in `GORMES_HOME`, defaulting to `~/.gormes`.

**Recommended first-time setup**: `gormes setup provider` (interactive wizard). Manual editing is optional for advanced users.

## Quick Commands

```bash
gormes config path        # where is config.toml?
gormes config show        # see current config (secrets redacted)
gormes config check       # validate config.toml schema
gormes config edit        # open in your editor
gormes config set <k> <v> # set a value (dotted keys for sections)
gormes config env-path    # where is .env secrets file?
```

## Setup Wizard

The `gormes setup` command covers common configuration interactively:

| Command | What it configures |
|---|---|
| `gormes setup provider` | Provider endpoint, API key (→ `.env`), default model |
| `gormes setup model` | Interactive model picker |
| `gormes setup agent` | Multi-agent setup guidance |
| `gormes setup workspace` | Workspace directory |
| `gormes setup bindings` | Channel→agent routing |

## Config Sections

Current top-level TOML sections:

| Section | Purpose |
|---|---|
| `[hermes]` | Provider endpoint, API key, model, fallback chain |
| `[gateway]` | Gateway channel tokens and settings |
| `[telegram]` | Telegram bot token and allowed chat |
| `[discord]` | Discord bot token and channel |
| `[slack]` | Slack bot token and app token |
| `[web]` | Web search backend selection |
| `[browser]` | Browser automation settings |
| `[security]` | Approval and sandboxing policy |
| `[skills]` | Skills root, limits, selection cap |
| `[goncho]` | Memory store settings |
| `[cron]` | Scheduled automations |
| `[[agents.list]]` | Agent definitions (multi-agent) |
| `[[bindings]]` | Channel→agent routing rules |
| `[agents.defaults]` | Default workspace and agent settings |
| `[delegation]` | Sub-agent delegation policy |

## Provider Configuration

### API Key

```bash
gormes auth add openai --api-key sk-...
gormes auth add anthropic --api-key sk-ant-...
gormes auth add opencode --api-key <your-key>
```

Secrets are stored in `~/.gormes/auth.json` (credential pool) and `~/.gormes/.env` (dotenv). They are never written to `config.toml`.

### OAuth (Codex, Anthropic, Nous)

```bash
gormes auth add openai-codex --type oauth
gormes auth add anthropic --type oauth
gormes auth add nous --type oauth
```

OAuth tokens are stored in the credential pool with automatic refresh support.

### Direct config.toml (advanced)

```toml
[hermes]
endpoint = "https://api.openai.com/v1"
model = "gpt-4o"
```

API key in `~/.gormes/.env`:

```
GORMES_API_KEY=sk-...
```

## Gateway Channel Secrets

Keep channel tokens in `.env` and use TOML for non-secret routing and allowlist values:

```dotenv
GORMES_TELEGRAM_TOKEN=123:abc
SLACK_BOT_TOKEN=xoxb-...
SLACK_APP_TOKEN=xapp-...
```

```toml
[telegram]
allowed_chat_id = 42

[discord]
allowed_channel_id = "1234567890"

[slack]
default_channel_id = "C0123456789"
```

## Multi-Agent Configuration

```toml
[[agents.list]]
id = "main"
name = "Main"
workspace = "/home/xel/.gormes/workspace"
model = "gpt-4o"
default = true

[[agents.list]]
id = "coder"
name = "Coder"
workspace = "/home/xel/projects"
model = "claude-sonnet-4-20250514"

[[bindings]]
agent_id = "coder"
[bindings.match]
channel = "telegram"
account_id = "my-coding-bot"
```

When no bindings are configured, all channels route to the default agent.

## Environment Variables

| Variable | Config equivalent |
|---|---|
| `GORMES_HOME` | Runtime home directory |
| `GORMES_API_KEY` | `hermes.api_key` |
| `GORMES_ENDPOINT` | `hermes.endpoint` |
| `GORMES_INFERENCE_MODEL` | `hermes.model` |
| `GORMES_INFERENCE_PROVIDER` | `hermes.provider` |
| `GORMES_SKILLS_ROOT` | `skills.root` |
| `GORMES_RESTART_GATEWAY` | Installer restart policy |

Use generated config reference for exact field types before publishing a stable release.
