---
title: "Troubleshooting"
description: "Diagnose Gormes by running doctor, looking up common errors, and reading the runtime log."
aliases:
  - /getting-started/troubleshooting/
  - /guides/debugging/
---

# Troubleshooting

When something is wrong, start with concrete evidence — the running binary, its config, and its log. Three pages cover the common paths.

Termux recovery note: `v0.2.21` carries the installer fix for the `v0.2.20` executable-argument failure that reported `unknown command /data/data/com.termux/files/usr/bin/gormes for gormes`. See [Common errors](./common-errors/) if an older Termux install is still on PATH.

| | |
|---|---|
| **[Doctor](./doctor/)** | What `gormes doctor` checks, online vs. offline runs, how to read the output |
| **[Common errors](./common-errors/)** | Symptom → likely cause → fix table |
| **[Logs](./logs/)** | Where `gormes.log` lives, how to rotate it, how to grep for tool errors |

The fastest first move is always:

```bash
which -a gormes
gormes version
gormes doctor --offline
```
