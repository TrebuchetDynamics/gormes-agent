---
title: "gormes navivox"
description: "Navivox HTTP channel utilities"
---

# gormes navivox

Navivox HTTP/WebSocket channel utilities.

## Synopsis

```
gormes navivox [flags]
gormes navivox [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes navivox connect-info` | Print Navivox connect URLs for active VPN/local interfaces |

## Current Channel Contract

`connect-info` is the supported host-facing setup command for the Flutter
Navivox app. It prints base URLs the app can paste or scan, plus the matching
health URL.

Supported runtime endpoints:

| Endpoint | Purpose |
|---|---|
| `GET /healthz` | Basic readiness probe |
| `GET /v1/navivox/status` | Authenticated channel status |
| `GET /v1/navivox/sessions` | Authenticated session list |
| `GET /v1/navivox/sessions/{session_id}` | Authenticated session detail |
| `POST /v1/navivox/turn` | Authenticated text turn enqueue |
| `WS /v1/navivox/stream` | Authenticated event stream |

## Trust Boundaries

- The Navivox channel is disabled unless `[navivox].enabled` is true.
- Local exposure prints loopback URLs.
- VPN-class exposure prints active VPN interface URLs.
- Public exposure is validated by server config and requires explicit
  confirmation.
- Token values are never printed; output only reports whether a token is
  required.

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for navivox |

## See also

- [CLI reference](../)
