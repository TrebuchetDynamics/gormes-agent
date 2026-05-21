---
title: "Profiles and workspaces"
description: "Configure Gormes profiles and workspace boundaries for client, project, and agent isolation."
aliases:
  - /configure/profiles/
  - /configure/workspaces/
---

Profiles separate Gormes state for clients, projects, or agents. Workspaces
control what project directories are exposed to the model/tool runtime.

## Basic commands

```bash
gormes profile create client-a
gormes profile use client-a
gormes profile show
```

Run a single command with an explicit profile:

```bash
gormes --profile client-a chat
```

## Workspace boundary

An empty `agents.defaults.workspaces` list means the operator home is the
default project workspace. A non-empty list is the model-facing project
read/write allow-list.

Use [Profiles for client work](../../operate/profiles-client-work/) for the operator workflow
and [CLI reference: profile](../../cli/profile/) for exact commands and flags.
