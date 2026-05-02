---
title: "Config Schema"
description: "Native Gormes config sections and where to inspect the active file."
weight: 20
---

# Config Schema

Inspect the active config:

```bash
gormes config path
gormes config show
gormes config check
```

Native config lives under `GORMES_HOME`, defaulting to `~/.gormes`.

Current top-level config sections in source include:

```text
hermes
gateway
display
tui
input
telegram
discord
slack
yuanbao
web
browser
security
cron
skills
delegation
goncho
```

Use generated config reference for exact fields before publishing a stable release promise.
