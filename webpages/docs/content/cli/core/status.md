---
title: "gormes status"
description: "Show Gormes runtime and progress blockers"
---

# gormes status

Show Gormes runtime and progress blockers.

## Synopsis

```
gormes status [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for status |
| `--json` | `false` | emit a machine-readable `{blockers: [...]}` JSON document (suitable for monitoring/automation) |
| `--progress` | `webpages/docs/content/building-gormes/architecture_plan/progress.json` | `progress.json` path used for blocker status |

## See also

- [CLI reference](../)
- [`gormes doctor`](../doctor/)
- [`gormes setup`](../setup/)
