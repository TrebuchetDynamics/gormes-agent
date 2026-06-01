---
title: "gormes update"
description: "Update a managed Gormes source checkout"
---

# gormes update

Update a managed Gormes source checkout: fetch + fast-forward, sync
bundled skills, rebuild the web UI, run a config schema migration check,
and restart the gateway.

## Backup and rollback

```bash
gormes update --backup           # take a pre-update zip of GORMES_HOME
                                 # before pulling source. Set
                                 # `[updates] pre_update_backup = true`
                                 # in config.toml to make this the default.
gormes restore --list            # enumerate available pre-update zips,
                                 # newest first.
gormes restore --latest --yes    # roll back to the most recent zip
                                 # (overwrites files in GORMES_HOME).
```

When `--backup` is set and the update later fails, the report ends with a
`update_rollback_hint` line spelling out the restore command above so the
recovery path is visible inline.

## Synopsis

```
gormes update [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `--backup` | `false` | create a single-run pre-update backup zip of `~/.gormes`; restore later with `gormes restore --latest --yes` |
| `--branch` | `main` | Git branch to fetch and fast-forward |
| `--check` | `false` | check update readiness without mutating the checkout |
| `-h`, `--help` | | help for update |
| `--json` | `false` | emit a machine-readable JSON report instead of the human-readable progress UX (suitable for CI/cron consumers) |
| `--kill-stale-dashboard` | `false` | stop stale dashboard processes after a successful update |
| `--no-backup` | `false` | force-skip the pre-update backup; beats `--backup` and config opt-in |
| `--restart-gateway` | `auto` | restart policy for a live gateway: `auto`, `always`, or `never` |
| `--skip-web` | `false` | skip the web UI rebuild step after pulling source |
| `-y`, `--yes` | `false` | assume yes for non-destructive recovery prompts |

## See also

- [CLI reference](../../)
- [`gormes restore`](../restore/)
