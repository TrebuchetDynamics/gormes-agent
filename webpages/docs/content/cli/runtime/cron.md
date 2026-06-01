---
title: "gormes cron"
description: "Manage scheduled cron jobs"
---

# gormes cron

List, inspect, and remove cron jobs managed by the native cron scheduler.

Cron jobs run agent prompts or shell scripts on a schedule and deliver
results to configured targets (Telegram, Slack, etc.).

## Synopsis

```
gormes cron [flags]
gormes cron [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes cron list` | List all cron jobs with status and last run time |
| `gormes cron remove` | Remove a cron job by ID (prefix matching supported) |
| `gormes cron status` | Show detailed job info and recent run history |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `--db` | (autodetect from `GORMES_HOME`) | Path to session DB |
| `-h`, `--help` | | help for cron |

## Examples

```bash
gormes cron list              # show all cron jobs
gormes cron status <job-id>   # show job details
gormes cron remove <job-id>   # remove a cron job
```

## See also

- [CLI reference](../../)
