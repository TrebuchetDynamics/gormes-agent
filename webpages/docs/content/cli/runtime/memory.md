---
title: "gormes memory"
description: "Inspect persisted memory and extractor state"
---

# gormes memory

Inspect persisted memory and extractor state.

## Synopsis

```
gormes memory [flags]
gormes memory [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes memory status` | Show extractor queue depth and dead letters |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for memory |

## Learning loop role

Memory holds durable "what" facts for the learning loop. Use `gormes memory
status` to check extractor queue state and persistence evidence before treating
an assistant statement like "I will remember that" as durable memory.

## See also

- [CLI reference](../../)
- [`gormes goncho`](../goncho/)
- [`gormes session`](../session/)
- [Learning loop proof](../../../building-gormes/core-systems/learning-loop/)
