---
title: "gormes dashboard"
description: "Start the Gormes web dashboard"
---

# gormes dashboard

Starts an HTTP server with an htmx-based web dashboard for managing
sessions, config, skills, and logs.

## Synopsis

```
gormes dashboard [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for dashboard |
| `--host` | `127.0.0.1` | Dashboard HTTP server bind host |
| `--no-open` | `false` | do not open the dashboard in a browser |
| `--port` | `43827` | Dashboard HTTP server port |

## Connect Hermes Wing or Hermes Desktop

The dashboard API supports the core remote client contract: capability discovery,
session/history reads, run submission and SSE events, stop, and fail-closed
approval responses. Optional Desktop administration surfaces are not advertised.

Keep the default loopback bind for same-host clients. For a trusted VPN or
tailnet address, configure a strong key before binding externally:

```bash
export GORMES_DASHBOARD_API_KEY="$(openssl rand -hex 32)"
gormes dashboard --host 100.64.0.10 --port 43827 --no-open
```

Use `http://100.64.0.10:43827` as the client origin and the configured key as
the bearer/dashboard token. Gormes refuses non-loopback startup when the key is
missing or a known development placeholder. Prefer HTTPS through a trusted
reverse proxy when traffic is not already protected by a VPN or tailnet.

## See also

- [CLI reference](../../)
