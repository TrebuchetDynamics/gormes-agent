---
title: "gormes claw"
description: "Hermes-compatible OpenClaw migration tools"
---

# gormes claw

Hermes-compatible OpenClaw migration tools.

## Synopsis

```
gormes claw [flags]
gormes claw [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes claw cleanup` | Archive leftover OpenClaw directories under `HOME` by renaming them to `.pre-migration` variants |
| `gormes claw migrate` | Migrate OpenClaw state into Gormes using the Hermes `claw migrate` spelling |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for claw |

## See also

- [CLI reference](../../)
- [`gormes migrate`](../migrate/)
