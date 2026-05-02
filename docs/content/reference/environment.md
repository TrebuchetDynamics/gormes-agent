---
title: "Environment Variables"
description: "Environment variables that affect Gormes config, browser, provider, and gateway behavior."
weight: 30
---

# Environment Variables

Core variables:

| Variable | Purpose |
|---|---|
| `GORMES_HOME` | Overrides the native Gormes state/config root. |
| `GORMES_INFERENCE_PROVIDER` | Invocation-time provider override for TUI/one-shot startup. |
| `GORMES_INFERENCE_MODEL` | Invocation-time model override for TUI/one-shot startup. |
| `GORMES_ENDPOINT` | Invocation-time provider endpoint override. |
| `BROWSER_CDP_URL` | Local browser/CDP endpoint for browser tools. |
| `CHROME_REMOTE_DEBUGGING_URL` | CDP endpoint alias accepted for Chrome remote debugging. |

Provider and channel credentials vary by integration. Keep raw secrets in the dotenv path returned by:

```bash
gormes config env-path
```
