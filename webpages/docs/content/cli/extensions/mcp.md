---
title: "gormes mcp"
description: "Manage Hermes-compatible MCP servers"
---

# gormes mcp

Manage Hermes-compatible MCP servers.

## Synopsis

```
gormes mcp [flags]
gormes mcp [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes mcp login` | Refresh OAuth login for an MCP server |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for mcp |
| `--json` | `false` | emit machine-readable JSON on invalid invocation: `{build, action: 'unknown_subcommand', error}` |

## See also

- [CLI reference](../)
- [`gormes plugins`](../plugins/)
