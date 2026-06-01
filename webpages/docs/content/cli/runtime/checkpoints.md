---
title: "gormes checkpoints"
description: "Inspect and manage Gormes file-operation rollback state"
---

# gormes checkpoints

View and control the filesystem checkpoint store at
`$XDG_DATA_HOME/gormes/checkpoints/`.

Commands mirror Hermes' `hermes checkpoints` CLI. None require the agent to be
running — safe to call at any time.

## Synopsis

```
gormes checkpoints [flags]
gormes checkpoints [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes checkpoints clear` | Delete the entire checkpoint base (all `/rollback` history) |
| `gormes checkpoints clear-legacy` | Delete only the `legacy-<ts>/` archives from v1 migration |
| `gormes checkpoints list` | Alias for `status` |
| `gormes checkpoints prune` | Delete orphan/stale checkpoints and GC the store |
| `gormes checkpoints status` | Show total size, project count, and per-project breakdown |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for checkpoints |

## See also

- [CLI reference](../)
