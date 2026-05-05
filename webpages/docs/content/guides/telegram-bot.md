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
```

Operator checklist:

1. Create a bot token with BotFather.
2. Configure the allowed chat ID.
3. Verify config with `gormes config show`.
4. Verify runtime state with `gormes gateway status`.
5. Capture exact Telegram output for UX parity bugs.

Formatting issues such as visible Markdown markers, duplicated tool progress, raw internal labels, or wrong command menus are product bugs. Log them with the command, input, and visible channel output.
