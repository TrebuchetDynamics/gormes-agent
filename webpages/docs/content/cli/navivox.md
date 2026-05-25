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
matching health, capability-document, and WebSocket stream URLs. For first-run
setup, `gormes setup gateway` also writes `$GORMES_HOME/navivox/pairing.png`:
a scannable QR image containing a `navivox://connect` descriptor with the base
URL, WebSocket URL, capability-document URL, auth mode, and REST token when
token auth is selected. Treat that PNG as secret material.

Supported runtime endpoints:

| Endpoint | Purpose |
|---|---|
| `GET /healthz` | Basic readiness probe |
| `GET /v1/navivox/status` | Authenticated channel status plus lightweight capability names |
| `GET /v1/navivox/capabilities` | Authenticated versioned Navivox capability document |
| `GET /v1/navivox/profile-contacts` | Authenticated profile contact snapshot |
| `GET /v1/navivox/profile-routing` | Authenticated server/profile routing snapshot |
| `POST /v1/navivox/profile-seed` | Draft or apply a profile from operator text |
| `GET /v1/navivox/config-admin[/schema]` | Authenticated safe config admin read/schema |
| `POST /v1/navivox/config-admin/{diff,validate,apply}` | Authenticated safe config admin mutations |
| `GET /v1/navivox/voice-profiles` | Authenticated per-profile STT/TTS voice profile state |
| `POST /v1/navivox/voice-profiles/validate` | Authenticated voice profile validation |
| `GET /v1/navivox/run-records/{run_id_or_session_id}` | Authenticated run-record lookup |
| `GET /v1/navivox/memory/overview` | Authenticated bounded memory overview |
| `GET /v1/navivox/sessions` | Authenticated session list |
| `GET /v1/navivox/sessions/{session_id}` | Authenticated session detail |
| `POST /v1/navivox/turn` | Authenticated text turn enqueue |
| `WS /v1/navivox/stream` | Authenticated canonical Navivox event stream |

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
- JSON entries include `base_url`, `healthz_url`, `capabilities_url`, and
  `websocket_url`; IPv6 hosts are bracketed in emitted URLs.
- Navivox clients should enable profile creation/import, attachment, voice,
  and stream UI affordances from `/v1/navivox/capabilities`, not by calling
  dashboard `/api/profiles` routes or assuming unsupported uploads exist.

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for navivox |

## See also

- [CLI reference](../)
