---
title: "Configuration"
weight: 50
---

# Configuration

Gormes reads config from TOML files, env vars, and CLI flags — in that precedence.

## Config files

| Path | Purpose |
|---|---|
| `$XDG_CONFIG_HOME/gormes/config.toml` | User-level defaults |
| `./gormes.toml` | Project-local overrides (checked into the repo you're working in) |

Example:

```toml
[hermes]
endpoint = "http://127.0.0.1:8642"
api_key = ""
model = "claude-4-sonnet"

[input]
max_bytes = 65536
max_lines = 500
```

## Env vars

| Var | Purpose |
|---|---|
| `GORMES_ENDPOINT` | Provider endpoint override |
| `GORMES_API_KEY` | Provider API key override |
| `GORMES_TELEGRAM_TOKEN` / `TELEGRAM_BOT_TOKEN` | Telegram bot token |
| `GORMES_TELEGRAM_CHAT_ID` / `TELEGRAM_HOME_CHANNEL` | Telegram chat ID or home channel |
| `GORMES_TELEGRAM_ALLOWED_USERS` / `TELEGRAM_ALLOWED_USERS` | Comma-separated Telegram user IDs |

## State directories

| Path | Contents |
|---|---|
| `~/.gormes/sessions.db` | bbolt session resume map |
| `~/.gormes/memory.db` | SQLite memory store |
| `~/.gormes/memory/USER.md` | Human-readable entity/relationship mirror |
| `~/.gormes/gormes.log` | Runtime log |
