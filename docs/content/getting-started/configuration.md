---
title: "Configuration"
description: "Understand GORMES_HOME, config.toml, dotenv secrets, and local state."
weight: 30
---

# Configuration

Gormes uses a native home directory. `GORMES_HOME` wins; otherwise Gormes defaults to `~/.gormes`.

```bash
gormes config path
gormes config env-path
gormes config show
gormes config check
```

Default paths:

| Item | Default |
|---|---|
| Config | `~/.gormes/config.toml` |
| Secrets dotenv | `~/.gormes/.env` |
| Sessions DB | `~/.gormes/sessions.db` |
| Memory DB | `~/.gormes/memory.db` |
| Log | `~/.gormes/gormes.log` |

Use `gormes config set` for supported keys. Secret-like keys are routed to `.env` instead of `config.toml`.

```bash
gormes config set hermes.provider openai-codex
gormes config set hermes.model gpt-5.5
gormes config show
```

Do not document `~/.hermes` as the active Gormes state root unless the page is explicitly about migrating from Hermes.
