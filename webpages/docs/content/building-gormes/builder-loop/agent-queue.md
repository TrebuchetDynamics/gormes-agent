---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Navivox structured tool event cards Flutter UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: The sibling Navivox Flutter app consumes the Gormes backend structured tool-progress contract, upserts one durable ToolCallCard per tool_call_id for started/updated/finished states, renders redacted artifact rows, and never converts tool events into assistant prose.
- Trust class: operator, gateway, system
- Ready when: `Navivox structured tool event cards backend API` is complete and exposes started/updated/finished backend fixtures., Flutter already has a basic NavivoxToolCall model and ToolCall body renderer to extend.
- Not ready when: The UI writes backend progress as assistant text instead of structured ToolCallCard messages., The UI displays raw tool arguments, stdout, credentials, or full logs., The UI changes the Gormes backend contract or requires non-repo credentials to validate.
- Degraded mode: If a channel lacks structured tool progress, gateway falls back to existing bounded text progress; if Navivox receives malformed tool metadata, it renders redacted error evidence inside a tool card instead of assistant text.
- Fixture: `../navivox-app/test/core/channel/gateway_navivox_channel_test.dart; ../navivox-app/test/features/chat/`
- Write scope: `../navivox-app/lib/core/gateway/navivox_gateway_protocol.dart`, `../navivox-app/lib/core/channel/gateway_navivox_channel.dart`, `../navivox-app/lib/core/protocol/navivox_event.dart`, `../navivox-app/lib/features/chat/widgets/simple_chat_adapter.dart`, `../navivox-app/test/core/channel/gateway_navivox_channel_test.dart`, `../navivox-app/test/features/chat/`, `docs/content/building-gormes/architecture_plan/progress.json`, `../navivox-app/lib/features/chat/transcript_tool_call_presentation.dart`, `../navivox-app/test/features/chat/transcript_tool_call_presentation_test.dart`, `../navivox-app/test/features/chat/tool_artifacts_render_test.dart`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/core/channel/gateway_navivox_channel_test.dart test/features/chat'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter renders backend tool-progress lifecycle events as durable cards with redacted status/artifact evidence and no assistant-prose leakage.
- Acceptance: GatewayNavivoxChannel upserts a single ToolCall message per tool_call_id for started, updated, and finished backend events., ToolCallCard renders status, summary, approval state when present, and bounded artifact rows with id/kind/title/summary/ref., Malformed or oversized event metadata is truncated/redacted in UI fixtures and never becomes assistant prose., Existing chat fixtures remain green for normal assistant streaming and final messages.
- Source refs: docs/content/building-gormes/architecture_plan/progress.json:Navivox structured tool event cards backend API, ../navivox-app/lib/core/protocol/navivox_event.dart:NavivoxToolCall, ../navivox-app/lib/core/channel/gateway_navivox_channel.dart:_upsertToolCall, ../navivox-app/lib/features/chat/widgets/transcript_bubble.dart:_ToolCallBody, ../navivox-app/lib/features/chat/transcript_tool_call_presentation.dart:TranscriptToolCallPresentation
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. Navivox voice run records Flutter inspection UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Render the sibling Flutter Navivox inspection surface for backend run records after the Gormes run-record API lands. The app should fetch or receive redacted run records, show text and voice transcript evidence, STT/TTS metadata, tool timeline cards, attachment/artifact refs, terminal status, and provider usage/cost with explicit unknown states. This row is intentionally cross-root and must not be selected during repo-only Gormes iterations.
- Trust class: operator, gateway, system
- Ready when: `Navivox voice run records backend API` has landed and exposes a redacted JSON read model for text and voice turns., The sibling Navivox app already has the connect-and-talk chat fixture and can identify the active session/run after a turn ends., The UI keeps raw audio, provider secrets, and direct Gormes store access out of Flutter state.
- Not ready when: The backend run-record API is still missing or returns only mock data., The slice stores raw audio by default or hides retention status., The slice reports fake token cost when provider usage is absent., The slice edits Gormes backend persistence instead of consuming the backend contract from the app side.
- Degraded mode: If audio bytes, STT, TTS, or usage data are unavailable, the run record stores explicit unavailable evidence and preserves the text transcript/tool timeline instead of dropping the run or faking costs.
- Fixture: `../navivox-app/test/features/chat/navivox_run_records_test.dart`
- Write scope: `../navivox-app/lib/core/protocol/navivox_event.dart`, `../navivox-app/lib/core/channel/gateway_navivox_channel.dart`, `../navivox-app/lib/features/chat/`, `../navivox-app/test/core/channel/gateway_navivox_channel_test.dart`, `../navivox-app/test/features/chat/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/core/channel/gateway_navivox_channel_test.dart test/features/chat'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter can inspect backend-produced text and voice run records with transcript, tool timeline, STT/TTS metadata, artifacts, terminal status, and explicit usage/cost unknown states.
- Acceptance: After a text turn ends, the app can open a run detail or transcript panel showing redacted transcript, status, timestamps, session/run ids, and provider usage when available., After a voice turn ends, the app renders device transcript, optional server STT evidence, audio duration/codec metadata, TTS metadata, and explicit raw-audio retention status., Typed tool timeline entries render as tool cards or bounded timeline rows instead of assistant prose., Provider usage/cost fields render as `unknown` or unavailable when absent; the UI never fabricates zero cost.
- Source refs: internal/apiserver/runs.go:runRecord, internal/channels/navivox/channel.go:sessionState, ../navivox-app/lib/core/protocol/navivox_event.dart:NavivoxVoiceMessage, ../navivox-app/lib/core/channel/gateway_navivox_channel.dart, ../navivox-app/lib/features/chat/, https://docs.dograh.com/core-concepts/how-dograh-works
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
