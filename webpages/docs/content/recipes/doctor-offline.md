---
title: "Smoke-test offline with doctor"
description: "Prove the local Gormes runtime works without any provider credentials."
difficulty: "S"
---

# Smoke-test offline with doctor

> **Outcome:** A passing local readiness check for runtime identity, secrets, TUI, and the built-in tool catalog — no network, no credentials.
>
> **Prerequisites:** `gormes` installed on `$PATH`.

## Steps

1. **Run doctor in offline mode**
   ```bash
   gormes doctor --offline
   ```
   Lines beginning with `[PASS]` indicate healthy local subsystems. `[SKIP] provider health` is expected — it confirms the offline path skipped the network probe.

2. **Open the native TUI without provider calls (optional)**
   ```bash
   gormes --offline
   ```
   The Bubble Tea TUI opens with no provider health checks or network submits. Quit with `Ctrl-C`.

## Verify

```bash
gormes doctor --offline
```

Expected output (truncated):

```
[PASS] SecretRef runtime: resolved=0 inactive=0 unavailable=0
[SKIP] provider health: skipped (--offline)
[PASS] Native TUI: available: Go-native Bubble Tea TUI compiled into gormes
[PASS] Toolbox: 35 tools registered (...)
```

If every non-`[SKIP]` line reads `[PASS]`, the local runtime is healthy.

## Troubleshooting

- **`command not found`** → The binary is not on `$PATH`. Re-run the installer or follow [install from source](../../install/from-source/).
- **`[WARN] build identity: dirty build`** → You are running a locally built binary with uncommitted source. Safe to ignore for smoke tests.
- **`[FAIL]` on any local subsystem** → Run [diagnose a broken install](../diagnose/).

## See also

- [Connect a provider and open chat](../first-turn/)
- [Diagnose a broken install](../diagnose/)
