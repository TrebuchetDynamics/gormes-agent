---
title: "Telegram Adapter"
weight: 40
---

# Telegram Adapter

Run Gormes as a Telegram bot. Same kernel, same tools, different edge.

## Setup

1. Create a bot with [@BotFather](https://t.me/BotFather) — get the token
2. Get your chat ID (DM [@userinfobot](https://t.me/userinfobot))
3. Launch:

```bash
TELEGRAM_BOT_TOKEN=... TELEGRAM_ALLOWED_USERS=123456789 TELEGRAM_HOME_CHANNEL=123456789 gormes telegram
```

`GORMES_TELEGRAM_TOKEN`, `GORMES_TELEGRAM_CHAT_ID`, and
`GORMES_TELEGRAM_ALLOWED_USERS` are the native names. Gormes also accepts
Hermes-compatible `TELEGRAM_BOT_TOKEN`, `TELEGRAM_HOME_CHANNEL`, and
`TELEGRAM_ALLOWED_USERS` for copied `.env` files and `gormes migrate hermes`.

## Behaviour

- Long-poll ingress (no webhook server needed)
- Edit coalescer: streamed tokens update the same Telegram message at ~1 Hz to avoid rate limits
- Session resume: each `(platform, chat_id)` maps to a persistent session_id via bbolt

## Multiple chats

Use `TELEGRAM_ALLOWED_USERS` to allow specific users across DMs and groups. Set
`TELEGRAM_HOME_CHANNEL` when cron or proactive delivery needs a default chat.
