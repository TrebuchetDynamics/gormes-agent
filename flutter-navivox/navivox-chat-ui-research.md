# Navivox Chat UI Research

Status: planning draft
Updated: 2026-05-16
Source: current Navivox product direction and prior open-source UI survey

## 1. Decision Summary

Navivox should keep the current simple chat adapter until the connect-and-talk
loop is proven. The planned production chat foundation remains
`flyerhq/flutter_chat_ui` because it is backend-agnostic and supports custom
message builders, which Navivox needs for tool cards and voice bubbles.

The Flutter app talks to Gormes, not directly to model providers. Any chat UI
package is only a rendering layer over `GatewayNavivoxChannel` state.

## 2. Adopt

### 2.1 Chat Framework

Adopt a full chat framework when these requirements are all active:

- Streaming assistant text.
- Custom tool call message rendering.
- Voice message bubbles.
- System/control messages.
- Mobile and desktop layout coverage.

The package must allow:

- External ownership of message state.
- Custom message types.
- Incremental updates for streaming text.
- Stable keys for message replacement.
- Accessible message actions.

### 2.2 Text Streaming Renderer

Use a streaming text renderer for `assistant_delta` events only after the
channel tests prove one assistant message is updated per request. The renderer
should not create a new bubble for each delta.

## 3. Inspire From

### 3.1 Chat Bubble Packages

Useful patterns:

- Group consecutive messages from the same author.
- Keep tool cards as separate blocks.
- Support compact timestamps.
- Support swipe or long-press actions where platform-appropriate.
- Keep voice messages visually distinct from plain text.

### 3.2 AI Chat Interfaces

Useful patterns:

- Event streams drive UI state.
- Tool calls are separate renderer types.
- Artifacts open in dedicated viewers.
- Approval controls are explicit and stateful.
- Errors include recovery actions.

### 3.3 Admin Interfaces

Useful patterns:

- Schema-driven forms.
- Field-level validation.
- Redacted secret status.
- Diff preview before apply.
- Confirmation for risky changes.

## 4. Skip Or Defer

Do not adopt packages that:

- Own model/provider orchestration in the Flutter app.
- Require a hosted chat backend.
- Make tool calls plain transcript text.
- Force a single mobile-only layout.
- Add broad persistence before the product has retention rules.
- Require telephony concepts before the first local agent turn works.

## 5. Message Types

### 5.1 Text Message

Fields:

- `id`
- `session_id`
- `request_id`
- `author`
- `text`
- `is_final`
- `created_at`
- `updated_at`

Rendering:

- User messages align to the trailing side.
- Assistant messages align to the leading side.
- Streaming assistant text updates in place.
- Markdown is allowed after sanitization.

### 5.2 ToolCallCard

Fields:

- `tool_call_id`
- `tool_name`
- `status`
- `summary`
- `input_preview`
- `output_preview`
- `artifacts`
- `requires_approval`
- `redaction_level`

Rendering:

```text
+-- execute_command ----------------------------+
| Status: running                                |
| Summary: checking system status                |
|                                                |
| [Inputs] [Output] [Artifacts]                  |
+------------------------------------------------+
```

Rules:

- Tool cards always start a new block.
- Sensitive fields are redacted by default.
- Approval buttons render only when the event contract supports them.
- Raw JSON is behind a debug action, never the default UI.

### 5.3 Voice Message Bubble

Fields:

- `voice_run_id`
- `session_id`
- `transcript`
- `transcript_source`
- `confidence`
- `duration_ms`
- `capture_status`
- `playback_status`

Rendering:

- Shows transcript first.
- Shows waveform/playback when audio exists.
- Shows capture/transcription error as recoverable state.
- Allows transcript edit before send for device-captured turns.

### 5.4 System Message

Use for:

- Connected/disconnected status.
- Agent switch confirmation.
- Config apply result.
- Voice mode state.
- Safe errors.

System messages should be short and actionable.

## 6. Composer

Default composer:

```text
+------------------------------------------------+
| [+] Type a message...                  [mic] > |
+------------------------------------------------+
```

States:

- Default: text field, attachment/future action, mic, send.
- Recording: transcript preview, cancel, send transcript.
- Connecting: disabled send with reconnect status.
- Unauthorized: token action.
- Offline: retry action.

The composer must keep text fallback available even when voice capture fails.

## 7. Tool Event Mapping

| Gateway Event | UI Result |
|---------------|-----------|
| `tool_call_started` | Create or update a running `ToolCallCard`. |
| `tool_call_finished` | Mark card completed/failed and attach safe summary. |
| `error` with tool context | Mark card failed when a tool id is present. |
| Future approval request | Add approve/deny controls to the card. |

## 8. Voice Event Mapping

Current gateway behavior can send a device transcript as text. Voice UI should
therefore start as a transcript capture and confirmation flow.

Future voice events should map to:

- Voice run created.
- Capture started/stopped.
- Transcript partial/final.
- Server STT complete.
- TTS audio ready.
- Playback started/stopped.
- Voice error.

## 9. Agent UI Patterns

The seed flow should feel like creating a draft, not filling a large form first.

```text
Seed: [ screen inbound leads ]
      [Generate Draft]

Draft:
- Name
- Goal
- Instructions
- Tools
- Voice
- Safety
```

The operator can edit every generated section before apply.

## 10. Config UI Patterns

Config forms are generated from server schema.

Required components:

- Section list.
- Typed field renderer.
- Secret status indicator.
- Diff viewer.
- Validation result panel.
- Confirmation sheet.
- Apply result banner.

Risk states:

- Local exposure: normal.
- VPN exposure: show interface evidence.
- Public exposure: require explicit confirmation.
- Provider or model change: show reconnect/restart impact.
- Secret change: write-only confirmation.

## 11. Layout Patterns

Mobile:

- Bottom navigation.
- Full-width chat.
- Sheets for agent/server switchers.
- Voice panel from bottom.
- Tool card details as bottom sheet.

Desktop:

- Left rail.
- Persistent top bar.
- Status bar.
- Optional split detail panel for tool/config details.
- Keyboard-first composer.

## 12. Accessibility Requirements

- Icon buttons have text labels or tooltips.
- Status is not color-only.
- Tool cards are keyboard expandable.
- Voice transcripts are editable before send.
- Secret fields announce redacted state.
- Error banners include a primary recovery action.

## 13. Acceptance For Replacing The Simple Adapter

- Existing setup-to-chat tests still pass.
- Streaming text updates one assistant bubble.
- `ToolCallCard` has widget tests for running, completed, failed, and redacted
  states.
- Voice bubble has text-only and transcript-confirmation tests.
- Mobile and desktop layouts have snapshot or widget coverage.
- The first turn still works without telephony setup.
