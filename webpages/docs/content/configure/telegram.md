---
title: "Channel credentials"
description: "Configure messaging channel credentials, starting with the Telegram bot-token path."
weight: 34
aliases:
  - /using-gormes/telegram-adapter/
  - /guides/telegram-bot/
---

# Channel credentials

This page owns channel credential setup. Gateway operation and routing live in
[Operate](../../operate/).

## Telegram

The Telegram adapter runs Gormes against a Telegram bot using long-poll
ingress (no webhook server needed). The same kernel, tools, and session
machinery the TUI uses run behind every Telegram message.

## Minimal Telegram setup

1. Create a bot with [@BotFather](https://t.me/BotFather) and copy the
   token it gives you.
2. Find your Telegram user ID (DM [@userinfobot](https://t.me/userinfobot))
   and the chat ID where the bot should answer. For DMs the chat ID matches
   your user ID.
3. Store the bot token in the dotenv and configure the allow-list. Either
   route both values to the config helpers or set them on the command line.

Using the config helpers:

```bash
gormes config set telegram.bot_token "123456:abc..."          # routed to .env
gormes config set telegram.allowed_user_ids "[123456789]"
gormes telegram
```

Using environment variables:

```bash
GORMES_TELEGRAM_BOT_TOKEN=123456:abc... \
GORMES_TELEGRAM_ALLOWED_USERS=123456789 \
TELEGRAM_HOME_CHANNEL=123456789 \
  gormes telegram
```

`gormes telegram` exits with a clear error when the token or allow-list is
missing. Capture the output and compare against `gormes config show` if a
setting is not taking effect.

## Token, chat, and user resolution

Gormes evaluates these env names in order; the first non-empty value wins,
and `[telegram]` config keys are the fallback.

| Setting | Variables (first non-empty wins) | Config field |
|---|---|---|
| Bot token | `GORMES_TELEGRAM_BOT_TOKEN`, `GORMES_TELEGRAM_TOKEN`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_TOKEN` | `[telegram].bot_token` |
| Home chat ID | `GORMES_TELEGRAM_CHAT_ID`, `TELEGRAM_HOME_CHANNEL`, `TELEGRAM_CHAT_ID` | `[telegram].allowed_chat_id` |
| Allowed users | `GORMES_TELEGRAM_ALLOWED_USERS`, `TELEGRAM_ALLOWED_USERS` | `[telegram].allowed_user_ids` |
| Allowed chats | `GORMES_TELEGRAM_ALLOWED_CHATS`, `TELEGRAM_ALLOWED_CHATS` | `[telegram].allowed_chats` |

The `GORMES_*` names are canonical; the legacy `TELEGRAM_*` and
`HERMES_TELEGRAM_*` aliases exist so a copied Hermes `.env` keeps working
after `gormes migrate hermes`.

## Allow-list, guest mode, and group chats

By default the adapter rejects every message that does not match the
allow-list. There are three knobs that loosen this:

| Knob | Purpose |
|---|---|
| `[telegram].allowed_chats` | Extra chat IDs beyond `allowed_chat_id`. Useful for group chats. |
| `[telegram].guest_mode` (env `GORMES_TELEGRAM_GUEST_MODE`) | Accept unknown senders in read-only mode. |
| `[telegram].require_mention` | When true, the bot only responds in groups when @mentioned. |

Discover chat and user IDs at runtime with:

```bash
gormes telegram --help
gormes config show
gormes gateway status
```

The adapter coalesces streaming edits to roughly one Telegram message update
per second (`[telegram].coalesce_ms = 1000` by default), which avoids
Telegram rate limits during long streamed responses.

## Notifications

`[telegram].notifications` (also `GORMES_TELEGRAM_NOTIFICATIONS` /
`HERMES_TELEGRAM_NOTIFICATIONS` / `TELEGRAM_NOTIFICATIONS`) controls the
notification severity Telegram receives. Default is `"important"`.

Setting it to `"all"` mirrors every assistant turn into a Telegram
notification; `"silent"` suppresses notifications entirely while still
delivering the message.

## Memory mirror and recall

The Telegram adapter is also the front-end for the memory mirror that writes
`$GORMES_HOME/memory/USER.md`. Tuning knobs live under `[telegram]`:

- `mirror_enabled`, `mirror_path`, `mirror_interval`
- `recall_enabled`, `recall_max_facts`, `recall_depth`,
  `recall_weight_threshold`, `recall_decay_horizon_days`
- `semantic_enabled` and the related `semantic_*` and `embedder_*` knobs for
  opt-in semantic fusion.

Defaults are listed in [Config file](../config-file/#telegram).

## Verifying the live adapter

```bash
gormes config show
gormes config check
gormes gateway status      # while a gateway is running
```

Telegram-specific UX bugs (wrong markdown, duplicated tool progress, missing
mentions) are product bugs — capture the exact input and the visible channel
output when reporting them.

## Discord and Slack

Discord and Slack gateways use the same runtime boundary. Use
`gormes gateway status` to verify configured channels and see
[Operate](../../operate/) for multi-channel workflows. Keep channel tokens in
`.env` or configured secret storage, not in copied documentation or chat logs.
