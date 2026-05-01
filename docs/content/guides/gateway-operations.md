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
```

Current available subcommands:

| Command | Status |
|---|---|
| `gormes gateway status` | Inspect configured channels and persisted runtime state. |
| `gormes gateway stop` | Stop the live gateway recorded in runtime status. |
| `gormes gateway start/restart/install/uninstall` | Unavailable in this CLI path; use the service restart helper where deployed. |

Document channel readiness from `gateway status`, not from roadmap rows. A channel can have fixtures or progress rows without being operator-ready in a live fleet.
