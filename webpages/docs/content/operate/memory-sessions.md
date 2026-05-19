---
title: "Memory and sessions"
description: "Inspect local Gormes sessions, SQLite memory, and durable context during operator workflows."
---

Gormes keeps sessions and durable context local. Operator docs lead with the
observable surfaces; deeper Goncho/Honcho compatibility details live in
[Concepts](../../concepts/) and [Build Gormes](../../building-gormes/).

## Inspect sessions

```bash
gormes session --help
gormes session list
gormes session export
```

Use session commands when you need to find prior turns, resume work, or export
conversation evidence for debugging.

## Inspect memory

```bash
gormes memory --help
gormes goncho --help
gormes config show
```

Use the memory and Goncho command surfaces to inspect persisted memory and
extractor state. Exact command flags live in the [CLI reference](../../cli/).

## Find local state

```bash
gormes config path
gormes config env-path
gormes config show
```

By default, Gormes stores config and state under `$GORMES_HOME`
(`~/.gormes`). See [Paths and logs](../../configure/paths/) for the exact files.

## Debug bad recall

1. Confirm the active profile with `gormes profile show`.
2. Confirm the expected workspace is configured.
3. Check `gormes config show` for memory-related settings.
4. Export or inspect the relevant session.
5. Use [Logs](../../troubleshooting/logs/) when memory extraction or recall fails.
