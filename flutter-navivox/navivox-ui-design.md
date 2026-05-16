# Navivox UI Design Guide

Status: planning draft
Updated: 2026-05-16
Source: current HTTP/WebSocket gateway plan, PRD, and app shell

## 1. Design Principles

1. **Connect and talk first**: the first useful path is base URL, token, health
   check, stream, chat.
2. **Work-focused, not marketing**: no landing page before the operator can talk
   to an agent.
3. **Tool calls are first-class UI**: use structured cards, not transcript log
   dumps.
4. **Secrets are invisible by design**: write-only fields, redacted status, no
   read-back.
5. **Server-authoritative config**: the app edits drafts; Gormes validates and
   applies.
6. **Voice is an input mode, not a setup blocker**: text turn fallback always
   works.
7. **Trust boundaries stay visible**: connected host, auth mode, exposure mode,
   and token-required state are shown without leaking secrets.
8. **Dense, adaptive layout**: mobile uses bottom navigation; desktop uses a
   left rail and status bar.

## 2. Primary Screens

### 2.1 Setup Screen

```text
+------------------------------------------------+
| Navivox                                        |
| Connect to Gormes                              |
+------------------------------------------------+
| Base URL                                       |
| [ http://127.0.0.1:8765                  ]     |
|                                                |
| Token                                          |
| [ ************************************** ]     |
|                                                |
| Health:  not checked                           |
| Status:  not checked                           |
| Stream:  not connected                         |
|                                                |
| [Connect]                                      |
+------------------------------------------------+
| Host command                                   |
| gormes navivox connect-info                    |
+------------------------------------------------+
```

States:

- Empty: show the host command and base URL field.
- Health failed: show gateway unavailable and retry.
- Unauthorized: show token/auth action.
- Exposure blocked: show server-side exposure guidance.
- Connected: navigate to chat.

### 2.2 Chat Screen

```text
+------------------------------------------------+
| Gormes Gateway             default agent   OK  |
+------------------------------------------------+
| Today                                          |
|                                                |
|                         Check server status    |
|                                                |
| Agent                                          |
| Checking now...                                |
|                                                |
| +-- execute_command ------------------------+  |
| | Status: running                            |  |
| | Command: uptime                            |  |
| | [Expand]                                   |  |
| +---------------------------------------------+ |
|                                                |
| Agent                                          |
| Server is healthy.                             |
+------------------------------------------------+
| [+] Type a message...                  [mic] > |
+------------------------------------------------+
```

Key UI elements:

- Server label opens server switcher.
- Agent pill opens agent switcher.
- Status chip shows connected, reconnecting, offline, unauthorized, or blocked.
- User messages are right-aligned.
- Assistant messages are left-aligned.
- Streaming assistant text updates in place.
- `ToolCallCard` appears inline with status and expandable details.
- Voice button can submit a transcript through the current turn path.

### 2.3 Voice Active

```text
+------------------------------------------------+
| Gormes Gateway             default agent   OK  |
+------------------------------------------------+
| Voice turn                                      |
|                                                |
| Listening...                              [x]  |
| [ transcript appears here as the device hears ] |
|                                                |
| Confidence: high                               |
|                                                |
| [Send Transcript] [Cancel]                     |
+------------------------------------------------+
```

Voice rules:

- A transcript can be sent as text immediately.
- Audio upload and playback state appear only after voice run records exist.
- Text fallback is always available.

### 2.4 Servers Screen

```text
+------------------------------------------------+
| Servers                                  [+]   |
+------------------------------------------------+
| local-gormes                                    |
| http://127.0.0.1:8765                           |
| Exposure: local    Auth: token required         |
| Health: OK         Stream: connected            |
| [Use] [Details]                                |
|                                                |
| tailnet-host                                    |
| http://100.64.1.2:8765                          |
| Exposure: tailscale   Auth: tailnet identity    |
| Health: offline                                |
| [Retry] [Details]                              |
+------------------------------------------------+
```

Server detail shows:

- Base URL.
- Health URL.
- Auth mode.
- Exposure mode.
- Token-required state.
- Last successful status.
- Last stream error.
- Redacted local credential status.

### 2.5 Agents Screen

```text
+------------------------------------------------+
| Agents                                  [+]    |
+------------------------------------------------+
| Create from seed                              |
| [ screen inbound leads                   ]     |
| [Generate Draft]                               |
|                                                |
| default agent                                  |
| Voice: system default                          |
| Tools: safe default set                        |
| [Edit] [Use]                                   |
|                                                |
| support triage                                 |
| Voice: warm assistant                          |
| Tools: ticket lookup, summarize                |
| [Edit] [Use]                                   |
+------------------------------------------------+
```

Agent creation starts with a short phrase and produces an editable draft:

- Name and description.
- Prompt/instructions.
- Tool set and permissions.
- Voice defaults.
- STT/TTS provider/profile preferences.
- Safety and escalation notes.

### 2.6 Agent Editor

```text
+------------------------------------------------+
| Edit Agent                              [Save] |
+------------------------------------------------+
| Name                                           |
| [ support triage                         ]     |
|                                                |
| Instructions                                   |
| [ summarize the issue, ask one clarifier... ]  |
|                                                |
| Tools                                          |
| [ ] shell                                      |
| [x] ticket lookup                              |
| [x] summarize                                  |
|                                                |
| Voice Profile                                  |
| Provider: [Server default v]                   |
| Voice:    [Warm assistant v]                   |
| STT:      [Server default v]                   |
|                                                |
| Safety                                         |
| [x] Confirm before irreversible actions        |
| [x] Redact sensitive tool output               |
+------------------------------------------------+
```

### 2.7 Config Overview

```text
+------------------------------------------------+
| Config                                  Admin  |
+------------------------------------------------+
| Source: server schema                           |
| Secrets: redacted                               |
| Pending restart: no                             |
|                                                |
| Provider and Models                     >       |
| Voice Providers                         >       |
| Navivox Gateway                         >       |
| Tools and Approvals                     >       |
| Agents                                  >       |
| Security                                >       |
|                                                |
| [Reload Schema] [Review Pending Changes]       |
+------------------------------------------------+
```

### 2.8 Config Section

```text
+------------------------------------------------+
| Navivox Gateway                         [Back] |
+------------------------------------------------+
| Enabled                                  [x]   |
| Bind Host                                127.0.0.1 |
| Port                                     8765  |
| Exposure Mode                            local |
| Auth Mode                                static_token |
| Token                                    configured, redacted |
|                                                |
| [Set Token] [Test Connection]                  |
|                                                |
| Diff                                           |
| exposure_mode: local -> tailscale              |
|                                                |
| [Validate] [Apply]                             |
+------------------------------------------------+
```

Config rules:

- Secret values never render after entry.
- Diff shows non-secret before/after values.
- Server validation errors map to fields.
- Public exposure requires an explicit confirmation dialog.

### 2.9 Tool Call Card

```text
+-- execute_command ----------------------------+
| Status: completed                              |
| Duration: 0.3s                                 |
| Summary: uptime returned load averages         |
|                                                |
| [Inputs] [Output] [Artifacts]                  |
+------------------------------------------------+
```

Tool card states:

- queued
- running
- needs approval
- approved
- denied
- completed
- failed

Sensitive inputs and outputs are redacted by default with an explicit reveal
gate when the server allows it.

### 2.10 Settings Screen

```text
+------------------------------------------------+
| Settings                                       |
+------------------------------------------------+
| Appearance                                     |
| Theme: [Dark v]                                |
| Density: [Compact v]                           |
|                                                |
| Voice Defaults                                 |
| Wake Word: [NAVI]                              |
| Text fallback: enabled                         |
|                                                |
| Security                                       |
| App Lock: [on]                                 |
| Lock Timeout: [5 minutes v]                    |
|                                                |
| Data                                           |
| [Clear Local Cache]                            |
| [Forget This Gateway]                          |
|                                                |
| About                                          |
| Navivox 0.1.0                                  |
+------------------------------------------------+
```

## 3. Component Library

### 3.1 Shared Components

```dart
// Navigation
AppScaffold
ConnectionStatusBar
ServerSwitcher
AgentPill
ErrorRecoverySheet

// Setup
ConnectInfoForm
HealthProbeStatus
TokenField
ExposureModeNotice

// Chat
MessageBubble
ToolCallCard
VoiceMessageBubble
TypingIndicator
DateSeparator
MessageComposer
VoiceControlBar

// Config
ConfigSectionCard
ConfigFormField
SecretStatusIndicator
ConfigDiffViewer
ApplyConfirmSheet
LocalUnlockGate

// Agents
AgentCard
AgentSeedInput
AgentDraftEditor
VoiceProfilePicker

// Servers
ServerCard
GatewayStatusPanel
ReachabilityBadge
```

## 4. Status And Error States

| State | UI Treatment | Primary Action |
|-------|--------------|----------------|
| Gateway offline | Red status chip, setup error | Retry health check |
| Unauthorized | Amber status chip | Enter token |
| Exposure blocked | Red notice | Fix server config |
| Stream disconnected | Reconnecting banner | Retry or edit connection |
| Inbox full | Assistant/system error | Try again |
| Tool failed | Failed tool card | Expand details |
| Secret denied | Field error | Request admin role or unlock |

All error copy should point to a next action and avoid raw provider errors or
secret-shaped values.

## 5. Visual Density

Mobile:

- Bottom navigation.
- Single-column chat.
- Sheets for server and agent switching.
- Voice controls as bottom panel.

Desktop:

- Left rail.
- Persistent top/status bars.
- Optional split pane for config diffs and tool details.
- Keyboard-friendly command input.

## 6. Accessibility

- All icon buttons have labels/tooltips.
- Status chips have text, not color alone.
- Tool cards are keyboard expandable.
- Voice transcript can be edited before send.
- Secret fields describe state without exposing values.

## 7. Product Boundaries

Do now:

- Connect to gateway.
- Talk to agent.
- Show structured tool activity.
- Seed editable agents.
- Edit config through server APIs.
- Record voice run metadata before streaming audio complexity.

Defer:

- Phone numbers.
- Outbound campaigns.
- Retry schedulers.
- Call routing.
- Human handoff.
- Generic server administration surfaces.
