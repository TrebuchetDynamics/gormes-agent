---
title: "Diagnose a broken install"
description: "Use doctor, onboard, and gateway status to localize the failure."
difficulty: "S"
---

# Diagnose a broken install

> **Outcome:** A reproducible diagnosis (one failing subsystem, one error string) you can paste into an issue or hand to a maintainer.
>
> **Prerequisites:** `gormes` on `$PATH`.

## Steps

1. **Run doctor with full provider check**
   ```bash
   gormes doctor
   ```
   This runs the offline checks *and* the provider health probe. The first `[FAIL]` line names the failing subsystem.

2. **Compare with the offline subset**
   ```bash
   gormes doctor --offline
   ```
   If offline passes but full `doctor` fails, the issue is provider credentials, endpoint, or model selection — not the local runtime.

3. **Check first-run configuration state**
   ```bash
   gormes onboard
   ```
   `onboard` reports `Home`, `Config`, `Provider`, `Auth`, configured `Agents`, and `Bindings`. The `Next steps` block lists the commands needed to repair missing pieces.

4. **Inspect the gateway runtime (if you use channels)**
   ```bash
   gormes gateway status
   gormes logs
   ```
   `gateway status` shows the persisted runtime state and per-channel lifecycle errors. `logs` tails the most recent gateway log entries.

## Verify

```bash
gormes doctor --json
```

Expected: a `{checks: [...]}` JSON document. Every check has a `status` of `pass`, `skip`, or `fail`. Failing entries include the error reason in the same record — copy that into your issue report.

## Troubleshooting

- **`Auth: missing`** → Run `gormes auth add <provider>` (see [first turn](../first-turn/)).
- **`provider health: [FAIL]`** → Wrong endpoint, wrong model id, or expired credentials. Try [provider setup](../../guides/provider-setup/).
- **Gateway `lifecycle=failed`** → The channel adapter failed to start. The `error` field in `gateway status` names the cause (token conflict, missing allowlist, etc.).
- **`stale_pid` in gateway runtime** → A previous gateway crashed without clearing state. Run `gormes gateway stop` or remove the pid file under `~/.gormes`.

## See also

- [Smoke-test offline with doctor](../doctor-offline/)
- [Troubleshooting overview](../../troubleshooting/)
