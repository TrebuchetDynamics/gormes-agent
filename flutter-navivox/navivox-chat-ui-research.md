# Navivox Chat UI Research

Status: planning draft
Updated: 2026-05-16
Source: current Navivox product direction and prior open-source UI survey

## 1. Decision Summary

Navivox should keep the current simple chat adapter until the connect-and-talk
loop is proven. After that, the production chat surface should become
Telegram-inspired: dense chat list, fast message scan, grouped bubbles with
tails, compact timestamps, status ticks, bottom-sheet actions, voice transcript
bubbles, and first-class tool cards.

The Flutter app talks to Gormes, not directly to model providers. Any chat UI
package is only a rendering layer over `GatewayNavivoxChannel` state.

## 2. Telegram-Inspired Reference Plan

### 2.1 Verified References

Use these as current, source-backed references:

- `v_chat_bubbles` on pub.dev: strong candidate for bubble rendering because it
  supports a Telegram style preset, custom bubble types, selection mode,
  callbacks, text formatting, voice bubbles, and all Flutter target platforms.
- Flutter Material 3 docs: use current `NavigationBar`/rail patterns instead
  of legacy bottom navigation when adopting Material 3.
- Flutter `DraggableScrollableSheet`: use for Telegram-like action panels,
  server/agent switchers, transcript review, and tool detail sheets.
- `tdlib/td`: study only for Telegram client lifecycle and update ordering.
  Navivox must not add TDLib, MTProto, or Telegram network dependencies.
- `babakcode/flutter_chat`: lightweight visual reference for a Telegram-like
  Flutter chat app, useful for layout study but not a production architecture
  donor.

### 2.2 User-Supplied References To Verify Before Use

The operator also named Telware Cross-Platform, telega2,
`telegram_ios_ui_kit`, and `teleflutter`. Treat these as research leads until a
builder can verify source URL, license, maintenance state, platform support, and
API shape. Do not add them as dependencies or cite their behavior in product
contracts without that evidence.

### 2.3 Product Translation

Telegram pattern | Navivox translation
---|---
Chat list with avatars, last message, unread/status/time | Gateway/session list with agent icon, last user/assistant turn, unread tool/voice state, health badge
Message bubbles | User/assistant/system bubbles backed by `GatewayNavivoxChannel`
Read ticks | Local send/queued/streaming/done/error state, not server read receipts
Voice message | Device transcript bubble first; audio playback only after voice run records
Attachment/action tray | Draggable sheet for tools, voice, config, agent seed, and future files
Pinned banner | Gateway status and active agent warning
Context menu | Copy, retry, inspect tool, reveal redacted fields when authorized
Search | Local session/message search once persistence lands

## 3. Adopt

### 3.1 Bubble Renderer

Adopt `v_chat_bubbles` for the bubble layer if a builder proves the package can
wrap existing Navivox message state without taking over routing, persistence, or
backend behavior. The first use should be narrow:

- `VBubbleScope(style: VBubbleStyle.telegram)` at the chat screen boundary.
- `VTextBubble` for user and assistant text.
- `VVoiceBubble` only after voice run records define audio/playback state.
- `VCustomBubble` for `ToolCallCard`, so tools remain structured UI objects.
- Performance config for long transcripts.

The package must allow:

- External ownership of message state.
- Custom message types.
- Incremental updates for streaming text.
- Stable keys for message replacement.
- Accessible message actions.
- Desktop/web rendering without mobile-only assumptions.

Fallback: if package behavior, license, accessibility, performance, or theming
is unsuitable, keep the current simple adapter and implement local Telegram-like
widgets directly.

### 3.2 Text Streaming Renderer

Use a streaming text renderer for `assistant_delta` events only after the
channel tests prove one assistant message is updated per request. The renderer
should not create a new bubble for each delta.

## 4. Inspire From

### 4.1 Telegram-Style Apps

Useful patterns:

- A chat list is an operational dashboard, not just navigation.
- The chat screen should keep the composer always reachable.
- Presence/status belongs in small badges, not large banners unless degraded.
- Voice, attachments, search, and settings should live in sheets/drawers rather
  than taking over the main transcript.
- Media-heavy affordances are secondary for Navivox; tool and voice affordances
  are primary.

### 4.2 AI Chat Interfaces

Useful patterns:

- Event streams drive UI state.
- Tool calls are separate renderer types.
- Artifacts open in dedicated viewers.
- Approval controls are explicit and stateful.
- Errors include recovery actions.

### 4.3 Admin Interfaces

Useful patterns:

- Schema-driven forms.
- Field-level validation.
- Redacted secret status.
- Diff preview before apply.
- Confirmation for risky changes.

## 5. Skip Or Defer

Do not adopt packages that:

- Own model/provider orchestration in the Flutter app.
- Require a hosted chat backend.
- Require Firebase, MTProto, TDLib, or Telegram login for Navivox chat.
- Make tool calls plain transcript text.
- Force a single mobile-only layout.
- Add broad persistence before the product has retention rules.
- Require telephony concepts before the first local agent turn works.

## 6. Message Types

### 6.1 Text Message

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

### 6.2 ToolCallCard

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

### 6.3 Voice Message Bubble

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

### 6.4 System Message

Use for:

- Connected/disconnected status.
- Agent switch confirmation.
- Config apply result.
- Voice mode state.
- Safe errors.

System messages should be short and actionable.

## 7. Composer

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

## 8. Tool Event Mapping

| Gateway Event | UI Result |
|---------------|-----------|
| `tool_call_started` | Create or update a running `ToolCallCard`. |
| `tool_call_finished` | Mark card completed/failed and attach safe summary. |
| `error` with tool context | Mark card failed when a tool id is present. |
| Future approval request | Add approve/deny controls to the card. |

## 9. Voice Event Mapping

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

## 10. Agent UI Patterns

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

## 11. Config UI Patterns

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

## 12. Layout Patterns

Mobile:

- Material 3 `NavigationBar` with Chats, Agents, Config, Servers.
- Full-width chat transcript.
- `DraggableScrollableSheet` for agent/server/action switchers.
- Voice transcript panel from bottom.
- Tool card details as bottom sheet.

Desktop:

- Left rail.
- Persistent top bar.
- Status bar.
- Optional split detail panel for tool/config details.
- Keyboard-first composer.

## 13. Accessibility Requirements

- Icon buttons have text labels or tooltips.
- Status is not color-only.
- Tool cards are keyboard expandable.
- Voice transcripts are editable before send.
- Secret fields announce redacted state.
- Error banners include a primary recovery action.

## 14. Acceptance For Replacing The Simple Adapter

- Existing setup-to-chat tests still pass.
- Streaming text updates one assistant bubble.
- `ToolCallCard` has widget tests for running, completed, failed, and redacted
  states.
- Voice bubble has text-only and transcript-confirmation tests.
- Chat list previews show avatar/agent, last message, time, and status.
- Agent/server/tool sheets use `DraggableScrollableSheet` or an equivalent
  tested sheet interaction.
- Mobile and desktop layouts have snapshot or widget coverage.
- The first turn still works without telephony setup.
