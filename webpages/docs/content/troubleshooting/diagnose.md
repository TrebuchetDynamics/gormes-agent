---
title: "Diagnose a broken install"
description: "Use doctor, setup, and gateway status to localize the failure."
difficulty: "S"
aliases:
  - /recipes/diagnose/
---

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
   gormes doctor --offline --target terminal --json
   ```
   The `target` block reports readiness, missing setup pieces, and `next_command`. Run `gormes setup --quick --target terminal` when the target is not ready.

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

Expected: a `{build, failed, checks: [...]}` JSON document. Every check has a `name`, a `status` of `PASS`, `WARN`, `SKIP`, or `FAIL`, and a `summary`. The `summary` of a failing check carries the error reason — copy that into your issue report.

## Troubleshooting

- **Termux latest-release `v0.2.20` reports `unknown command /data/data/com.termux/files/usr/bin/gormes for gormes`** → This is the known live release caveat: the fix is already committed on `development`, but it is not in the public latest release. Do not report Termux latest install as repaired until a follow-up release exists.
- **`Auth: missing`** → Run `gormes auth add <provider>` (see [first chat](../../operate/first-chat/)).
- **`provider health: [FAIL]`** → Wrong endpoint, wrong model id, or expired credentials. Try [provider setup](../../configure/providers/).
- **Gateway `lifecycle=failed`** → The channel adapter failed to start. The `error` field in `gateway status` names the cause (token conflict, missing allowlist, etc.).
- **`stale_pid` in gateway runtime** → A previous gateway crashed without clearing state. Run `gormes gateway stop` or remove the pid file under `~/.gormes`.

## See also

- [Smoke-test offline with doctor](../../recipes/doctor-offline/)
- [Troubleshooting overview](../)
