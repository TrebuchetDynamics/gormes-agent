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
   profile's own `config.toml`. Current releases round-trip the workspace list
   but do not yet enforce it as an access boundary.

5. **Configure provider and model inside the profile**
   ```bash
   gormes auth add openai --api-key sk-...
   gormes setup model
   gormes config show
   ```
   Gormes state commands run against the active profile home. Current profiles
   are not enforced filesystem sandboxes, and selecting one does not change the
   project working directory for shell tools. The planned Gormes workspace
   policy treats an empty `agents.defaults.workspaces` list as the operator
   home, and a non-empty list as the project read/write allow-list while still
   allowing the active profile root for profile state.

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
- **Shell tools still see the operator home** → Current Gormes profile roots do not yet provide Hermes-style profile-local subprocess `HOME`; that parity slice is planned.
- **Workspace list does not restrict access yet** → `gormes setup profiles`
  persists `agents.defaults.workspaces` today, but runtime enforcement is still
  row-backed. Until that slice ships, do not treat the list as a sandbox.

## See also

- [Connect a provider and open chat](../first-turn/)
- [Migrate from Hermes or OpenClaw](../migrate-hermes/) — migrate into a separate profile home.
