---
title: "Switch profiles for client work"
description: "Keep separate Gormes homes for different bots, projects, or clients."
difficulty: "S"
---

# Switch profiles for client work

> **Outcome:** Two or more separate Gormes profile homes, each with its own config, secrets, sessions, and memory store, switchable in one command.
>
> **Prerequisites:** `gormes` installed.

## Steps

1. **List existing profiles**
   ```bash
   gormes profile list
   ```
   `*` marks the active profile.

2. **Create a new profile**
   ```bash
   gormes profile create client-acme
   ```
   This builds a fresh named profile under the active Gormes home. Creating a profile does not make it active. Add `--clone-all` to copy profile data from the default Gormes home while excluding infrastructure and runtime process files.

3. **Switch to the profile**
   ```bash
   gormes profile use client-acme
   ```
   `profile set` is an accepted alias for `profile use`.

4. **Record profile workspaces and channels**
   ```bash
   gormes setup profiles
   ```
   The interactive flow can select the active profile and persist
   `agents.defaults.workspaces` plus `agents.defaults.channels` into that
   profile's own `config.toml`. A non-empty workspace list is enforced as the
   model-facing project allow-list.

5. **Configure provider and model inside the profile**
   ```bash
   gormes auth add openai --api-key sk-...
   gormes setup model
   gormes config show
   ```
   Gormes state commands run against the active profile home. Profile
   selection separates config, secrets, sessions, memory, skills, logs, cron,
   and gateway state. The Gormes workspace policy treats an empty
   `agents.defaults.workspaces` list as the operator home, and a non-empty list
   as the project read/write allow-list for file tools, project-mode
   `execute_code`, and coding-agent delegation. Local terminal and
   `execute_code` shell subprocesses use the active profile's `home/`
   directory as `HOME` when that directory exists. Runtime internals still own
   the active profile root for state, but model-facing profile edits are
   limited to explicit profile-owned content: identity files such as `SOUL.md`
   and `IDENTITY.md`, plus the active profile's `skills/` directory. Secrets,
   sessions, memory databases, logs, and sibling profiles are not project
   workspaces.

6. **Open chat under a specific profile without switching**
   ```bash
   gormes --profile client-acme chat
   ```

## Verify

```bash
gormes profile show
```

Expected output:

```
active profile: client-acme
root: .../client-acme
```

## Troubleshooting

- **`profile not found`** → Re-run `gormes profile list` to confirm the exact name, or create it with `gormes profile create <name>`.
- **Wrong profile picked up by a script** → Set the profile per-invocation with `--profile <name>` rather than relying on the persisted active profile.
- **Shell tools still see the operator home** → Confirm the active profile has a `home/` directory and the command is running under `gormes --profile <name>` or after `gormes profile use <name>`. Legacy profile roots created before this parity slice may need `mkdir -p <profile-root>/home`, using the root printed by `gormes profile show`.
- **Terminal is blocked after adding workspaces** → A non-empty
  `agents.defaults.workspaces` list makes the local terminal fail closed until
  a sandbox-capable terminal backend is configured. Use file tools or
  project-mode `execute_code` only when the command can be represented through
  a confined surface.
- **Profile files are still broad state** → Editing profile identity and skills
  is intended, but the allow-list does not make the whole profile root a
  model-facing workspace. Keep secrets and runtime state out of normal file
  tool workflows.

## See also

- [Connect a provider and open chat](../first-turn/)
- [Migrate from Hermes or OpenClaw](../migrate-hermes/) — migrate into a separate profile home.
