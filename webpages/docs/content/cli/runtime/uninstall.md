---
title: "gormes uninstall"
description: "Remove Gormes artifacts from this system"
---

# gormes uninstall

Enumerates every artifact Gormes wrote. Dry-run by default; use `--yes` to
confirm.

## Synopsis

```
gormes uninstall [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `--dry-run` | `true` | List artifacts without deleting |
| `-h`, `--help` | | help for uninstall |
| `--json` | `false` | emit machine-readable JSON: dry-run returns `{build, action: 'preview', dry_run, total, groups: [...]}`; `--yes --dry-run=false` returns `{build, action: 'uninstalled', dry_run: false, removed, failed, groups: [...]}` |
| `--keep-config` | `false` | Preserve config files |
| `--keep-credentials` | `false` | Preserve credential pool |
| `--yes` | `false` | Confirm destructive removal |

## See also

- [CLI reference](../../)
