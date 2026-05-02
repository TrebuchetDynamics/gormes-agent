---
title: "Paths and Logs"
description: "Native Gormes filesystem locations for config, secrets, sessions, memory, and logs."
weight: 60
---

# Paths and Logs

Ask the binary for active paths when possible:

```bash
gormes config path
gormes config env-path
```

Default native paths:

| Path | Purpose |
|---|---|
| `~/.gormes/config.toml` | Native config. |
| `~/.gormes/.env` | Local dotenv secrets. |
| `~/.gormes/sessions.db` | Session map. |
| `~/.gormes/memory.db` | Memory database. |
| `~/.gormes/gormes.log` | Runtime log path. |

Pages about migration may mention `~/.hermes`. Normal Gormes operation should document `~/.gormes`.
