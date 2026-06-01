---
title: "gormes migrate"
description: "Migrate state from upstream agents into Gormes (dry-run only in this slice)"
---

# gormes migrate

Migrate state from upstream agents into Gormes (dry-run only in this slice).

## Synopsis

```
gormes migrate [flags]
gormes migrate [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes migrate hermes` | Migrate Hermes `config.yaml` + `.env` into Gormes (dry-run manifest or `--yes` apply) |
| `gormes migrate openclaw` | Migrate OpenClaw config, env, memory, user, and skill surfaces into Gormes (dry-run manifest or `--yes` apply) |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for migrate |

## See also

- [CLI reference](../../)
- [`gormes claw`](../claw/)
