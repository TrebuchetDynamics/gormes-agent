---
title: "Command Surface"
description: "How command parity should be documented and verified."
weight: 20
---

# Command Surface

Command parity spans several surfaces:

- native CLI commands;
- TUI slash commands;
- Telegram/Slack/channel commands;
- install/runtime commands;
- visible error and progress wording.

Docs should publish command behavior from the current binary and channel fixtures, not from command names alone.

Useful checks:

```bash
gormes --help
gormes config --help
gormes gateway --help
gormes setup --help
```
