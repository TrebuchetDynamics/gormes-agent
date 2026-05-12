---
title: "gormes onboard"
description: "First-run status — see what's configured and what to do next"
---

# gormes onboard

First-run status — see what's configured and what to do next.

## Synopsis

```
gormes onboard [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for onboard |
| `--json` | `false` | emit machine-readable JSON: `{build, home, config_path, provider, auth_configured, agents, bindings, ...}` |
| `--non-interactive` | `false` | render the wizard without prompts or external launches |
| `--wizard` | `false` | show the first-run wizard plan |

## See also

- [CLI reference](../)
- [`gormes setup`](../setup/)
- [`gormes doctor`](../doctor/)
