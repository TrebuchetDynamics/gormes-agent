---
title: "Debugging"
description: "Collect useful evidence before changing code or configuration."
weight: 60
---

# Debugging

Collect small, concrete evidence:

```bash
gormes version
which -a gormes
gormes config path
gormes config show
gormes doctor --offline
gormes gateway status
```

For channel bugs, save:

- the exact command or message sent;
- the visible reply;
- whether Mineru/Hermes and Gormes differed;
- the running binary path;
- the relevant config path.

For docs bugs, prefer a failing link, stale command, or conflicting claim over general wording feedback.
