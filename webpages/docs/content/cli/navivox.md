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
| `gormes navivox connect` | Print Navivox connect URLs for active VPN/local interfaces |

## Current Channel Contract

`connect` is the supported token-redacted host-facing setup command for
the Flutter Navivox app. It prints base URLs the app can paste, plus the
matching health and WebSocket stream URLs. For first-run setup,
`gormes setup gateway` also writes `$GORMES_HOME/navivox/pairing.png`: a
scannable QR image containing a `navivox://connect` descriptor with the base
URL, WebSocket URL, auth mode, and REST token when token auth is selected.
Treat that PNG as secret material.

Supported runtime endpoints:

| Endpoint | Purpose |
|---|---|
| `GET /healthz` | Basic readiness probe |
| `GET /v1/navivox/status` | Authenticated channel status |
| `GET /v1/navivox/profile-contacts` | Authenticated profile contact snapshot |
| `GET /v1/navivox/sessions` | Authenticated session list |
| `GET /v1/navivox/sessions/{session_id}` | Authenticated session detail |
| `POST /v1/navivox/turn` | Authenticated text turn enqueue |
| `WS /v1/navivox/stream` | Authenticated event stream |

## Trust Boundaries

- The Navivox channel is disabled unless `[navivox].enabled` is true.
- Local exposure prints loopback URLs.
- VPN-class exposure prints active VPN interface URLs, including Tailscale,
  WireGuard, and generic tun-class VPN interfaces.
- Public exposure is validated by server config and requires explicit
  confirmation.
- Token values are never printed by `connect`; output only reports
  whether a token is required.
- Setup QR images may embed token values and are written with owner-only file
  permissions under `$GORMES_HOME/navivox/`.
- REST clients send `Authorization: Bearer <Navivox token>` for
  `/v1/navivox/*`; browser WebSocket clients may use the Navivox token
  subprotocol.
- JSON entries include `base_url`, `healthz_url`, and `websocket_url`; IPv6
  hosts are bracketed in emitted URLs.

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for navivox |

## See also

- [CLI reference](../)
