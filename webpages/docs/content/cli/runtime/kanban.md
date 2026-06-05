---
title: "gormes kanban"
description: "Manage the durable local multi-agent Kanban board"
---

# gormes kanban

Manage the durable local multi-agent Kanban board.

## Synopsis

```
gormes kanban [flags]
gormes kanban [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes kanban block` | Block a Kanban task with a reason |
| `gormes kanban boards` | Manage named Kanban boards |
| `gormes kanban claim` | Atomically claim a ready Kanban task |
| `gormes kanban complete` | Mark a Kanban task done |
| `gormes kanban create` | Create a durable Kanban task |
| `gormes kanban gc` | Garbage-collect terminal Kanban events and worker logs |
| `gormes kanban init` | Initialize the local Kanban database |
| `gormes kanban link` | Add a dependency link |
| `gormes kanban list` | List durable Kanban tasks |
| `gormes kanban log` | Print the worker log for a Kanban task |
| `gormes kanban notify-list` | List Kanban notification subscriptions |
| `gormes kanban notify-subscribe` | Subscribe a gateway source to Kanban task events |
| `gormes kanban notify-unsubscribe` | Remove a Kanban notification subscription |
| `gormes kanban runs` | Show Kanban task run history |
| `gormes kanban show` | Show one Kanban task |
| `gormes kanban specify` | Flesh out a triage Kanban task with the configured model |
| `gormes kanban stats` | Show Kanban board status and assignee counts |
| `gormes kanban tail` | Follow a Kanban task's event stream |
| `gormes kanban unblock` | Unblock a Kanban task |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `--board` | (active) | operate on a named Kanban board for this invocation |
| `-h`, `--help` | | help for kanban |

## See also

- [CLI reference](../../)
