# Navivox PRD

Status: decision baseline accepted; implementation rows pending
Scope: Flutter app plus Gormes `navivox` channel design
Platforms: Android, iOS, Linux, Windows

Decision baseline: `navivox-decision-record.md` is canonical for current chat
UI, Flutter stack, protocol framing, voice, config admin, and implementation
order decisions.

## 1. Product Summary

Navivox is a cross-platform Flutter app for operating one or more Gormes agent
servers over SSH. It combines a Telegram-like chat experience with a continuous
voice mode similar to the ChatGPT mobile app. Voice is not a separate product
mode: speech, text, tool progress, agent switching, and configuration changes all
belong to the same chat threads and agent sessions.

The app connects to remote machines with SSH keys only. It can generate and
import keys, import Termius export files from day one, pin host keys, connect to
multiple servers, and probe each server for Gormes. When Gormes is present, the
app starts a structured `navivox` channel over SSH. When a server is generic
SSH, the app still offers terminal and file-import/server-management behavior,
but Gormes-specific chat, voice, agent, config, and tool features require the
remote `gormes navivox serve --stdio` command.

## 2. Goals

- Provide a polished chat-first Gormes client for Android, iPhone, Linux, and
  Windows.
- Use SSH key authentication only. Password login is not allowed.
- Support multiple servers and multiple agents per server.
- Support Termius file import on the first usable version.
- Provide full text chat over a first-class Gormes `navivox` channel.
- Provide continuous voice mode with local wake/control commands and remote
  agent responses.
- Let each agent have its own voice profile and runtime language policy.
- Let Navivox create and edit agents, including custom workspace directories.
- Let Navivox administer Gormes configuration, including models, providers,
  Telegram bot tokens, channels, tools, gateway settings, secrets, and runtime
  behavior.
- Render Gormes tool calls as structured progress, approval, result, and
  artifact UI instead of raw logs.

## 3. Non-Goals For V1

- Password-based SSH login.
- Termius cloud sync or account login.
- Editing raw remote files directly from the app as the primary Gormes config
  path.
- Shell scraping as the main chat protocol.
- Git clone-from-URL agent creation. V1 selects existing remote directories.
- Running Gormes locally on the phone.
- Treating voice as a separate session history from chat.

## 4. Primary Personas

- Operator: runs one or more Gormes agents on remote Linux machines and wants a
  mobile-first chat and voice interface.
- Developer: maps one agent to one repository/workspace and switches agents
  quickly.
- Admin: configures providers, models, tokens, tools, channels, and security
  policy without editing TOML or dotenv files by hand.

## 5. System Architecture

### Client

The Flutter app owns local UX, local server inventory, local chat cache, key
import/generation, host key trust, device STT for commands, microphone capture,
audio playback, terminal UI, and local notifications.

### Transport

Navivox uses SSH as the only remote transport in V1. The app opens an SSH
session and starts a non-PTY command:

`gormes navivox serve --stdio`

The app and server then exchange framed structured protocol messages over
stdin/stdout. Terminal scraping is reserved only for the generic SSH terminal
tab.

### Gormes Server

Gormes owns agent execution, config mutation, secret persistence, provider/model
routing, tool execution, permissions, workspace validation, language policy,
voice generation, and gateway/channel behavior.

### Gormes Channel Files

Planned Gormes code shape:

- `cmd/gormes/navivox.go`: thin CLI entrypoint.
- `internal/channels/navivox/channel.go`: gateway channel adapter.
- `internal/channels/navivox/protocol.go`: framing, event types, compatibility.
- `internal/channels/navivox/voice.go`: voice event translation.
- `internal/channels/navivox/pairing.go`: device pairing and roles.
- `internal/channels/navivox/config_admin.go`: typed config administration.

The channel should follow the same gateway principles as Telegram, while adding
capabilities Telegram does not need: SSH pairing, binary audio frames, config
admin, agent admin, and structured tool-call rendering.

## 6. SSH And Server Management

### Authentication

- SSH password login is not supported.
- Imported password-only Termius identities are rejected with a clear reason.
- Encrypted SSH private keys are allowed; the passphrase is only for decrypting
  the local key and is not a remote password.
- Generated keys should default to Ed25519.
- Imported keys should support common OpenSSH private key formats used by
  Termius and standard SSH tools.
- Host key verification is mandatory. First connection displays the fingerprint;
  later changes require explicit re-trust.

### Local Key Storage

- Key metadata lives in the app database.
- Private key material lives in platform secure storage when practical.
- If a platform secure store cannot safely store large key blobs, use an
  encrypted local blob with the wrapping key in secure storage.
- Linux requires a working secret service such as libsecret/GNOME Keyring/KWallet.
  The app must surface a clear setup error when secure storage is unavailable.

### Server Inventory

Each server record includes:

- display name
- hostname
- port
- username
- selected key identity
- pinned host key fingerprint
- last connection status
- Gormes probe result
- preferred agent
- optional terminal profile
- imported Termius source metadata

### Termius Import

V1 supports file import of Termius exports. The import should map:

- hosts
- groups/folders
- identities
- SSH keys
- known hosts/fingerprints when present
- port-forward metadata when present
- labels/tags where present

Termius account/cloud sync is out of scope for V1. Import is repeatable and
deduplicates by host, port, username, key fingerprint, and stable Termius IDs
when available.

## 7. Navivox Protocol

### Framing

The protocol is framed, versioned, and binary-safe. It should not be raw newline
JSON for audio. Accepted frame shape:

- 4-byte magic: ASCII `NVOX` (`0x4e564f58`)
- 4-byte protocol version, unsigned integer, network byte order
- 4-byte JSON header length, unsigned integer, network byte order
- JSON header with event type, ids, timestamps, `payload_length`, and metadata
- optional binary payload for audio or files, exactly `payload_length` bytes
- request/response correlation id in the header when applicable

Every frame must be bounded by size limits. Large media should stream in chunks.
Frame readers must reject bad magic, unsupported versions, invalid JSON,
oversized lengths, and payloads whose byte count does not match
`payload_length`.

For V1, the header carries framing, routing, correlation, content type, payload
length, and safe metadata. Non-binary event bodies are encoded as a UTF-8 JSON
payload with `content_type: "application/json"`. Binary bodies use the payload
bytes directly and keep codec/chunk metadata in the header. Protocol v1 is the
only accepted prelude version for the first server slice; `hello` may advertise
future supported versions but must be sent in a supported frame version.

### Core Events

- `hello`: version negotiation, device id, supported features.
- `server.status`: Gormes version, config version, active channels.
- `chat.submit`: user text, voice transcript, or mixed text/audio turn.
- `chat.message`: assistant message or user echo.
- `chat.update`: streamed assistant update.
- `chat.final`: final assistant text for the turn.
- `chat.delete`: message removal when the gateway supports it.
- `typing.set`: assistant/agent typing state.
- `voice.submit`: audio payload and device transcript metadata.
- `voice.transcript`: server transcript updates.
- `voice.audio`: server-generated TTS audio.
- `agent.list`, `agent.get`, `agent.create`, `agent.update`,
  `agent.archive`, `agent.select`: agent administration.
- `tool.call.started`, `tool.call.progress`, `tool.call.completed`,
  `tool.call.failed`, `tool.call.cancelled`, `tool.call.blocked`:
  structured tool-call lifecycle.
- `tool.approval.requested`, `tool.approval.responded`: user approval flow.
- `tool.artifact.ready`: file, diff, screenshot, terminal output, or other
  inspectable artifact.
- `config.schema`, `config.get`, `config.diff`, `config.set`,
  `config.secret.set`, `config.secret.status`, `config.validate`,
  `config.apply`, `config.reload`, `config.rollback`: config administration.
- `ping`, `pong`, `error`: health and error handling.

### Gateway Mapping

`chat.submit` maps into `gateway.InboundEvent` with:

- `Platform`: `navivox`
- `ChatID`: Navivox conversation/device thread id
- `UserID`: paired device/user identity
- `MsgID` or equivalent: client message id
- agent override: selected Navivox agent for the session

Slash commands should reuse the existing gateway parsing path so commands such
as `/status`, `/new`, `/stop`, `/tts`, and `/busy` keep the same semantics.

## 8. Chat Experience

The first screen after setup is chat, not a landing page. Chat behaves like a
dense Telegram-style messenger:

- chronological bubbles
- server and agent context in the top bar
- message delivery/streaming states
- markdown rendering for assistant text
- inline tool-call cards
- inline voice messages
- text composer always visible
- mic button always available
- agent switcher reachable without leaving the thread

Final assistant text must not include raw tool progress. Tool details live in
structured cards and artifact sheets.

## 9. Voice Experience

### Unified With Chat

Voice input creates normal chat turns. A voice turn may include:

- audio payload
- device transcript
- confidence
- locale
- partial/final state
- command candidate
- wake word evidence

The server may accept the device transcript, re-transcribe the audio, or ask for
confirmation. Text and audio are both preserved as part of the same turn.

### Wake And Control Model

NAVI is the default assistant/wake name and is configurable.

- In an active conversation, the user can speak naturally without a wake word.
- Local control commands require NAVI by default, such as "NAVI switch agent
  mineru".
- When idle/backgrounded, NAVI is required before remote submission.
- Local-only commands can run from device STT without a remote round trip when
  confidence is high.

### Local STT

Device STT is used for:

- wake/control command detection
- short command testing
- fallback transcript when the server cannot transcribe

Because desktop STT support is uneven across platforms, local STT is not the
only transcription path. The app can always send audio plus the device
transcript to Gormes.

### Voices And Language

Gormes is the source of truth for agent voice profiles. The app may cache them
for offline display and low-latency UI, but server config wins.

Each agent has:

- voice provider
- voice id
- default locale/language
- speed
- pitch/style
- fallback voice
- runtime language policy

Agents can switch language at runtime. A language switch affects STT hinting,
server prompt/runtime policy, and TTS voice selection for later responses.

## 10. Agent Management

Navivox can list, create, update, archive, and select agents. V1 supports adding
an agent for an existing remote repository/workspace directory.

Agent fields:

- id
- display name
- workspace directory
- agent directory
- default flag
- model override
- skills
- tool allow list
- tool deny list
- sandbox mode/scope
- group-chat mention patterns
- voice profile
- language policy

Agent creation validates:

- directory exists
- directory is accessible to the SSH user
- optional repo marker exists when user expects one-agent-per-repo
- agent id is valid and unique
- no more than one default agent

Session agent selection must be a real gateway/session field, not hidden in
chat text or chat IDs. Recommended Gormes work: add a request-level selected
agent id to inbound events and route in priority order:

1. explicit Navivox session selection
2. configured bindings
3. configured default agent

Changing the selected agent for a conversation does not mutate global config
unless the user explicitly pins/saves it.

## 11. Full Gormes Config UI

Navivox must expose full Gormes configuration management through a typed admin
surface. "Full access" means full safe management, not secret read-back.

### Config Authority

Gormes remains the config authority. The app never edits `config.toml` or `.env`
directly. It calls the `navivox` config admin protocol, and Gormes performs:

- schema-aware reads
- validation
- diff generation
- atomic TOML writes
- dotenv secret writes
- SecretRef status checks
- runtime reload when supported
- rollback to the previous valid config when apply fails

### Existing Gormes Config Sections To Cover

The UI must cover the current top-level Gormes config surface:

- `hermes`: endpoint, provider, model, API key status/reference.
- `runtime`: tool iteration limits, terminal backend, TTS provider, compression,
  session reset policy.
- `gateway`: proxy URL and proxy key status.
- `terminal`: backend and current working directory.
- `display`: tool progress mode and per-platform display overrides.
- `tui`: theme and mouse tracking.
- `input`: maximum input bytes and lines.
- `telegram`: bot token status/reference, allowed chat/user ids, mention
  requirement, bot username, coalescing, discovery, memory and semantic options.
- `discord`: token status/reference and channel settings.
- `slack`: bot/app token status/reference and Socket Mode settings.
- `yuanbao`: channel-specific settings.
- `web`: backend and gateway usage.
- `browser`: CDP URL.
- `security`: website blocklist settings.
- `secrets`: SecretRef provider defaults and configured providers.
- `agents`: defaults and agent list.
- `bindings`: channel/account/peer to agent routing.
- `cron`: scheduled runtime settings.
- `skills`: skills root and selection limits.
- `delegation`: subagent execution controls.
- `goncho`: memory facade, peer cards, summaries, dreams, and context limits.

When Gormes adds future config sections, the server should expose them through
the schema so the app can render a safe generic editor before a custom screen is
built.

### Config Screens

Recommended UI groups:

- Overview: active server, config path, env path, Gormes version, config version,
  health, reload status.
- Provider and Models: provider, endpoint, default model, API key status,
  provider-specific auth state, test connection.
- Channels: Telegram, Discord, Slack, Yuanbao, Navivox, and future channels.
- Agents and Bindings: agent CRUD, defaults, workspace validation, routing.
- Tools and Display: tool policy, progress mode, artifact display.
- Voice: TTS provider, per-agent voices, language policy, wake name.
- Runtime: max tool iterations, terminal backend, input limits, session reset.
- Browser and Web: web backend, gateway toggle, CDP URL, browser tool readiness.
- Security: host trust, website blocklist, dangerous tool warnings.
- Secrets: secret status, SecretRef providers, rotate/set/delete secret actions.
- Cron, Skills, Delegation, Goncho: advanced operational settings.
- Advanced: redacted TOML view, schema-driven editor, diff and rollback.

### Secret Handling

Secrets are write-only after save.

Navivox may:

- show whether a secret is configured
- show source: env, SecretRef, file provider, runtime snapshot, missing
- set or replace a secret value
- delete a secret value when the server supports deletion
- configure a SecretRef target
- test whether a SecretRef resolves
- reload runtime secrets

Navivox must not:

- read a saved token value back from Gormes
- render secret-bearing errors
- write raw tokens into `config.toml`
- log tokens in app analytics, crash reports, or debug logs

Examples:

- Telegram bot token is set through `config.secret.set` for
  `telegram.bot_token`, stored by Gormes as `GORMES_TELEGRAM_TOKEN` or a
  SecretRef, and shown later only as `set [REDACTED]`.
- Provider API key is set through the same secret path for `hermes.api_key` or
  `api_key`.
- Gateway proxy key is managed as a secret, not as readable config text.

Opening the local secret editor should require biometric/PIN unlock on platforms
that support it. Linux falls back to the OS keyring/session security model plus
an app-level unlock when configured.

### Roles And Pairing

V1 should not assume every SSH-authenticated device can mutate production
config. Recommended role model:

- Owner: first paired device, can manage roles and all config.
- Admin: can change config, secrets, agents, tools, channels, and models.
- Operator: can chat, switch agents, approve tool calls, view most status.
- Viewer: read-only chat/status, no tool approval or config mutation.

Pairing should create a device identity bound to the SSH user and Gormes home.
Admin/config actions require an owner/admin role even if the SSH login
succeeds. A single-user install can accept the first paired device as owner to
avoid setup friction.

### Apply Model

Config writes use a staged flow:

1. App requests schema and current redacted values.
2. User edits fields.
3. App requests diff.
4. Server validates.
5. User confirms if the change is sensitive or disruptive.
6. Server writes atomically.
7. Server reloads the affected runtime surface when possible.
8. Server reports applied, pending restart, or rolled back.

Sensitive/disruptive changes include:

- provider API keys
- Telegram/Discord/Slack tokens
- gateway proxy key
- terminal backend
- tool allow/deny policies
- browser CDP URL
- workspace directories
- SecretRef providers
- security blocklist changes

## 12. Tool Calls

Gormes executes tools. Navivox renders and controls them.

Tool call UI requirements:

- show tool name, status, selected agent, workspace, and elapsed time
- show short progress summaries
- group logs/artifacts behind expandable details
- show risk level and mutation status
- show approval prompts when Gormes requires approval
- let the user approve, deny, or stop when permitted
- keep tool state attached to the chat turn that caused it
- avoid reading long logs aloud in voice mode unless asked

Tool event fields should include:

- turn id
- tool call id
- tool name
- display name
- icon hint
- preview
- status
- timestamps
- duration
- agent id
- workspace
- risk level
- mutating flag
- approval requirement
- result summary
- artifact refs
- redacted error

V1 can initially derive some progress from existing render frames, but the PRD
requires typed Gormes tool events as the durable contract.

## 13. UI Design

### Navigation

Primary app sections:

- Chats
- Servers
- Agents
- Config
- Keys
- Terminal
- Settings

On mobile, use bottom navigation for the main sections and sheets for detailed
operations. On desktop, use a left rail/sidebar and denser split views.

### First Run

First-run flow:

1. Import Termius file or add server manually.
2. Import or generate SSH key.
3. Connect and verify host fingerprint.
4. Probe for Gormes.
5. Start `gormes navivox serve --stdio` when available.
6. Pair the device and assign owner/admin role.
7. List agents.
8. Pick existing agent or create one from a remote workspace path.
9. Configure voice/language.
10. Open chat.

### Chat Screen

Required elements:

- server switcher
- active agent pill
- voice/listening state
- message stream
- tool-call cards
- text composer
- mic/continuous-voice control
- attachment/audio controls
- command suggestions for recognized local commands

### Config Screen

The Config tab is a work-focused admin console, not a marketing page. It should
use forms, segmented controls, toggles, tables, validation badges, and diff
drawers. Dangerous settings use confirmation sheets with exact before/after
values, while secret fields show only status and replacement controls.

### Terminal Screen

The Terminal tab provides a normal SSH terminal for generic server work and
debugging. It is not the Navivox chat protocol. The app must keep terminal
sessions visually separate from agent chat to avoid implying that shell output
is the assistant conversation.

## 14. Local App Data Model

Local persisted data:

- servers
- SSH identities and public key fingerprints
- host key pins
- imported Termius records
- device pairing records
- chat cache
- message/tool-call cache
- agent cache
- config schema cache
- voice profile cache
- local settings

Local caches are convenience state. Gormes remains the source of truth for
agents, runtime config, tool policy, and voice profiles.

## 15. Flutter Library Plan

Current recommended package set:

- Flutter SDK: Android, iOS, Linux, and Windows app framework.
- `dartssh2`: SSH client, private key auth, SSH exec, shell, and SFTP.
- `flutter_secure_storage`: platform secure storage for keys and secrets.
- `local_auth`: biometric/PIN gate where supported.
- `file_picker`: Termius export and SSH key file import.
- `record`: microphone capture and PCM/audio streaming.
- `speech_to_text`: local STT for mobile/macOS/web and limited desktop support;
  use as command/test STT, not the only transcription path.
- `just_audio`: playback for server-generated TTS/audio responses.
- `flutter_tts`: optional local command confirmations where platform support is
  available; not the primary Linux TTS path.
- `riverpod`: connection/session/config state management.
- `go_router`: app navigation and deep links.
- `drift`: local SQLite persistence and reactive queries.
- `sqlite3_flutter_libs`: SQLite runtime support for Flutter targets.
- `xterm.dart`: terminal screen widget.
- `cryptography`: Ed25519 generation and crypto primitives for key handling.
- `permission_handler`: microphone, notification, and file permission prompts
  where platform policy requires them.
- `freezed`, `json_serializable`, `build_runner`: protocol/data models.
- `path_provider`, `path`, `uuid`, `intl`: local paths, identifiers, formatting.

Implementation must verify current package platform support before coding.
Known caveats:

- Linux secure storage depends on system secret service availability.
- Local STT support is uneven on Linux/Windows, so server STT remains required.
- Flutter TTS is not the primary cross-platform TTS solution for this product.
- SSH key generation needs OpenSSH-compatible serialization tested against real
  SSH servers and `ssh-keygen -y`.

## 16. Security Requirements

- SSH passwords are never accepted.
- Host key pinning is mandatory.
- Secret values are never returned after save.
- Secret-bearing errors are redacted on both server and client.
- Config changes are staged and validated before apply.
- Dangerous config changes require confirmation.
- Tool approvals are explicit and auditable.
- Local key material is protected by secure storage or encrypted blobs.
- App logs redact tokens, private keys, passphrases, transcripts marked private,
  and tool outputs marked sensitive.
- Pairing roles gate config mutation.

## 17. Error Handling

Important failure states:

- SSH key rejected.
- Encrypted key passphrase wrong.
- Host key changed.
- Secure storage unavailable.
- Termius export partially unsupported.
- Gormes not installed.
- `gormes navivox serve --stdio` missing or incompatible.
- Protocol version mismatch.
- Gormes config validation failed.
- SecretRef missing or unsupported.
- Runtime reload failed but config write succeeded.
- Tool approval timed out.
- Audio capture permission denied.
- Server TTS/STT unavailable.

The app should present recovery actions, not stack traces. For example:

- retry with another key
- re-trust host fingerprint
- open terminal
- install/update Gormes
- set missing token
- rollback config
- send text instead of voice

## 18. Testing And Validation Plan

Flutter app tests:

- unit tests for protocol codecs
- unit tests for Termius import mapping and rejection rules
- unit tests for SSH identity metadata and host key trust decisions
- unit tests for local command parsing
- widget tests for chat, config forms, secret editor, and tool-call cards
- integration tests with a fake Navivox server
- desktop smoke tests for Linux and Windows
- mobile smoke tests for Android and iOS

Gormes tests:

- `navivox` command starts stdio server without PTY assumptions
- gateway inbound mapping preserves text command semantics
- selected agent id routes correctly
- typed tool events are emitted and redacted
- config schema exposes all supported sections
- config set validates and writes atomically
- secret set writes dotenv/SecretRef paths without leaking values
- reload/rollback behavior is covered
- pairing roles gate admin operations

End-to-end tests:

- generated key connects to test SSH server
- imported key connects to test SSH server
- host key change is blocked
- Termius export imports usable server/key records
- text chat round trip
- voice turn with audio plus transcript
- local command "NAVI switch agent mineru"
- tool approval and cancellation
- Telegram bot token set from Navivox and not readable afterward
- provider/model change applies or reports restart requirement

## 19. Delivery Phases

### Phase 0: Planning And Interface Rows

- Treat the decision baseline as accepted.
- Create builder-ready Gormes progress rows for `navivox` channel, config admin,
  agent selection, tool events, and secret-safe config mutation.
- Create Flutter implementation plan after the Gormes protocol/config rows are
  defined.

### Phase 1: SSH Shell

- Flutter app shell.
- Server inventory.
- Key import/generation.
- Host key pinning.
- Termius file import.
- Generic terminal.

### Phase 2: Navivox Text Channel

- `gormes navivox serve --stdio`.
- Protocol handshake.
- Text chat submit/update/final events.
- Gateway mapping.
- Chat UI cache.

### Phase 3: Agents And Config

- Agent list/select/create/update/archive.
- Workspace validation.
- Full Config tab with schema, redacted get, diff, validate, apply.
- Secret-safe token/API key management.
- Pairing roles.

### Phase 4: Voice

- Mic capture.
- Device STT for commands and testing.
- Audio plus transcript submission.
- Server TTS playback.
- Per-agent voices.
- Runtime language switching.

### Phase 5: Tools And Artifacts

- Typed tool-call events.
- Tool cards.
- Approval/deny/stop controls.
- Artifact viewers.

### Phase 6: Hardening

- Offline cache behavior.
- Reconnect and resume.
- More Termius import coverage.
- Platform-specific installers/packages.
- Accessibility and internationalization pass.

## 20. Implementation Defaults

These defaults close the remaining planning gaps unless the product owner
changes them before implementation:

- First paired device is Owner. Additional paired devices default to Operator
  until promoted by Owner/Admin.
- Linux uses the OS secret service for key storage. The secret editor also
  supports an app PIN when biometric/PIN APIs are absent or unavailable.
- V1 voice uses device transcript plus audio submission by default. Server STT
  is used when configured. Server-generated TTS is preferred for agent voices;
  local TTS is limited to short command confirmations.
- Advanced raw TOML editing is not V1. V1 gets schema-driven editing, redacted
  TOML view, diff, validation, apply, reload, and rollback.
- SFTP file transfer UI beyond Termius/key import is not V1.
- Config changes made during an active turn are staged. Non-disruptive changes
  may apply immediately; model/provider/tool/runtime/channel changes apply to
  the next turn or after explicit reload/restart confirmation.

## 21. V1 Acceptance Criteria

V1 is acceptable when:

- A fresh app can import a Termius file, connect to a Gormes server with an SSH
  key, verify the host, pair the device, select an agent, and send text chat.
- Password SSH login is impossible through the UI and protocol.
- Voice mode can capture audio, produce/test local command STT, submit audio and
  text transcript to Gormes, and play agent audio responses.
- "NAVI switch agent mineru" can switch the active agent when that agent exists.
- Different agents can use different voices.
- Agents can switch language at runtime.
- Navivox can create/edit an agent with a custom existing workspace directory.
- Config UI can change provider/model settings and set/rotate a Telegram bot
  token without reading the token back.
- Tool calls appear as structured UI with approval and result states.
- Secrets, SSH keys, and token-shaped values do not leak into app logs, Gormes
  logs, protocol errors, or visible config output.
