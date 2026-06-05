---
title: "gormes providers"
description: "Show provider setup commands"
---

# gormes providers

Show provider setup commands from the checked-in Hermes provider manifest.
This is Gormes-owned operator guidance around the Hermes-compatible provider
surfaces: `gormes setup provider`, `gormes auth add`, `gormes model`, and
`gormes doctor --offline`.

## Synopsis

```
gormes providers [provider] [flags]
gormes providers setup [provider]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes providers setup [provider]` | Show focused setup commands for one provider, or list manifest providers |

## Examples

```bash
gormes providers setup openrouter
gormes providers setup openai-codex
gormes providers setup bedrock
```

API-key providers show the non-interactive environment variables and a
credential-pool command. OAuth providers show the supported `gormes auth add
<provider> --type oauth` path when the adapter exists. Row-backed providers
print setup intent plus backlog guidance instead of failing as unknown.

## See also

- [CLI reference](../../)
- [`gormes auth`](../auth/)
- [`gormes model`](../model/)
- [`gormes setup`](../setup/)
- [Provider configuration](../../../configure/providers/)
