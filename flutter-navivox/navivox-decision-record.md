# Navivox Decision Record

Status: accepted baseline
Date: 2026-05-05
Scope: Flutter app plus Gormes `navivox` channel planning

This file is the canonical decision surface for the current Navivox planning
docs. The PRD, architecture, route, data-model, UI, library-research, and
testing-plan docs may expand these decisions, but should not contradict them.

## 1. Chat UI Foundation

Navivox will use `flyerhq/flutter_chat_ui` v2 as the chat foundation.

Rationale:

- It is backend-agnostic, which matches an SSH stdio protocol instead of a
  Firebase or vendor-hosted chat backend.
- It has a modular `flutter_chat_core` / `flutter_chat_ui` split and a builder
  system for replacing message renderers.
- `flyer_chat_text_stream_message` maps cleanly to `chat.update` events when
  stream state is owned by the Navivox channel/provider layer.

Navivox-specific extensions:

- `ToolCallCard`: rendered through custom message builders, never as raw logs
  inside assistant text.
- `VoiceMessageBubble`: waveform, playback, transcript, and confidence/status
  display for voice turns.
- `AgentSwitcherMessage`: an inline system/control message for agent switch
  events and local-command confirmations. The global agent picker remains a
  sheet/menu, not a chat bubble.

Implementation boundary:

- The Flutter chat controller mirrors server events and local cache state.
  Agent orchestration, tools, approvals, and model calls remain server-side in
  Gormes.

## 2. App Architecture Stack

Navivox will use Riverpod + GoRouter + Drift + Freezed.

Rationale:

- Riverpod owns connection, channel, chat, config, voice, and routing state with
  testable providers.
- GoRouter owns URL-shaped routes, shell routes, redirects, and deep links.
- Drift owns local SQLite cache for servers, identities, host keys, messages,
  tool calls, agents, config schema/cache, and settings.
- Freezed owns immutable domain models, protocol unions, and JSON codecs.

Folder structure:

- Keep the existing feature-first plan: `core/`, `data/`, `features/`,
  `router/`, and `shared/`.
- Use `features/<feature>/{providers,screens,widgets}` for UI-facing features.
- Keep protocol, SSH, secure storage, and crypto in `core/`; keep Drift tables,
  DAOs, repositories, and import mappers in `data/`.

## 3. Protocol Framing

Navivox will use binary-safe frames over `gormes navivox serve --stdio`.
It will not use raw newline-delimited JSON.

Frame prelude:

```text
4 bytes  magic          ASCII "NVOX" (0x4e564f58)
4 bytes  version        uint32, network byte order
4 bytes  header_length  uint32, network byte order
N bytes  JSON header    UTF-8 object
M bytes  payload        optional binary bytes
```

Required JSON header fields:

- `type`: protocol event type such as `chat.submit` or `voice.audio`.
- `message_id`: unique frame id.
- `timestamp`: server/client timestamp in RFC3339 format.
- `payload_length`: exact byte count for the binary payload after the header.

Optional JSON header fields:

- `correlation_id`
- `turn_id`
- `agent_id`
- `content_type`
- `metadata`

Event body placement:

- The fixed prelude and header carry framing, routing, correlation, content
  type, and payload length.
- Textual event bodies use a UTF-8 JSON payload with
  `content_type: "application/json"`.
- Binary event bodies, such as `voice.audio`, use the payload bytes directly and
  put only safe metadata such as codec, duration, chunk index, and transcript
  status in the header metadata.
- Empty control events such as `ping` may use `payload_length: 0`.

Version policy:

- Protocol v1 is the only accepted prelude version for the first server slice.
- `hello` may advertise `supported_versions`, but it must itself be encoded
  with a supported prelude version.
- Unsupported prelude versions are stream-level errors. They do not get an
  in-band error frame because the receiver cannot trust the framing contract.
- Future compatible upgrades must add golden codec fixtures before changing the
  accepted version range.

Reader rule:

- Read the fixed prelude, then exactly `header_length` bytes, parse the JSON
  header, then read exactly `payload_length` bytes.
- Reject frames with unknown magic, unsupported version, invalid JSON,
  negative/oversized lengths, or payload bytes that do not match
  `payload_length`.

## 4. Voice Architecture

Navivox will use server-first TTS and hybrid STT.

TTS:

- Gormes generates agent speech through configured server-side providers.
- Server audio streams back as `voice.audio` frames.
- Flutter plays server audio through `just_audio` with a custom
  `StreamAudioSource`.
- Local `flutter_tts` is optional and limited to short confirmations such as
  "Connected" or "Agent switched"; it is not the primary Linux/desktop TTS path.

STT:

- Local STT handles wake word and short control command detection.
- Audio plus the device transcript can still be submitted to Gormes for
  server-side transcription and higher-accuracy language handling.
- Mobile uses platform STT where available. Linux uses a local/offline fallback
  only when installed and configured; text-only fallback is always valid.

## 5. Config Administration

Navivox config admin will be schema-driven and server-authoritative.

Flow:

```text
config.schema + redacted config.get
  -> local edits
  -> config.diff
  -> config.validate
  -> user confirmation
  -> config.apply
  -> config.reload or pending-restart result
```

Rules:

- The app never edits `config.toml` or `.env` directly.
- Secrets are write-only. `config.get` returns status/source/redacted evidence,
  never secret values.
- Secret fields render as status indicators plus set/rotate/delete/test
  actions, gated by pairing role and local unlock policy.
- Sensitive or disruptive changes require explicit confirmation with exact
  before/after non-secret values.

## 6. Pairing And Host Reachability

- The host starts pairing from `gormes navivox pair`; the Flutter app should
  scan or paste the emitted descriptor instead of guessing SSH settings.
- The command emits a QR code plus a plain `navivox://pair?...` fallback with
  SSH host, port, user, `gormes navivox serve --stdio`, protocol version,
  device name, pairing code, and expiry.
- Tailscale SSH is the recommended network path. Host discovery prefers an
  explicit host, then Tailscale IPv4, then LAN IPv4, then loopback.
- Host preparation is explicit. `gormes navivox setup-host --plan` explains the
  Tailscale/OpenSSH/sudo steps; `--apply` previews exact commands and performs
  them only after confirmation.
- Sudo passwords are prompt-only, masked, and never stored in config, logs,
  pairing URIs, or QR payloads.

## 7. Implementation Order

The first implementation slice should be server protocol before full app UI.

Order:

1. Add a builder-ready Gormes row for `gormes navivox serve --stdio` with frame
   codec, `hello`, `server.status`, ping/pong, error frames, and fake transport
   tests.
2. Add `gormes navivox pair` QR/text descriptor and host setup planning UX.
3. Scaffold the Flutter app shell and fake Navivox channel using the accepted
   Riverpod/GoRouter/Drift/Freezed stack.
4. Add Flyer Chat integration with text, streaming text, tool-call card, voice
   bubble, and agent-switch system/control renderers against fake events.
5. Add config schema/diff/validate/apply events and secret-safe admin forms.
6. Add voice capture, hybrid STT command handling, and server TTS playback.

This order avoids building a polished UI on top of an undefined wire contract.
