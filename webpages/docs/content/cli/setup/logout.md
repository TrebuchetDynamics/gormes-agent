---
title: "gormes logout"
description: "Clear stored authentication for a Hermes-compatible provider"
---

# gormes logout

Clear stored authentication for a Hermes-compatible provider.

## Synopsis

```
gormes logout [--provider <provider>] [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for logout |
| `--json` | `false` | emit machine-readable JSON: `{build, action, provider, redacted}` |
| `--provider` | (none) | provider to log out from: `nous`, `openai-codex`, or `spotify` |

## See also

- [CLI reference](../../)
- [`gormes auth`](../auth/)
