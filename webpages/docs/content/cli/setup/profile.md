---
title: "gormes profile"
description: "Inspect and switch the active Gormes profile"
---

# gormes profile

Inspect and switch the active Gormes profile.

This page documents the current Gormes binary. `gormes profile --help` lists
both shipped commands and deterministic Hermes-parity placeholders, so status
matters: runtime-ready commands do real local work, while row-backed commands
return a typed unavailable result instead of silently pretending to be shipped.

## Profile model

A Gormes profile is a Gormes home root. Selecting a profile makes Gormes state
lookups use that root for config, secrets, sessions, memory, skills, logs,
cron, and gateway state.

Profile selection alone is state separation, not a blanket filesystem sandbox.
The model-facing workspace boundary is the Gormes-owned
`agents.defaults.workspaces` policy.

If `agents.defaults.workspaces` is empty, the default project workspace is the
operator home. If a profile config lists workspaces, local model-facing
read/write access is restricted to those normalized project roots. Runtime
internals still use the active profile root for config, auth, sessions, memory,
skills, logs, cron, and gateway state, but model-facing tools do not get
blanket project access to the whole profile root. The profile-owned editable
content is explicit: identity files such as `SOUL.md` and `IDENTITY.md`, plus
the active profile's `skills/` directory. Secrets and runtime state such as
`.env`, `auth.json`, session databases, memory databases, logs, and sibling
profiles stay outside the project workspace allow-list. File tools,
project-mode `execute_code`, and coding-agent delegation share that resolver.
A local terminal fails closed under a non-empty allow-list until a
sandbox-capable terminal backend is available; changing `cwd` is not enough to
make a shell a sandbox.

Upstream Hermes also creates wrapper commands and a profile-local subprocess
`HOME` at `HERMES_HOME/home/`. Gormes now creates `home/` for new profiles and
uses `GORMES_HOME/home` as subprocess `HOME` for local terminal and
`execute_code` shell execution when that directory exists. This is environment
separation only; it does not change the process working directory or replace
the separate workspace allow-list row. Gormes wrapper aliases remain row-backed
today.

## Synopsis

```
gormes profile [flags]
gormes profile [command]
```

## Runtime-ready subcommands

| Command | Purpose |
|---|---|
| `gormes profile create` | Create a named Gormes profile |
| `gormes profile info` | Show a profile distribution manifest |
| `gormes profile list` | List known Gormes profiles and mark the active one |
| `gormes profile show` | Show the active Gormes profile and its redacted root path |
| `gormes profile use` | Switch the active Gormes profile by name |

## Compatibility aliases

| Command | Purpose |
|---|---|
| `gormes profile set` | Compatibility alias for `gormes profile use`; JSON output preserves `action: "set"` for existing automation |

## Row-backed placeholders

These commands are registered for Hermes CLI parity, but their full behavior is
not shipped in Gormes yet. With `--json`, each returns
`action: "profile_command_unavailable"` and `status: "row_backed"`.

| Command | Purpose |
|---|---|
| `gormes profile alias` | Manage profile wrapper aliases |
| `gormes profile delete` | Delete a named Gormes profile |
| `gormes profile export` | Export a profile archive |
| `gormes profile import` | Import a profile archive |
| `gormes profile install` | Install a profile distribution |
| `gormes profile rename` | Rename a Gormes profile |
| `gormes profile update` | Update a profile distribution |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for profile |

## See also

- [CLI reference](../)
