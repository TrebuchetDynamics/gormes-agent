---
title: "Run a Telegram bot"
description: "Long-poll Telegram from one chat through the same kernel and tool loop as the TUI."
difficulty: "M"
---

# Run a Telegram bot

> **Outcome:** Gormes long-polls Telegram for DMs from an allowlisted chat, runs the same kernel + tool loop as the TUI, and persists turns to the local SQLite memory store.
>
> **Prerequisites:** A working `gormes chat -q` (see [first turn](../first-turn/)). A bot token from [@BotFather](https://t.me/botfather). The numeric chat ID you want the bot to answer.

## Steps

1. **Store the bot token in `~/.gormes/.env`**
   ```bash
   gormes config set GORMES_TELEGRAM_TOKEN '123456:ABC-DEF...'
   ```
   `*_TOKEN` keys are written to `.env`, not `config.toml`. `TELEGRAM_BOT_TOKEN` is also accepted.

2. **Set the allowed chat id in `config.toml`**
   ```bash
   gormes config set telegram.allowed_chat_id 123456789
   ```
   Replace `123456789` with your numeric chat or group id. Without an allowlist (or `telegram.first_run_discovery = true`), the adapter refuses to start.

3. **Verify config**
   ```bash
   gormes config show
   ```
   The `telegram` section should list the allowed chat id; secrets stay redacted.

4. **Start the Telegram adapter**
   ```bash
   gormes telegram
   ```
   The process long-polls Telegram and routes messages from the allowed chat through the configured agent.

## Verify

In your Telegram chat, send a message to the bot. It should reply with a model-generated response. Then:

```bash
gormes session list
```

A new session row appears for the channel turn. Persisted turns survive a process restart.

## Troubleshooting

- **`no Telegram bot token`** → `GORMES_TELEGRAM_TOKEN`, `TELEGRAM_BOT_TOKEN`, or `[telegram].bot_token` is unset.
- **`no chat/user allowlist and discovery disabled`** → Set `telegram.allowed_chat_id`, `TELEGRAM_ALLOWED_USERS`, or temporarily `telegram.first_run_discovery = true`.
- **`telegram_polling_conflict`** → Another process is polling this bot token. Stop it before starting `gormes telegram`.
- **Visible Markdown markers or duplicate tool progress in chat** → Capture the channel output and the input that produced it; this is a UX parity bug, not a misconfiguration.

## See also

- [Run a multi-channel gateway](../multi-channel/) — same kernel, multiple channels.
- [Route channels to different agents](../bindings/)
- [Provider setup](../../guides/provider-setup/)
