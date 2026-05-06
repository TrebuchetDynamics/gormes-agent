---
title: "Telegram Bot"
description: "Run Gormes from Telegram and classify visible channel UX issues."
weight: 20
---

# Telegram Bot

The direct Telegram adapter still exists:

```bash
gormes telegram --help
```

The persistent fleet path should prefer gateway operation when multiple channels or restart management matter:

```bash
gormes gateway status
gormes whatsapp
```

Runtime-ready channel docs promote Telegram, Discord, and Slack after `gormes gateway status` confirms configuration. WhatsApp remains row-backed/fixture-backed until the live bridge path is promoted.

Operator checklist:

1. Create a bot token with BotFather.
2. Store the bot token in `.env` with `GORMES_TELEGRAM_TOKEN` or `TELEGRAM_BOT_TOKEN`.
3. Configure the allowed chat ID or allowed users through config helpers or non-secret TOML fields.
4. Verify config with `gormes config show`.
5. Verify runtime state with `gormes gateway status`.
6. Capture exact Telegram output for UX parity bugs.

Formatting issues such as visible Markdown markers, duplicated tool progress, raw internal labels, or wrong command menus are product bugs. Log them with the command, input, and visible channel output.
