---
title: "Switch profiles for client work"
description: "Keep separate Gormes homes for different bots, projects, or clients."
difficulty: "S"
---

# Switch profiles for client work

> **Outcome:** Two or more isolated Gormes profiles, each with its own config, secrets, sessions, and memory store, switchable in one command.
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
   This builds a fresh `~/.gormes` lookalike directory for the profile. Add `--clone-all` to copy the default profile's non-runtime files into it.

3. **Switch to the profile**
   ```bash
   gormes profile use client-acme
   ```
   `profile set` is an accepted alias for `profile use`.

4. **Configure provider, model, and channels inside the profile**
   ```bash
   gormes auth add openai --api-key sk-...
   gormes setup model
   gormes config show
   ```
   Every command run while this profile is active reads and writes the profile's home only.

5. **Open chat under a specific profile without switching**
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
root: .../.gormes/profiles/client-acme
```

## Troubleshooting

- **`profile not found`** → Re-run `gormes profile list` to confirm the exact name, or create it with `gormes profile create <name>`.
- **Wrong profile picked up by a script** → Set the profile per-invocation with `--profile <name>` rather than relying on the persisted active profile.

## See also

- [Connect a provider and open chat](../first-turn/)
- [Migrate from Hermes or OpenClaw](../migrate-hermes/) — migrate into an isolated profile.
