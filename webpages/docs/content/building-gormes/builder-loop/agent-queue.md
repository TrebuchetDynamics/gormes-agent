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
## 1. Gormes-owned session tree navigator over lineage and labels

- Phase: 4 / 4.B
- Owner: `tui`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Add a native `/tree` session navigator that projects Gormes' existing session lineage, fork, compression, and title metadata into an in-place tree view with search/filter modes and operator labels. Selecting a prior user turn should restore that prompt for editing when safe; selecting non-user entries should switch the visible leaf or report why the stored transcript cannot be replayed. The implementation must use Gormes session stores and lineage tables, not Pi JSONL files.
- Trust class: operator, system
- Ready when: The builder reuses existing session directory, lineage, fork, resume, and TUI panel seams with fake stores in tests., The first slice may omit LLM-generated branch summaries if it records typed not-yet-supported evidence and keeps labels/navigation functional.
- Not ready when: The implementation introduces a second session file format, writes Pi JSONL sessions, or bypasses internal/session and store abstractions., The navigator silently mutates live session state while the kernel is active or loses fork/compression lineage evidence.
- Degraded mode: -
- Fixture: `internal/tui/tree_selector_test.go`
- Write scope: `internal/session/`, `internal/tui/tree_selector.go`, `internal/tui/slash_tree.go`, `internal/tui/slash_dispatch.go`, `cmd/gormes/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/session ./internal/tui -run 'Test.*(Tree\|Label\|Lineage\|Resume\|Branch)' -count=1`, `go test ./cmd/gormes -run 'Test.*Session.*(Tree\|Resume\|Branch)' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report seeded tree fixture output, label persistence evidence, replay/degraded cases, and progress validation.
- Acceptance: A seeded lineage fixture renders a tree with forks, compressed/continued sessions, titles, timestamps, and active leaf marker., Filter modes can show default, no-tools, user-only, labeled-only, and all-equivalent projections over Gormes transcript metadata where data exists., Labels/bookmarks can be set and cleared through a session metadata seam and appear in the tree selector., Selecting a prior user turn restores editable text or returns typed replay-unavailable evidence without corrupting the active session.
- Source refs: pi@fc8a155 packages/coding-agent/docs/sessions.md:/tree navigation, pi@fc8a155 packages/coding-agent/docs/session-format.md:labels, branch summaries, tree entries, pi@fc8a155 packages/coding-agent/src/modes/interactive/components/tree-selector.ts, internal/session/lineage.go:LineageKindFork and ResolveLineageTip, internal/session/directory.go:SessionDirectoryEntry, internal/tui/slash_sessions.go:/sessions and /resume picker, internal/tui/slash_branch.go:/branch fork seam
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. Per-file mutation queue for native write edit and patch tools

- Phase: 5 / 5.L
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Serialize concurrent file mutations that target the same canonical path across native write, edit, patch, and custom file-task tools while preserving parallel execution for independent files. The queue must resolve symlink aliases for existing files, use cleaned absolute paths for new files, cover the full read-modify-write window, and compose with the existing file staleness registry and atomic writer helpers.
- Trust class: operator, system
- Ready when: The builder can prove behavior with in-memory/fake concurrent tools and temp files; no provider call is needed., The queue is scoped to file mutation paths only and does not serialise unrelated tool calls globally.
- Not ready when: The slice disables concurrent tool execution entirely, relies only on stale-read rejection, or queues only the final write rather than the full mutation window., Symlink aliases for an existing file can still run in parallel and clobber one another.
- Degraded mode: -
- Fixture: `internal/tools/file_mutation_queue_test.go`
- Write scope: `internal/tools/`, `internal/kernel/toolexec.go`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools ./internal/kernel -run 'Test.*(MutationQueue\|FileState\|Atomic\|ToolExec)' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report same-file concurrency, different-file concurrency, symlink alias evidence, and progress validation.
- Acceptance: Two concurrent mutations to the same existing file run in deterministic serial order and preserve both changes when each operation is valid., Concurrent mutations to different files overlap or are not forced through a global lock., Existing-file symlink aliases share one queue key; missing/new files use the resolved absolute path key., Staleness registry and atomic replace behavior remain covered by existing tests.
- Source refs: pi@fc8a155 packages/coding-agent/docs/extensions.md:withFileMutationQueue guidance, pi@fc8a155 packages/coding-agent/src/core/tools/file-mutation-queue.ts, internal/tools/file_state_registry.go:FileStateRegistry, internal/tools/atomic_replace.go:atomic file replace helper, internal/tools/file_task_tools.go:native file task tools, internal/kernel/toolexec.go:tool execution concurrency boundary
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Gormes JSONL RPC mode over agent runtime events

- Phase: 5 / 5.Q
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Expose a local `gormes` JSONL RPC run mode for language-agnostic embedding. The protocol should accept prompt, steer, follow_up, abort, get_state, get_messages, session stats, model/thinking controls where existing runtime seams support them, and stream agent/tool/queue/compaction events as newline-delimited JSON with strict LF framing. It should reuse Gormes kernel/API-server event models and must not require a web server, Pi subprocess, or live provider in tests.
- Trust class: operator, system
- Ready when: The builder starts with stdio JSONL and fake provider/kernel fixtures; no HTTP listener or live credentials are required., The protocol names Gormes-owned events and documents any unsupported Pi command as typed unavailable evidence rather than pretending parity.
- Not ready when: The slice starts a gateway server, opens network ports, depends on Pi RPC clients, or blocks on subscription/OAuth provider credentials., JSON records are split by anything other than LF, raw stderr contaminates stdout, or command responses cannot be correlated by id.
- Degraded mode: -
- Fixture: `cmd/gormes/rpc_mode_test.go`
- Write scope: `cmd/gormes/`, `internal/kernel/`, `internal/gateway/`, `internal/apiserver/`, `pkg/gormes/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes ./internal/kernel ./internal/gateway ./internal/apiserver -run 'Test.*(RPC\|JSONL\|EventStream\|Queue\|Abort)' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report RPC fixture transcript, stdout-cleanliness proof, malformed-command evidence, and progress validation.
- Acceptance: `gormes --mode rpc --no-session` or the chosen subcommand starts a stdin/stdout JSONL loop with a session/header response and no startup chatter on stdout., A fake prompt command emits accepted response, agent/message/tool lifecycle events, and a final agent_end event in deterministic order., Steer/follow_up/abort commands update queue or cancellation state and emit structured responses with request ids., Malformed JSON, unknown commands, and unsupported model/session controls return structured errors without terminating the process unless stdin closes.
- Source refs: pi@fc8a155 packages/coding-agent/docs/rpc.md:Protocol Overview and Commands, pi@fc8a155 packages/coding-agent/docs/json.md:JSON Event Stream Mode, pi@fc8a155 packages/coding-agent/src/modes/rpc/rpc-types.ts, internal/kernel/frame.go:RenderFrame, internal/hermes/events.go:turn/run event types, internal/apiserver/runs.go:run inspection/event surfaces, cmd/gormes/main.go:root command mode selection
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Gormes-owned TUI extension status widget and footer seam

- Phase: 8 / 8.D
- Owner: `tui`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Introduce a small Go-native TUI extension context that lets trusted in-process Gormes extensions add or clear footer status entries, widgets above or below the editor, and working-indicator text/frames. The seam should be typed, width-safe, scoped to the active session, and degrade to no-op evidence in non-interactive modes; it must not execute TypeScript or import Pi packages.
- Trust class: operator, system
- Ready when: The builder defines a Go interface or small adapter layer rather than a script runtime; extension callbacks are fakeable in tests., The first slice only covers TUI status/widget/footer/working indicator rendering, not general tool registration or package installation.
- Not ready when: The slice loads third-party executable extension code, adds npm/TypeScript dependencies, or changes Hermes plugin CLI behavior., The extension seam can write files, mutate provider requests, or bypass existing tool safety in this TUI-only slice.
- Degraded mode: -
- Fixture: `internal/tui/status_bar_ext_test.go`
- Write scope: `internal/tui/`, `internal/kernel/extensions.go`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui -run 'Test.*Extension.*(Status\|Widget\|Footer\|Working)\|TestHermesChrome' -count=1`, `go test ./internal/kernel -run TestExtension -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report fake-extension render tests, non-interactive degraded evidence, and progress validation.
- Acceptance: A fake extension can set, replace, and clear a status entry that renders in the Hermes status/footer area without corrupting width-bounded output., A fake extension can set a widget above or below the editor and the widget composes with todo/panel/status chrome ordering., Working-indicator customization applies during active-turn frames and restores the default when cleared., Non-interactive or nil-extension contexts return typed unavailable/no-op evidence instead of panicking.
- Source refs: pi@fc8a155 packages/coding-agent/docs/extensions.md:ctx.ui.setStatus/setWidget/setFooter/setWorkingIndicator, pi@fc8a155 packages/coding-agent/docs/tui.md:Patterns 4-6, pi@fc8a155 packages/coding-agent/examples/extensions/custom-footer.ts, pi@fc8a155 packages/coding-agent/examples/extensions/status-line.ts, internal/tui/status_bar_ext.go:RenderFaceTicker and RenderContextBar, internal/tui/hermes_chrome.go:HermesChromeInput, internal/kernel/extensions.go:ExtensionChain
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Gormes-owned TUI queued-message widget and busy delivery modes

- Phase: 8 / 8.D
- Owner: `tui`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Adapt Pi's visible steering/follow-up queue pattern into the native Gormes Bubble Tea chat TUI without changing Hermes-compatible slash command semantics. While a turn is active, plain Enter should honor the configured busy-input mode, queued or steering drafts should be visible in the bottom-pinned chrome, queued entries should drain after the kernel returns idle, and the UI must keep Alt/Shift+Enter newline behavior intact.
- Trust class: operator, system
- Ready when: The builder keeps the work in the local Bubble Tea TUI and kernel submit/cancel seams; gateway/channel follow-up behavior remains unchanged., The implementation uses fake frames and pure TUI tests; no provider, gateway process, or live terminal automation is required.
- Not ready when: The slice changes Hermes-compatible Enter, Alt+Enter, Shift+Enter, Ctrl+C, slash dispatch, or active-turn policy semantics instead of only wiring visible queue state., The slice stores queued drafts in a side file, hidden TODO list, or non-session backlog outside the TUI/kernel state path.
- Degraded mode: -
- Fixture: `internal/tui/queued_messages_test.go`
- Write scope: `internal/tui/queued_messages.go`, `internal/tui/update.go`, `internal/tui/view.go`, `internal/tui/hermes_chrome.go`, `internal/tui/*queued*test.go`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui -run 'Test.*Queued\|TestHermesKeybindings_EnterPlainTextHonorsBusyInputMode\|TestHermesChrome' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report focused TUI test output, a short before/after render fixture for queued rows, and progress validation.
- Acceptance: Active-turn drafts submitted in queue mode appear in a three-row queued-message widget above the status rule and do not immediately call Submit., Steer mode shows steering evidence and schedules the draft through the existing active-turn injection path without hiding queued text from the operator., When a frame transitions to idle/failed, queued follow-up drafts drain in FIFO order through the existing submit callback and the widget clears., Alt+Enter and Shift+Enter still insert newlines and never enqueue or submit drafts.
- Source refs: pi@fc8a155 packages/coding-agent/README.md:Message Queue, pi@fc8a155 packages/coding-agent/docs/settings.md:Message Delivery, pi@fc8a155 packages/coding-agent/docs/rpc.md:queue_update events, internal/tui/queued_messages.go:QueuedMessages, internal/tui/update.go:HermesBusyInputMode and ResolveHermesKey, internal/tui/hermes_chrome.go:HermesChromeInput.QueuedMessages, internal/tui/view.go:RenderHermesChrome call site
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Navivox Telegram-inspired chat polish

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: After the connect-and-talk loop and profile contact summary API work, make Navivox feel like a polished Telegram-inspired operator client without changing the Gormes HTTP/WS backend. Render a flat profile-contact list with deterministic avatar, display name, small server label, sanitized latest preview, timestamp, health, attention badges, workspace counts, and mic availability; render the profile chat screen with grouped Telegram-style bubbles, compact timestamps, local send/queued/streaming/done/error ticks, a pinned redacted server/profile/trust banner, always-reachable composer, and a global continuous-voice bar when active. Use Telegram-like draggable sheets for profile/server/action/tool detail flows. Evaluate `v_chat_bubbles` as the bubble renderer (`VBubbleStyle.telegram`, `VCustomBubble` for ToolCallCard, performance config for long transcripts) but fall back to local widgets if the package fails accessibility, theming, performance, or dependency review. This row is visual/interaction polish only: no TDLib, MTProto, Firebase chat backend, Telegram login, telephony, campaigns, or call-center scope.
- Trust class: operator, gateway
- Ready when: `Navivox connect-and-talk first screen` has landed and provides a live chat fixture., `Navivox profile contact summary API` has landed so the list uses server-authoritative profile contacts, not mocks., The UI docs classify Telegram/TDLib references as rendering/lifecycle inspiration only, not backend dependencies., The package/dependency review for `v_chat_bubbles` can be performed without blocking text chat.
- Not ready when: The slice adds Telegram account login, MTProto, TDLib, Firebase Firestore, or another chat backend., The slice makes `v_chat_bubbles` mandatory before proving it can render ToolCallCard as a custom bubble., The slice hides gateway status, auth mode, token-required state, or redaction evidence in favor of decorative chat styling., The slice introduces telephony, campaigns, scheduling, retries, or human handoff.
- Degraded mode: If `v_chat_bubbles` is unsuitable, Navivox keeps the current simple adapter and implements only local Telegram-like preview tiles, grouped bubbles, status ticks, and sheets. Text chat, tool cards, and voice transcript fallback remain usable.
- Fixture: `../navivox-app/test/features/chat/profile_contact_list_test.dart + ../navivox-app/test/features/chat/transcript_bubble_test.dart + ../navivox-app/test/features/chat/transcript_thread_test.dart + ../navivox-app/test/shared/app_shell_test.dart`
- Write scope: `../navivox-app/pubspec.yaml`, `../navivox-app/lib/features/chat/`, `../navivox-app/lib/features/servers/`, `../navivox-app/lib/shared/widgets/`, `../navivox-app/test/features/chat/`, `../navivox-app/test/features/servers/`, `../navivox-app/navivox-chat-ui-research.md`, `../navivox-app/navivox-ui-design.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/features/chat test/features/servers test/router/app_router_test.dart test/shared/app_shell_test.dart'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox opens to a Telegram-inspired chat list and chat surface that is visually polished, operationally useful, and still backed only by the Gormes HTTP/WS gateway.
- Acceptance: Chat list previews show profile avatar, display name, server label, sanitized latest preview, timestamp, health/auth status, attention badges, workspace counts, and mic availability., Chat bubbles are grouped by author and time gap, have compact timestamps and local delivery-state ticks, and update one assistant bubble during streaming., ToolCallCard remains a structured custom bubble or local widget with redaction, status, and expandable details; tool output is not rendered as assistant prose., Profile/server/action/tool detail panels use `DraggableScrollableSheet` or a tested desktop side-panel equivalent., Mobile uses Material 3 `NavigationBar` with Chats, Servers, and Settings; desktop uses a rail/sidebar equivalent; both layouts have widget or golden coverage., A package gate documents whether `v_chat_bubbles` was adopted or rejected, including accessibility, theming, performance, license, and dependency findings.
- Source refs: ../navivox-app/navivox-chat-ui-research.md:2, ../navivox-app/navivox-ui-design.md:2.1, https://pub.dev/packages/v_chat_bubbles, https://docs.flutter.dev/ui/design/material, https://api.flutter.dev/flutter/widgets/DraggableScrollableSheet-class.html, https://github.com/tdlib/td, https://github.com/babakcode/flutter_chat
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. Navivox natural-language profile seed Flutter UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Add the sibling Navivox Flutter profile seed UI that calls the Gormes backend profile-seed API, offers Create from seed in the chat/profile flow, renders the returned editable draft fields, requires explicit workspace path entry or confirmation, applies only through the backend, and then shows the new profile as a contact. The Flutter app must not write TOML or infer/grant workspace roots on its own.
- Trust class: operator, gateway, system
- Ready when: `Navivox natural-language profile seed backend API` has landed so the app can call a real server-authoritative draft/apply API., Navivox app setup/connect flow can reach the configured Gormes HTTP gateway., The UI preserves the server-authoritative config model; the Flutter app requests creation and renders/edit drafts, it does not write TOML.
- Not ready when: The backend profile seed API is missing or unvalidated., The app writes profile config files directly instead of calling Gormes., The UI grants workspace paths inferred from the seed without explicit operator confirmation., The slice adds campaigns, bulk outbound calls, telephony transfer, or call-center scheduling.
- Degraded mode: Without a model/provider, seed creation uses a deterministic local template and marks generation_source=template; invalid or risky seeds are rejected with redacted typed evidence and do not mutate profile config or workspace roots.
- Fixture: `../navivox-app/test/features/profiles/profile_seed_flow_test.dart`
- Write scope: `../navivox-app/lib/features/chat/`, `../navivox-app/lib/features/profiles/`, `../navivox-app/test/features/chat/`, `../navivox-app/test/features/profiles/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/features/profiles test/features/chat/profile_contact_list_test.dart'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter can request a profile seed draft from Gormes, edit/confirm it safely, apply it through the backend, and show the new contact without direct TOML writes or unconfirmed workspace access.
- Acceptance: Navivox profile creation exposes Create from seed, calls the backend draft API, and renders editable profile_id, display_name, instructions, provider/model, tool policy, voice metadata, and workspace suggestions., The app requires explicit workspace path entry/confirmation before apply and never writes TOML directly., After apply, the newly seeded profile appears in the profile/contact list and can be selected for a first chat turn., Provider-unconfigured backend responses render generation_source=template and redacted evidence without blocking first-run use.
- Source refs: ../navivox-app/navivox-ui-design.md:2.8, ../navivox-app/navivox-chat-ui-research.md:10, ../navivox-app/lib/features/chat/, ../navivox-app/lib/features/profiles/, internal/channels/navivox/channel.go, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Navivox HTTP/WS Flutter channel
- Unblocks: Navivox per-profile BYO voice profiles
- Why now: Unblocks Navivox per-profile BYO voice profiles.

## 8. Navivox per-profile BYO voice profiles Flutter UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Add the sibling Navivox Flutter profile/config controls that consume the backend voice-profile contract without writing config files or storing raw secrets.
- Trust class: -
- Ready when: `Navivox per-profile BYO voice profiles backend API` is complete and exposes authenticated read/validate fixtures., The app has authenticated GatewayNavivoxChannel wiring for config/profile backend calls.
- Not ready when: The Flutter app writes profile config TOML directly., Any widget, fixture, log, or snapshot stores raw provider credentials., The UI adds telephony, scheduling, campaigns, or human handoff.
- Degraded mode: -
- Fixture: `../navivox-app/test/features/profiles/; ../navivox-app/test/features/config/`
- Write scope: `../navivox-app/lib/features/profiles/`, `../navivox-app/lib/features/config/`, `../navivox-app/test/features/profiles/`, `../navivox-app/test/features/config/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/features/profiles test/features/config test/features/voice'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter lets operators inspect and edit per-profile voice settings through the backend contract; secrets stay write-only and degraded voice states remain explicit.
- Acceptance: Profile create/edit UI can set and display voice profile fields from the backend schema/read model., Credential controls show status/source refs only and route changes through backend safe config/admin flows., Voice provider fallback evidence from run records is visible after a fake or real voice turn., Text chat remains usable when voice providers are unavailable.
- Source refs: ../navivox-app/navivox-ui-design.md:2.8, ../navivox-app/lib/features/profiles/, ../navivox-app/lib/features/config/, docs/content/building-gormes/architecture_plan/progress.json:Navivox per-profile BYO voice profiles backend API
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Navivox safe config admin Flutter UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Render the Navivox config admin backend contract in the sibling Flutter app: schema-driven controls, redacted current values, diff/validate/apply confirmation, secret set/rotate/delete/test actions, and reload-or-pending-restart status. Flutter consumes backend schema and actions only; it never edits config.toml, .env, or raw secret values directly.
- Trust class: operator, gateway, system
- Ready when: `Navivox safe config admin backend API` has landed and exposes schema/get/diff/validate/apply/reload-or-pending-restart fixtures., The app has an authenticated GatewayNavivoxChannel connection and can route config-admin requests through the backend contract., The UI renders only schema-provided fields and action metadata; no free-form TOML editor.
- Not ready when: The backend API child row is still planned or failing validation., Flutter attempts to edit config.toml or .env directly., Any widget, log, fixture, or snapshot stores raw secret values.
- Degraded mode: When config validation fails, Navivox shows typed field errors and keeps the last-good server config active; secret values are never returned, logged, or echoed, only status/source/redacted evidence.
- Fixture: `../navivox-app/test/features/config/config_screen_test.dart`
- Write scope: `../navivox-app/lib/core/channel/navivox_channel.dart`, `../navivox-app/lib/core/channel/gateway_navivox_channel.dart`, `../navivox-app/lib/features/config/`, `../navivox-app/test/features/config/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/features/config'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter can inspect, validate, and apply safe config changes through backend schema/actions only; secret values stay write-only and pending_restart/reload evidence is visible.
- Acceptance: Schema-driven controls render supported safe config fields and current redacted values from the backend., Diff and validate responses render exact non-secret before/after confirmation and field-scoped errors., Secret controls render set/rotate/delete/test actions and status/source evidence without reading or storing raw secret values., Apply success renders reload_applied or pending_restart evidence from the backend.
- Source refs: ../navivox-app/navivox-decision-record.md:141, ../navivox-app/lib/core/channel/navivox_channel.dart, ../navivox-app/lib/core/channel/gateway_navivox_channel.dart, ../navivox-app/lib/features/config/screens/config_screen.dart, ../navivox-app/test/features/config/config_screen_test.dart, docs/content/building-gormes/architecture_plan/progress.json:Navivox safe config admin backend API
- Unblocks: Navivox per-profile BYO voice profiles
- Why now: Unblocks Navivox per-profile BYO voice profiles.

## 10. Navivox structured tool event cards Flutter UI

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
- Write scope: `../navivox-app/lib/core/gateway/navivox_gateway_protocol.dart`, `../navivox-app/lib/core/channel/gateway_navivox_channel.dart`, `../navivox-app/lib/core/protocol/navivox_event.dart`, `../navivox-app/lib/features/chat/widgets/simple_chat_adapter.dart`, `../navivox-app/test/core/channel/gateway_navivox_channel_test.dart`, `../navivox-app/test/features/chat/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/core/channel/gateway_navivox_channel_test.dart test/features/chat'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter renders backend tool-progress lifecycle events as durable cards with redacted status/artifact evidence and no assistant-prose leakage.
- Acceptance: GatewayNavivoxChannel upserts a single ToolCall message per tool_call_id for started, updated, and finished backend events., ToolCallCard renders status, summary, approval state when present, and bounded artifact rows with id/kind/title/summary/ref., Malformed or oversized event metadata is truncated/redacted in UI fixtures and never becomes assistant prose., Existing chat fixtures remain green for normal assistant streaming and final messages.
- Source refs: docs/content/building-gormes/architecture_plan/progress.json:Navivox structured tool event cards backend API, ../navivox-app/lib/core/protocol/navivox_event.dart:NavivoxToolCall, ../navivox-app/lib/core/channel/gateway_navivox_channel.dart:_upsertToolCall, ../navivox-app/lib/features/chat/widgets/simple_chat_adapter.dart:_ToolCallBody
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
