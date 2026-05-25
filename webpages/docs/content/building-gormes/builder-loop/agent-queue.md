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
## 1. Gormes-owned TUI extension status widget and footer seam

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

## 2. Gormes-owned TUI queued-message widget and busy delivery modes

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

## 3. Navivox Telegram-inspired chat polish

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

## 4. Navivox natural-language profile seed Flutter UI

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

## 5. Navivox per-profile BYO voice profiles Flutter UI

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

## 6. Navivox safe config admin Flutter UI

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

## 7. Navivox structured tool event cards Flutter UI

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

## 8. Navivox voice run records Flutter inspection UI

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

## 9. Hermes integrations claim audit + source-backed plugin/skill parity map

- Phase: 8 / 8.C
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Turn the sanitized Reddit/WebAfterAI Hermes integrations post into a source-backed parity map without accepting marketing shorthand as fact: classify each named integration as first-party bundled skill, bundled plugin/backend, gateway/platform/tool, optional skill, indirect web/browser/MCP/scraping workflow, Gormes-owned candidate, or unsupported/excluded claim. The audit must explicitly handle cases where a workflow is achievable through generic web scraping, browser automation, MCP, or Firecrawl-style extraction without being a direct Hermes plugin or tool, and it must not create implementation rows for Reddit, Stripe, InsForge, Graphiti/Zep, or Fireflies unless exact current Hermes source appears.
- Trust class: operator, system
- Ready when: The audit uses only sanitized transcript text plus checked-in Hermes/Gormes source refs; no live private ~/.hermes, credentials, browser sessions, or external API accounts are read., Each of the 12 post items is classified with source refs or explicit unsupported/excluded evidence., Indirect capabilities are allowed as a separate class: generic web scraping, browser automation, Firecrawl extraction, MCP, or skill workflows may satisfy a use case without proving a direct Hermes plugin/tool exists.
- Not ready when: The row is used to implement all integrations in one pass instead of producing a bounded source-backed audit/map., Unsupported claims are copied into README/docs as if they were Hermes-native integrations., The audit treats `hermes plugins install reddit\|stripe\|insforge\|graphiti\|fireflies` as valid without exact current Hermes source or an external plugin repository URL., The audit reads live user config, token stores, memory databases, or private home directories as evidence.
- Degraded mode: Until the claims are source-classified, public roadmap and parity work can overstate Hermes/Gormes integration breadth by treating scraped workflows, optional skills, and unsupported social-post claims as native plugins.
- Fixture: `webpages/docs/content/building-gormes/architecture_plan/hermes-integrations-claim-audit.md`
- Write scope: `webpages/docs/content/building-gormes/architecture_plan/hermes-integrations-claim-audit.md`, `webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`, `webpages/docs/content/building-gormes/architecture_plan/upstream-coverage-ledger.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`, `README.md`
- Test commands: `go run ./cmd/progress validate`, `go test ./webpages/docs -run 'TestUpstreamCoverageLedgerMatchesSourceClasses\|TestProgressCanonical' -count=1`, `git diff --check`
- Done signal: Report the 12-row classification table, exact Hermes source refs, unsupported/excluded claims, indirect scraping/browser/MCP classifications, and any newly-created follow-up progress row names.
- Acceptance: A checked-in audit document or architecture-plan section lists all 12 post items and classifies each as direct first-party skill/plugin/tool/gateway, optional skill, indirect scraping/browser/MCP workflow, Gormes-owned candidate, or unsupported/excluded., The audit explicitly notes that some user-visible workflows are not direct tools: e.g. competitor/site/reddit-style research may be covered by generic web search/extract/crawl, browser automation, or future MCP/web-scraping rows rather than a named Hermes Reddit plugin., Unsupported/excluded claims for Reddit, Stripe, InsForge, Graphiti/Zep, and Fireflies remain excluded or row-backed as discovery-only until source refs are found., Any follow-up implementation intent is routed into separate small progress rows by source class; this audit row does not broaden into a 12-integration implementation batch., Public messaging/docs are updated only with evidence-backed wording and avoid inflated integration counts.
- Source refs: sanitized user-provided Reddit/WebAfterAI transcript 2026-05-24: '12 Hermes Integrations That Actually Matter', hermes-agent/hermes_cli/plugins_cmd.py@43e566f77: `hermes plugins install` clones Git plugins into ~/.hermes/plugins and does not imply a built-in short-name registry for every social-post claim, hermes-agent/hermes_cli/plugins.py@43e566f77: bundled/user/project/pip plugin discovery and opt-in semantics, hermes-agent/skills/productivity/google-workspace/SKILL.md@43e566f77: first-party Gmail/Calendar/Drive/Docs/Sheets skill, hermes-agent/skills/note-taking/obsidian/SKILL.md@43e566f77: filesystem-first Obsidian vault skill, hermes-agent/plugins/web/firecrawl/plugin.yaml@43e566f77 and provider.py: bundled Firecrawl web backend with direct/gateway/self-hosted config, hermes-agent/tools/web_tools.py@43e566f77: generic web_search/web_extract/web_crawl dispatch; supports web-scraping/extraction workflows without naming them as native integrations, hermes-agent/skills/github/DESCRIPTION.md@43e566f77 and skills/github/*/SKILL.md: GitHub auth/repo/issues/PR/code-review skills, hermes-agent/skills/media/youtube-content/SKILL.md@43e566f77: YouTube transcript helper skill, hermes-agent/gateway/platforms/discord.py@43e566f77 and hermes-agent/tools/discord_tool.py@43e566f77: Discord gateway and Discord admin/core tools, hermes-agent/optional-skills/productivity/telephony/SKILL.md@43e566f77 and scripts/telephony.py: Twilio, Bland.ai, and Vapi optional telephony skill, hermes-agent/gateway/platforms/sms.py@43e566f77: Twilio-backed SMS gateway contract, repository search 2026-05-24: no first-party Hermes refs found for reddit, stripe API plugin, insforge, graphiti/zep, or fireflies beyond incidental text
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. CLIProxyAPI-compatible upstream route adapter

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P3`
- Contract: After the Gormes Router MVP exists, optionally allow a CLIProxyAPI server to be configured as a normal OpenAI-compatible upstream base URL. This must not import CLIProxyAPI runtime code, management APIs, OAuth automation, or multi-account pooling; it only treats CLIProxyAPI as a user-configured upstream endpoint.
- Trust class: operator, system
- Ready when: The builder uses fake providers/httptest and checked-in fixtures, not live credentials or locally installed Ollama/LM Studio., The implementation preserves the user-owned credential boundary from the router plan.
- Not ready when: The slice claims free/unlimited LLM access, requires Ollama/LM Studio, automates OAuth/browser token capture, or copies CLIProxyAPI runtime code., Secrets appear in logs, status JSON, docs, progress evidence, or tests.
- Degraded mode: -
- Fixture: `internal/provider/router/cliproxy_upstream_test.go`
- Write scope: `internal/provider/router/`, `internal/config/`, `cmd/gormes/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/provider/router -run 'TestRouterCLIProxyAPIUpstream' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report focused fake-provider test output, redaction evidence, and progress validation.
- Acceptance: A CLIProxyAPI-style upstream can be represented as a custom OpenAI-compatible route with base_url and api_key_env., The adapter only relies on /v1/models and /v1/chat/completions-compatible behavior., No OAuth, management API, WebSocket, token scraping, or account-pool semantics are added., Tests use httptest fake CLIProxyAPI-compatible responses and redaction assertions.
- Source refs: docs/content/building-gormes/architecture_plan/gormes-router-plan.md:Config schema, CLIProxyAPI@50d19e2 README.md: /v1/chat/completions and provider route aliases, CLIProxyAPI@50d19e2 docs/sdk-advanced.md: /v1/models exposure, internal/hermes/provider_transport.go:chat_completions transport
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
