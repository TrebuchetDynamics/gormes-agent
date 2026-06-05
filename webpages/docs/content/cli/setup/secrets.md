---
title: "gormes secrets"
description: "Apply, audit, configure, and reload SecretRef-backed runtime secrets"
---

# gormes secrets

Apply, audit, configure, and reload SecretRef-backed runtime secrets.

## Synopsis

```
gormes secrets [flags]
gormes secrets [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes secrets apply` | Resolve a generated SecretRef plan into the runtime snapshot |
| `gormes secrets audit` | Audit plaintext secrets, unresolved refs, and snapshot precedence drift |
| `gormes secrets configure` | Build and preflight a typed SecretRef mapping for one config path |
| `gormes secrets reload` | Atomically re-resolve SecretRefs and keep the last-good snapshot on failure |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for secrets |

## See also

- [CLI reference](../../)
- [`gormes security`](../security/)
- [`gormes config`](../config/)
