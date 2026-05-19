---
title: "Dashboard, status, and logs"
description: "Inspect a running Gormes runtime with dashboard, status, gateway status, and logs."
---

Use these commands for day-two operation. Troubleshooting owns the detailed log
guide; this page keeps the operator entry points together.

```bash
gormes status
gormes dashboard
gormes gateway status
gormes logs
```

## When to use each command

| Command | Use it for |
|---|---|
| `gormes status` | Current runtime state and progress blockers. |
| `gormes dashboard` | Local browser dashboard for sessions, config, skills, logs, and audits. |
| `gormes gateway status` | Configured channels and live gateway state. |
| `gormes logs` | Recent gateway/runtime log output. |

## Related references

- [Logs](../../troubleshooting/logs/) for file locations and grep patterns.
- [Doctor](../../troubleshooting/doctor/) for readiness checks.
- [Paths and logs](../../configure/paths/) for exact local state paths.
