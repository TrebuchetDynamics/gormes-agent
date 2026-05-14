# Navibox Tailnet Gateway Channel

Navibox connects to the native `gormes gateway` process over HTTP and
WebSocket. The channel is owned by the Gormes gateway: there is no SSH tunnel,
sidecar daemon, random extra server, or public unauthenticated API in the normal
app path. SSH remains an operator/admin break-glass option only.

## Trust Boundary

- The Gormes gateway owns routing, auth, logging, and runtime dispatch.
- Navibox sends typed HTTP/WebSocket actions; it does not get raw shell access.
- The channel is disabled by default and binds to loopback by default.
- Tailnet exposure is preferred for phones and tablets.
- Public exposure is discouraged and requires `navibox.public_confirmed=true`.
- Tokens are secrets. They belong in `.env` as `GORMES_NAVIBOX_TOKEN`, not in
  logs, screenshots, committed config, or support bundles.

## Setup

Run:

```sh
gormes setup gateway
```

Select `navibox`, then answer:

- `Enable Navibox Gateway Channel?`
- `Exposure mode`: `local`, `tailscale`, or `public`
- `Bind host`
- `Port`
- `Auth mode`: `pairing_token`, `static_token`, or `tailscale_identity`
- `Open firewall rule for this port now?`

Setup writes the channel config and generates a pairing token when token auth
is selected. It prints the HTTP/WebSocket URLs but does not print the token.

In the Flutter Navibox app setup screen, enter the gateway base URL, for
example `http://127.0.0.1:8765` or the Tailscale host URL, then enter the
pairing token. After connection, chat messages use the WebSocket stream.

## Config Keys

```toml
[navibox]
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
GORMES_NAVIBOX_TOKEN=...
```

## Exposure Modes

`local`
: Development default. Requires a loopback bind such as `127.0.0.1`.

`tailscale`
: Preferred production/mobile mode. Bind to the host's Tailscale IP or use
  Tailscale Serve in front of the selected local port.

`public`
: Discouraged. Requires `public_confirmed = true`; never enable this casually.

## Endpoints

- `GET /healthz`
- `GET /v1/navibox/status`
- `GET /v1/navibox/sessions`
- `GET /v1/navibox/sessions/{session_id}`
- `POST /v1/navibox/turn`
- `WS /v1/navibox/stream`

HTTP and WebSocket requests use bearer auth unless `auth_mode` is
`tailscale_identity`.

## WebSocket Messages

Client messages:

- `ping`
- `start_turn`
- `cancel_turn`
- `subscribe_session`

Server events:

- `pong`
- `session_started`
- `assistant_delta`
- `assistant_message`
- `tool_call_started`
- `tool_call_finished`
- `error`
- `done`

Every client request must include `request_id`. Runtime actions use
`session_id` when available.

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
curl -H "Authorization: Bearer $GORMES_NAVIBOX_TOKEN" \
  http://127.0.0.1:8765/v1/navibox/status
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

The Navibox channel does not expose arbitrary command execution. App actions
must map to typed gateway actions and are still constrained by normal Gormes
runtime safety controls.
