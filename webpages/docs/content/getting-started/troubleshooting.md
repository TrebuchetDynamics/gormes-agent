---
title: "Troubleshooting"
description: "Start with symptoms, then run the smallest useful diagnostic command."
weight: 40
---

# Troubleshooting

Start with the current binary and local runtime state:

```bash
which -a gormes
gormes version
gormes doctor --offline
gormes config check
gormes gateway status
```

| Symptom | Check | Likely next step |
|---|---|---|
| Command behavior looks old | `which -a gormes` | Rebuild and run the intended binary path directly. |
| Config is not picked up | `gormes config path` | Check `GORMES_HOME` and edit the reported file. |
| Secrets are missing | `gormes config env-path` | Put provider tokens in the reported `.env` path. |
| Gateway is silent | `gormes gateway status` | Check configured channel credentials and runtime state. |
| Provider turn fails | `gormes doctor` | Verify provider, model, endpoint, and auth. |
| Browser tools fail | `gormes doctor --offline` | Install Chrome/Chromium, configure CDP, or install the browser harness. |
| Telegram output is ugly | Save the exact transcript | Treat it as a gateway UX parity issue with visible evidence. |
