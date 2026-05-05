---
title: "Browser / CDP"
description: "Connect browser tools to local Chrome or Chromium through CDP."
weight: 50
---

# Browser / CDP

Browser tools need a browser backend. Local Chrome/Chromium through CDP is the operator-friendly path when API extractors cannot read a page.

Start with diagnostics:

```bash
gormes doctor --offline
```

CDP endpoints are configured through browser settings or environment aliases such as:

```bash
export BROWSER_CDP_URL=http://127.0.0.1:9222
export CHROME_REMOTE_DEBUGGING_URL=http://127.0.0.1:9222
```

Use a dedicated Chrome profile when launching remote debugging so an existing desktop browser process does not ignore the port.

This page should eventually include OS-specific Chrome install and launch commands after they are verified on Linux, macOS, Windows, WSL, and Termux.
