---
title: "gormes curator"
description: "Manage Hermes-compatible background skill curation"
---

# gormes curator

Manage Hermes-compatible background skill curation.

## Synopsis

```
gormes curator [flags]
gormes curator [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes curator archive` | Archive one agent-created skill |
| `gormes curator backup` | Take a manual curator snapshot |
| `gormes curator list-archived` | List archived curator skills |
| `gormes curator pause` | Pause curator runs until resumed |
| `gormes curator pin` | Pin an agent-created skill so curator never auto-transitions it |
| `gormes curator prune` | Archive idle agent-created skills |
| `gormes curator restore` | Restore one archived skill |
| `gormes curator resume` | Resume curator runs |
| `gormes curator rollback` | Restore skills from a curator snapshot |
| `gormes curator run` | Trigger a curator review now |
| `gormes curator status` | Show curator status and skill stats |
| `gormes curator unpin` | Unpin an agent-created skill |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for curator |

## Learning loop role

The curator is the maintenance surface for agent-created skills in the
learning loop. Use `gormes curator status` to inspect review state, recent
summaries, archived skills, backups, and the operator review boundary before
trusting a skill as durable behavior.

## See also

- [CLI reference](../)
- [`gormes skills`](../skills/)
- [Learning loop proof](../../building-gormes/core-systems/learning-loop/)
