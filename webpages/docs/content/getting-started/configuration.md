---
title: "Configuration"
description: "GORMES_HOME, config.toml, setup wizard, credential pool, dotenv secrets, and multi-agent config."
weight: 30
---

# Configuration

Gormes uses `GORMES_HOME` (default `~/.gormes`). All state lives there.

## Quick Setup (Recommended)

```bash
gormes setup provider       # interactive wizard — no file editing needed
gormes setup model          # pick your default model
gormes onboard              # see what's configured
```

## Config Commands

```bash
gormes config path          # where is config.toml?
gormes config env-path      # where is .env?
gormes config show          # current config (secrets redacted)
gormes config check         # validate schema
gormes config edit          # open in $EDITOR
gormes config set <k> <v>   # set a value
```

## Default Paths

| Item | Default |
|---|---|
| Config | `~/.gormes/config.toml` |
| Secrets | `~/.gormes/.env` |
| Credential pool | `~/.gormes/auth.json` |
| Sessions DB | `~/.gormes/sessions.db` |
| Memory DB | `~/.gormes/memory.db` |
| Gateway log | `~/.gormes/gateway.log` |

## Provider Credentials

**API key** (goes to `.env`, never in `config.toml`):

```bash
gormes auth add openai --api-key sk-...
gormes auth add opencode --api-key <your-key>
```

**OAuth** (stored in credential pool with auto-refresh):

```bash
gormes auth add openai-codex --type oauth
gormes auth add anthropic --type oauth
gormes auth add nous --type oauth
```

**Direct config** (advanced):

```toml
[hermes]
endpoint = "https://api.openai.com/v1"
model = "gpt-4o"
```

```
# ~/.gormes/.env
GORMES_API_KEY=sk-...
```

## Multi-Agent Configuration

```toml
[[agents.list]]
id = "main"
name = "Main"
workspace = "/home/xel/.gormes/workspace"
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
account_id = "my-bot"
```

Without bindings, all channels route to the default agent (`main`).

## Gateway Channels

```toml
[telegram]
allowed_chat_id = 42

[discord]
allowed_channel_id = "..."

[slack]
default_channel_id = "C0123456789"
```

Keep channel tokens in `.env` or the credential helpers; do not paste them into config examples. Use `gormes gateway status` to verify channel connections.
