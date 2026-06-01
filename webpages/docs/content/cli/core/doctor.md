---
title: "gormes doctor"
description: "Verify Gormes runtime: provider readiness + built-in tools"
---

# gormes doctor

Verify Gormes runtime: provider readiness + built-in tools.

## Synopsis

```
gormes doctor [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for doctor |
| `--json` | `false` | emit a machine-readable `{checks: [...]}` JSON document (suitable for fleet-health monitoring) |
| `--offline` | `false` | skip the provider health check and validate local runtime checks |

## See also

- [CLI reference](../)
- [`gormes status`](../status/)
- [`gormes setup`](../setup/)
