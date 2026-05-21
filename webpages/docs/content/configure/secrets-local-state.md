---
title: "Secrets and local state"
description: "How Gormes separates config, secrets, profiles, sessions, memory, and logs under GORMES_HOME."
aliases:
  - /configure/secrets/
  - /security/secrets/
---

Gormes keeps configuration inspectable and routes secrets away from
`config.toml`.

| File or directory | Purpose |
|---|---|
| `$GORMES_HOME/config.toml` | Non-secret configuration. |
| `$GORMES_HOME/.env` | Provider keys, bot tokens, and other secrets. |
| `$GORMES_HOME/profiles/` | Profile-local config and state. |
| `$GORMES_HOME/sessions/` | Persisted session state. |
| `$GORMES_HOME/memory/` | Local memory and mirrors. |
| `$GORMES_HOME/gormes.log` | Runtime log. |

Default `GORMES_HOME` is `~/.gormes`.

## Write settings safely

Prefer `gormes config set`. Secret-looking keys are routed to `.env`
automatically.

```bash
gormes config set hermes.provider openai
gormes config set hermes.api_key sk-...
gormes config show
gormes config check
```

Do not commit or paste `.env`, provider keys, bot tokens, profile secrets, or
session exports that contain private data.

## Related reference

- [Config file](../config-file/) for exact TOML fields.
- [Environment variables](../environment/) for supported env names.
- [Paths and logs](../paths/) for exact local state paths.
