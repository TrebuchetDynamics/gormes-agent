---
title: "gormes skills"
description: "Manage skills"
---

# gormes skills

Manage skills.

## Synopsis

```
gormes skills [flags]
gormes skills [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes skills install` | Install a skill from a direct SKILL.md URL |
| `gormes skills list` | List installed skills |
| `gormes skills sync` | Sync bundled skills into all configured profiles |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for skills |

## Learning loop role

Skills hold reusable "how" knowledge for the learning loop. Use `gormes skills
list` to confirm that a repeated workflow became an inspectable SKILL.md
surface instead of staying buried in one chat transcript.

## See also

- [CLI reference](../)
- [`gormes curator`](../curator/)
- [`gormes plugins`](../plugins/)
- [Learning loop proof](../../building-gormes/core-systems/learning-loop/)
