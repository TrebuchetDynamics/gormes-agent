---
title: "gormes usage"
description: "Show runtime/provider account usage"
---

# gormes usage

Show runtime/provider account usage.

## Synopsis

```
gormes usage [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `--account-id` | (none) | provider account identifier when required |
| `--api-key` | (configured hermes `api_key`) | provider API/OAuth token for account usage |
| `--base-url` | (none) | provider account usage base URL override |
| `-h`, `--help` | | help for usage |
| `--json` | `false` | emit machine-readable JSON: `{build, provider, account_id, plan, source, fetched_at, windows: [...], details, unavailable}` |
| `--provider` | (none) | provider account usage to query (`openai-codex`, `anthropic`, `openrouter`) |

## See also

- [CLI reference](../../)
- [Providers](../../../configure/providers/)
- [`gormes gateway`](../../runtime/gateway/) (see `gateway usage-cost`)
