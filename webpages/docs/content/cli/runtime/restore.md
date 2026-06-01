---
title: "gormes restore"
description: "Discover and restore from a pre-update backup zip"
---

# gormes restore

Discover and restore from a pre-update backup zip.

## Synopsis

```
gormes restore [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for restore |
| `--json` | `false` | emit machine-readable JSON: `--list` returns `{build, backups: [...]}`; dry-run returns `{build, action: 'preview', path, dest, dry_run, would_overwrite, would_create}`; `--yes` returns `{build, action: 'restored', path, dest, file_count, overwrote, created}` |
| `--latest` | `false` | restore the newest pre-update backup (resolved by `mtime`) |
| `--list` | `false` | list available pre-update backup zips, newest first |
| `--path` | (none) | path to a `pre-update-*.zip` to restore (overwrites files in `GORMES_HOME`) |
| `-y`, `--yes` | `false` | confirm the destructive restore (without `--yes` the command runs as a dry-run preview) |

## See also

- [CLI reference](../)
- [`gormes update`](../update/)
