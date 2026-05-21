# Navivox Tailnet Gateway Channel

Navivox connects to the native `gormes gateway` process over HTTP and
WebSocket. The channel is owned by the Gormes gateway: there is no SSH tunnel,
sidecar daemon, random extra server, or public unauthenticated API in the normal
app path. SSH remains an operator/admin break-glass option only.

## Trust Boundary

- The Gormes gateway owns routing, auth, logging, and runtime dispatch.
- Navivox sends typed HTTP/WebSocket actions; it does not get raw shell access.
- The channel is disabled by default and binds to loopback by default.
- Tailnet exposure is preferred for phones and tablets.
- Public exposure is discouraged and requires `navivox.public_confirmed=true`.
- Tokens are secrets. They belong in `.env` as `GORMES_NAVIVOX_TOKEN`, not in
  logs, screenshots, committed config, or support bundles.

## Setup

Run:

```sh
gormes setup gateway
```

Select `navivox`, then answer:

- `Enable Navivox Gateway Channel?`
- `Exposure mode`: `local`, `tailscale`, `wireguard`, `vpn`, or `public`
- `Bind host`
- `Port`
- `Auth mode`: `pairing_token`, `static_token`, `tailscale_identity`, or `token_and_tailscale_identity`
- `Open firewall rule for this port now?`

Setup writes the channel config and generates a pairing token when token auth
is selected. It prints the HTTP/WebSocket URLs but does not print the token.
`gormes navivox connect-info --json` also emits `base_url`, `healthz_url`, and
`websocket_url` for each reachable interface; IPv6 addresses are bracketed in
emitted URLs.

In the Flutter Navivox app setup screen, enter the gateway base URL, for
example `http://127.0.0.1:8765` or the Tailscale host URL, then enter the
pairing token. After connection, chat messages use the WebSocket stream.

## Config Keys

```toml
[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"
allow_origins = []
allowed_tailnet_identities = []
public_confirmed = false
```

Secret:

```sh
GORMES_NAVIVOX_TOKEN=...
```

## Exposure Modes

`local`
: Development default. Requires a loopback bind such as `127.0.0.1`.

`tailscale`
: Preferred production/mobile mode. Bind to the host's Tailscale IP or use
  Tailscale Serve in front of the selected local port.

`wireguard`
: VPN-class mode for WireGuard interfaces. The gateway startup path validates
  that `bind_host` matches an active WireGuard interface IP.

`vpn`
: Generic VPN-class mode for active Tailscale, WireGuard, or tun-class VPN
  interfaces. The gateway startup path validates that `bind_host` matches an
  active VPN interface IP.

`public`
: Discouraged. Requires `public_confirmed = true`; never enable this casually.

## Endpoints

- `GET /healthz`
- `GET /v1/navivox/status`
- `GET /v1/navivox/profile-contacts`
- `GET /v1/navivox/sessions`
- `GET /v1/navivox/sessions/{session_id}`
- `POST /v1/navivox/turn`
- `WS /v1/navivox/stream`

HTTP and WebSocket requests use bearer auth for `pairing_token`,
`static_token`, and `token_and_tailscale_identity`. `tailscale_identity` uses
Tailnet identity headers. `token_and_tailscale_identity` requires both layers.
Browser WebSocket clients may pass the bearer token through the supported
Navivox token subprotocol.

## WebSocket Messages

Client messages:

- `ping`
- `start_turn`
- `cancel_turn`
- `stop_turn`
- `subscribe_session`

Server events:

- `pong`
- `session_started`
- `assistant_delta`
- `assistant_message`
- `tool_call_started`
- `tool_call_finished`
- `safety_warning`
- `approval_required`
- `profile_contact_update`
- `error`
- `done`

Every client request must include `request_id`. Runtime actions use
`session_id` when available. Profile contact rows are keyed by `server_id` plus
`profile_id`; local-only metadata such as pins, aliases, command word,
calibration, and trusted-server state stays in the app.

Tool events are structured Navivox events, not assistant text. They include
`tool_call_id`, `tool_name`, `status`, and a bounded `message` summary. The
gateway deliberately avoids serializing raw tool arguments, stdout, secrets, or
full logs into these events. Non-Navivox channels keep their existing text
progress behavior.

## Firewall

Setup does not silently open firewall rules. This implementation records no
firewall changes and prints that no rules were changed.

If an operator chooses to open a local Linux firewall manually, use the narrowest
rule for the selected interface and port. Examples:

```sh
sudo ufw allow in on tailscale0 to any port 8765 proto tcp
sudo ufw delete allow in on tailscale0 to any port 8765 proto tcp
```

```sh
sudo firewall-cmd --add-port=8765/tcp --zone=trusted
sudo firewall-cmd --remove-port=8765/tcp --zone=trusted
```

Confirm the interface and firewall backend before running either example.

## Smoke Test

Local mode:

```sh
gormes gateway
curl http://127.0.0.1:8765/healthz
curl -H "Authorization: Bearer $GORMES_NAVIVOX_TOKEN" \
  http://127.0.0.1:8765/v1/navivox/status
```

WebSocket ping:

```json
{"type":"ping","request_id":"req-smoke"}
```

Expected response:

```json
{"type":"pong","request_id":"req-smoke"}
```

## Refusals

The Navivox channel does not expose arbitrary command execution. App actions
must map to typed gateway actions and are still constrained by normal Gormes
runtime safety controls.
