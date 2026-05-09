---
title: "Environment Variables"
description: "Environment variables that affect Gormes config, provider, gateway, and installer behavior."
weight: 30
---

# Environment Variables

## Core

| Variable | Purpose |
|---|---|
| `GORMES_HOME` | Runtime state/config root (default `~/.gormes`). |
| `GORMES_SKILLS_ROOT` | Custom skills directory override. |
| `GORMES_INFERENCE_PROVIDER` | Provider override for TUI/one-shot startup. |
| `GORMES_INFERENCE_MODEL` | Model override for TUI/one-shot startup. |
| `GORMES_ENDPOINT` | Provider endpoint override (invocation-only). |
| `GORMES_API_KEY` | Provider API key (invocation-only, not persisted for `--api-key` flag). |

## Browser

| Variable | Purpose |
|---|---|
| `BROWSER_CDP_URL` | Local browser/CDP endpoint for browser tools. |
| `CHROME_REMOTE_DEBUGGING_URL` | CDP endpoint alias for Chrome remote debugging. |

## Provider Credentials

Provider-specific API key variables (auto-detected):

| Variable | Provider |
|---|---|
| `OPENAI_API_KEY` | OpenAI |
| `ANTHROPIC_API_KEY` | Anthropic |
| `DEEPSEEK_API_KEY` | DeepSeek |
| `GROQ_API_KEY` | Groq |
| `OPENCODE_ZEN_API_KEY` | OpenCode Zen |
| `OPENCODE_GO_API_KEY` | OpenCode Go |
| `GOOGLE_API_KEY` / `GEMINI_API_KEY` | Gemini |
| `DASHSCOPE_API_KEY` | Alibaba / Qwen |
| `HF_TOKEN` | HuggingFace |
| `KIMI_API_KEY` | Kimi / Moonshot |
| `XAI_API_KEY` | xAI / Grok |

All provider credentials can also be managed via `gormes auth add <provider>`.

## Gateway Service Manager

| Variable | Purpose |
|---|---|
| `GORMES_GATEWAY_SERVICE_MANAGER` | Set to `1` or `true` to signal that a service manager (systemd/launchd) owns the gateway process. Enables exit-code-based restart. |

## Telegram Gateway

Gormes accepts the native `GORMES_TELEGRAM_*` names and the Hermes-compatible
`TELEGRAM_*` names so copied Hermes `.env` files keep working after migration.
Native names win when both are set.

| Variable | Purpose |
|---|---|
| `GORMES_TELEGRAM_TOKEN` / `TELEGRAM_BOT_TOKEN` | Telegram bot token. `TELEGRAM_TOKEN` remains a legacy alias. |
| `GORMES_TELEGRAM_CHAT_ID` / `TELEGRAM_HOME_CHANNEL` | Default Telegram chat/channel for gateway allowlisting and cron delivery. `TELEGRAM_CHAT_ID` remains a legacy alias. |
| `GORMES_TELEGRAM_ALLOWED_USERS` / `TELEGRAM_ALLOWED_USERS` | Comma-separated Telegram user IDs allowed to use the bot. |

## Installer

| Variable | Purpose |
|---|---|
| `GORMES_BRANCH` | Git branch cloned or updated by `install.sh` (default `main`). |
| `GORMES_INSTALL_HOME` | Managed install home (default `~/.gormes`). |
| `GORMES_INSTALL_DIR` | Managed source checkout directory override. |
| `GORMES_BIN_DIR` | Published binary directory override. |
| `GORMES_RESTART_GATEWAY` | Restart policy: `auto`, `always`, `never`. |
| `GORMES_SKIP_SETUP` | Skip the post-install `gormes setup` wizard when set to `1`, `true`, `yes`, or `on`. |
| `GORMES_GO_VERSION` | Go version for managed Go download (default `1.26.0`). |
| `GORMES_GO_SHA256` | Optional SHA-256 checksum for managed Go download. |
| `GORMES_SKIP_SERVICE` | Set to `1` to skip systemd/launchd service installation. |

## Dotenv

Secrets are stored in the dotenv file at:

```bash
gormes config env-path   # prints ~/.gormes/.env
```

Example `.env`:

```dotenv
GORMES_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
TELEGRAM_BOT_TOKEN=123:abc
TELEGRAM_ALLOWED_USERS=123456789
TELEGRAM_HOME_CHANNEL=123456789
```

Secret-like keys (`*_API_KEY`, `*_TOKEN`, `api_key`) are never written to `config.toml`. Use `gormes config set` or `gormes auth add` — the CLI routes secrets to `.env` automatically.
