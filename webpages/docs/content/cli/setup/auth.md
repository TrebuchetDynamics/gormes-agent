---
title: "gormes auth"
description: "Manage Hermes-compatible provider credentials"
---

# gormes auth

Manage Hermes-compatible provider credentials.

## Synopsis

```
gormes auth [flags]
gormes auth [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes auth add` | Add a provider credential to the Hermes-compatible credential pool |
| `gormes auth list` | List provider credentials with secrets redacted |
| `gormes auth logout` | Clear provider credentials |
| `gormes auth remove` | Remove a provider credential by index, id, or label |
| `gormes auth reset` | Reset provider credential cooldown/exhaustion state |
| `gormes auth status` | Show redacted provider auth status |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for auth |
| `--json` | `false` | emit a machine-readable `{build, action: 'subcommand_required', parent, available, error}` document on stdout (the bare credential pool listing remains the default text output) |

## See also

- [CLI reference](../)
- [`gormes logout`](../logout/)
- [`gormes setup`](../setup/)
