---
title: "Gateway Operations"
description: "Run and inspect configured messaging channels through the shared gateway runtime."
weight: 10
---

# Gateway Operations

The gateway command is the multi-channel operator surface.

```bash
gormes gateway --help
gormes gateway status
gormes gateway reload
gormes whatsapp
```

Runtime-ready channels are Telegram, Discord, and Slack when configured. WhatsApp is row-backed/fixture-backed until the live bridge bundle and gateway registration are promoted by runtime evidence.

Current available subcommands:

| Command | Status |
|---|---|
| `gormes gateway status` | Inspect configured channels and persisted runtime state. |
| `gormes gateway reload` | Ask a live gateway to reload swappable config without restarting. Invalid config keeps the last-good runtime config. |
| `gormes gateway stop` | Stop the live gateway recorded in runtime status. |
| `gormes gateway start/restart/install/uninstall` | Unavailable in this CLI path; use the service restart helper where deployed. |

Reload covers manager-level settings such as allowlists, first-run discovery flags, display/tool-progress modes, provider/model client routing, skills root, and agent bindings. Restart for binary updates, database path changes, or channel transport changes that require reconnecting clients.

Document channel readiness from `gateway status`, not from roadmap rows. A channel can have fixtures or progress rows without being operator-ready in a live fleet.
