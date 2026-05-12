---
title: "gormes config"
description: "Inspect or update the Gormes configuration files"
---

# gormes config

Inspect or update the Gormes configuration files.

## Synopsis

```
gormes config [flags]
gormes config [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes config check` | Validate `config.toml` schema without writing — reports `_config_version`, missing/empty fields, and dotenv presence |
| `gormes config edit` | Open the Gormes `config.toml` in `$EDITOR` / `$VISUAL` or a discovered fallback |
| `gormes config env-path` | Print the Gormes dotenv (`.env`) secrets path |
| `gormes config get` | Read a configuration value (redacted for secret keys) |
| `gormes config migrate` | Apply native Gormes `config.toml` schema/default migrations |
| `gormes config path` | Print the Gormes TOML config path |
| `gormes config set` | Set a configuration value (TOML for non-secret keys, `.env` for `*_API_KEY`/`*_TOKEN`/`api_key`) |
| `gormes config show` | Show resolved Gormes configuration with secrets redacted |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for config |

## See also

- [CLI reference](../)
- [Config schema](../../configure/config-file/)
- [Environment variables](../../configure/environment/)
