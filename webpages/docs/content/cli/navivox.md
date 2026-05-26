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
| `gormes navivox pair` | Start a local pairing bridge and hand Navivox a QR or Android deep link |

## Current Channel Contract

`gormes navivox pair` is the supported first-run setup command for the
Flutter Navivox app. It starts a network-reachable bridge, preferring a
Tailscale IPv4 address when one is detected, otherwise another VPN/LAN address
or an explicit `--host`. It prints the HTTP/WebSocket URLs, prints the pairing
token for manual fallback entry, writes `$GORMES_HOME/navivox/pairing.png`, and
prints a compact terminal QR when the current screen is wide enough. On
Android/Termux, `--open-navivox` hands the `navivox://connect` descriptor to
the Navivox Android app directly through `am start`, falling back to an Android
text-share payload when the VIEW intent fails. The QR image contains the base
URL, WebSocket URL, capability-document URL, auth mode, and REST token when
token auth is selected. Treat the printed token, QR, and PNG like WhatsApp Web
secret material. `--print-deeplink` prints the full secret descriptor only when
explicitly requested.

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
- Pairing URLs use a network-reachable IP by default; Tailscale IPv4 wins when
  detected.
- VPN-class exposure prints active VPN interface URLs, including Tailscale,
  WireGuard, and generic tun-class VPN interfaces.
- Public exposure is validated by server config and requires explicit
  confirmation.
- `pair` prints the pairing token for manual entry. Treat it like the QR: do
  not paste it into logs, screenshots, commits, or support bundles.
- Setup QR images may embed token values and are written with owner-only file
  permissions under `$GORMES_HOME/navivox/`.
- REST clients send `Authorization: Bearer <Navivox token>` for
  `/v1/navivox/*`; browser WebSocket clients may use the Navivox token
  subprotocol.
- Navivox clients should enable profile creation/import, attachment, voice,
  and stream UI affordances from `/v1/navivox/capabilities`, not by calling
  dashboard `/api/profiles` routes or assuming unsupported uploads exist.

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for navivox |

## See also

- [CLI reference](../)
