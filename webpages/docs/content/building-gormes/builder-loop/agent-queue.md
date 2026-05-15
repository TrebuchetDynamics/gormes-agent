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
## 1. PicoClaw-derived channel media and identity regression matrix

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Add a fake-adapter regression matrix, sourced from current PicoClaw channel reports, that proves Gormes preserves sender identity, allowlist decisions, durable rich-media envelopes, voice transcript text, PDF/document attachment metadata, Feishu-style tool-progress notifications, and same-agent final delivery semantics through the channel-neutral gateway path.
- Trust class: operator, gateway, system
- Ready when: Cross-platform image/document MEDIA delivery routing, Telegram voice/audio STT ingress, Gateway fresh-final send/delete fallback, and SessionContext prompt injection remain complete., The worker can use fake gateway channels and synthetic attachments only; no Telegram, Matrix, Feishu, Slack, Discord, PDF parser, or live provider credential is required.
- Not ready when: The slice adds a new live channel adapter, changes provider media content-part serialization, or rewrites existing channel authorization semantics instead of freezing the shared gateway invariants., The fixture stores prompt text, file contents, bot tokens, chat IDs beyond synthetic test IDs, or raw local filesystem paths in evidence.
- Degraded mode: Unsupported channel/media combinations return redacted typed evidence, unknown senders fail closed, and final answers are sent through the existing fresh-final or coalescer fallback without editing placeholder/tool-feedback messages into user-visible finals.
- Fixture: `internal/gateway/picoclaw_channel_regression_test.go`
- Write scope: `internal/gateway/picoclaw_channel_regression_test.go`, `internal/gateway/`, `internal/channels/threadtext/`, `internal/channels/telegram/`, `internal/channels/feishu/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway -run '^TestPicoClawChannelRegression_' -count=1`, `go test ./internal/gateway ./internal/channels/threadtext -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: The PicoClaw-derived channel regression fixture proves sender identity, allowlist, durable media envelope, voice transcript, tool-progress notification, and final-delivery invariants without live channels.
- Acceptance: TestPicoClawChannelRegression_SenderIdentityAndAllowlist proves Matrix/thread-text-style events inject sender identity into session context and deny disallowed senders before provider submission., TestPicoClawChannelRegression_RichMediaEnvelopePersists proves image, PDF/document, and voice-transcript attachments keep durable metadata and transcript text through the gateway event and final response path., TestPicoClawChannelRegression_FinalDeliveryDoesNotEditToolPlaceholder proves steering-heavy finals are emitted as final messages or fresh-final fallbacks, not as edits to tool-feedback placeholders., TestPicoClawChannelRegression_ToolProgressNotificationsAreComplete proves multi-tool progress events produce complete notification-center evidence for channels that cannot edit every intermediate message.
- Source refs: https://github.com/sipeed/picoclaw/issues/2855, https://github.com/sipeed/picoclaw/issues/2843, https://github.com/sipeed/picoclaw/issues/2839, https://github.com/sipeed/picoclaw/issues/2817, https://github.com/sipeed/picoclaw/issues/2816, https://github.com/sipeed/picoclaw/issues/2815, https://github.com/sipeed/picoclaw/issues/2798, https://github.com/sipeed/picoclaw/issues/2785, https://github.com/sipeed/picoclaw/issues/2702, internal/gateway/channel.go, internal/gateway/coalesce.go, internal/gateway/session_context.go, internal/gateway/media_delivery.go, internal/channels/threadtext/contract.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. PicoClaw-derived session ledger read-model regression matrix

- Phase: 9 / 9.F
- Owner: `memory`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Add a session-ledger read-model regression matrix that proves Gormes stores and renders multiple user messages in a turn, per-message timestamps, sender attribution, durable attachment references, and non-destructive reset metadata without collapsing history to session.updated or deleting older channel history.
- Trust class: operator, gateway, system
- Ready when: Transcript export, session lineage, session reset notification parity, and manual reset boundary hooks remain complete., Tests can use temp bbolt/SQLite stores and synthetic gateway session IDs only.
- Not ready when: The slice changes recall ranking, Goncho memory extraction, provider turn execution, or channel delivery behavior instead of freezing ledger rendering and metadata invariants., The implementation performs destructive cleanup of historical session rows as part of reset handling.
- Degraded mode: Legacy session records without per-message timestamps render explicit unknown timestamp evidence and keep original order; reset operations mark boundaries without deleting prior transcript rows.
- Fixture: `internal/session/picoclaw_ledger_regression_test.go`
- Write scope: `internal/session/picoclaw_ledger_regression_test.go`, `internal/session/`, `internal/transcript/`, `internal/store/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/session ./internal/transcript ./internal/store -run '^TestPicoClawSessionLedger_' -count=1`, `go test ./internal/session ./internal/transcript ./internal/store -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Session ledger fixtures prove multi-message visibility, per-message timestamps, non-destructive reset boundaries, and durable attachment reopen behavior.
- Acceptance: TestPicoClawSessionLedger_MultipleUserMessagesRemainVisible proves two user messages in one conversational turn both render in transcript export and API-style read models., TestPicoClawSessionLedger_PerMessageTimestampsDoNotUseSessionUpdated proves message timestamps are independent from session.updated while legacy records degrade visibly., TestPicoClawSessionLedger_ResetBoundaryIsNonDestructive proves fresh-session reset metadata creates a boundary and preserves prior messages for history/search., TestPicoClawSessionLedger_DurableAttachmentRefsSurviveReopen proves reopened history keeps durable attachment references and redacts non-durable temp refs with evidence.
- Source refs: https://github.com/sipeed/picoclaw/issues/2820, https://github.com/sipeed/picoclaw/issues/2796, https://github.com/sipeed/picoclaw/issues/2795, https://github.com/sipeed/picoclaw/issues/2787, internal/session/directory.go, internal/session/lineage.go, internal/transcript/markdown.go, internal/store/recording.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. PicoClaw-derived provider stream and auth regression matrix

- Phase: 9 / 9.F
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Add a provider regression matrix from PicoClaw reports that replays fake OpenRouter reasoning-model chunks, Codex Responses output_item.done events, 401/auth failures, local LM Studio/OpenAI-compatible model routing, and retryable LLM-call failures through Gormes provider seams with no live credentials.
- Trust class: operator, system
- Ready when: Cross-provider reasoning-tag sanitization, Codex stream repair, LM Studio provider adapter, provider retry diagnostics, and OpenRouter compatible-provider routing remain complete., The worker can replay synthetic SSE/JSON events through existing test seams; no OpenRouter, Codex, OpenAI, LM Studio, or Ollama endpoint is contacted.
- Not ready when: The slice changes token-vault storage, starts a local model server, or rewrites provider registry semantics instead of adding focused replay fixtures and any narrowly required adapter fixes., The test fixture requires real API keys, browser login, OAuth device code flow, or network access.
- Degraded mode: Provider failures surface classified, action-oriented diagnostics with provider/model/auth source evidence while raw stream frames remain available for audit and never leak reasoning text into assistant-visible content.
- Fixture: `internal/hermes/picoclaw_provider_regression_test.go`
- Write scope: `internal/hermes/picoclaw_provider_regression_test.go`, `internal/hermes/`, `internal/provider/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run '^TestPicoClawProviderRegression_' -count=1`, `go test ./internal/hermes ./internal/provider -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Provider replay fixtures prove reasoning isolation, Codex output_item.done content, auth diagnostics, local model routing, and retry classification without live credentials.
- Acceptance: TestPicoClawProviderRegression_OpenRouterReasoningHidden proves reasoning-tag and reasoning_content variants do not appear in assistant-visible content or stored final text., TestPicoClawProviderRegression_CodexOutputItemDoneYieldsAssistantText proves ChatGPT/Codex output_item.done stream items produce visible assistant text instead of empty final responses., TestPicoClawProviderRegression_Auth401NamesCredentialSource proves 401/invalid-key failures report provider, credential source, and next action without printing the key., TestPicoClawProviderRegression_LMStudioLocalModelRouting proves local OpenAI-compatible model IDs route deterministically through the LM Studio/OpenAI-compatible adapter without model-list false negatives., TestPicoClawProviderRegression_RetryableLLMFailureUsesClassifiedRetry proves retryable stream/open errors use the existing retry classifier and stop on fatal auth errors.
- Source refs: https://github.com/sipeed/picoclaw/issues/2769, https://github.com/sipeed/picoclaw/issues/2745, https://github.com/sipeed/picoclaw/issues/2674, https://github.com/sipeed/picoclaw/issues/2404, https://github.com/sipeed/picoclaw/issues/629, https://github.com/sipeed/picoclaw/issues/28, internal/hermes/stream.go, internal/hermes/http_client.go, internal/hermes/codex_responses_stream.go, internal/hermes/reasoning_tag_sanitizer.go, internal/hermes/lmstudio_adapter.go, internal/hermes/errors.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. MCP Streamable HTTP session lifecycle compatibility

- Phase: 9 / 9.F
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Extend the MCP HTTP client compatibility fixture to the current Streamable HTTP contract: initialize captures `Mcp-Session-Id`, all subsequent POST/GET/DELETE requests replay that header, SSE responses are accepted from the single MCP endpoint, 404 with a session header triggers a new initialization path, and legacy HTTP+SSE `/sse` endpoint events with `sessionId` are classified as backwards-compatibility input rather than silently dropping the session.
- Trust class: operator, system
- Ready when: MCP HTTP transport, OAuth refresh recovery, circuit breaker reconnect, and managed tool gateway bridge remain complete., The worker can use httptest servers returning JSON and text/event-stream responses; no real MCP server or OAuth provider is required.
- Not ready when: The slice implements unrelated MCP sampling/tool schemas, starts a live MCP server, or changes stdio behavior., The test relies on a network server outside httptest or assumes the removed 2024-11-05 HTTP+SSE transport is the default transport.
- Degraded mode: Servers that require a session ID but omit or reject it produce mcp_session_required or mcp_session_expired evidence; unsupported legacy SSE endpoint shapes remain visible compatibility failures instead of empty-session POSTs.
- Fixture: `internal/tools/mcp_streamable_http_session_test.go`
- Write scope: `internal/tools/mcp_streamable_http_session_test.go`, `internal/tools/mcp_http.go`, `internal/tools/mcp_client.go`, `internal/tools/managed_tool_gateway.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run 'TestMCPStreamableHTTP_\|TestMCPLegacySSEEndpoint_' -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: MCP HTTP fixtures prove session ID lifecycle, JSON/SSE single-endpoint behavior, expired-session recovery evidence, legacy SSE sessionId compatibility handling, and DELETE tolerance.
- Acceptance: TestMCPStreamableHTTP_CapturesAndReplaysSessionID proves Initialize stores `Mcp-Session-Id` and ListTools/CallTool send it on subsequent requests., TestMCPStreamableHTTP_SingleEndpointAcceptsJSONOrSSE proves the client posts to one endpoint with Accept including application/json and text/event-stream, and parses a JSON-RPC response carried by SSE., TestMCPStreamableHTTP_ExpiredSessionReinitializes proves HTTP 404 with an existing session ID clears the session and returns typed evidence for reinitialize., TestMCPLegacySSEEndpoint_SessionIDCompatibilityEvidence proves `/sse` endpoint events containing `/message?sessionId=...` are either upgraded into compatibility state or fail with explicit legacy_sse_unsupported evidence, never an empty sessionId POST., TestMCPStreamableHTTP_DeleteSessionHeader proves Close or explicit termination sends DELETE with the stored session header when supported and tolerates 405.
- Source refs: https://modelcontextprotocol.io/specification/2025-06-18/basic/transports#streamable-http, https://modelcontextprotocol.io/specification/2025-06-18/basic/transports#session-management, https://modelcontextprotocol.io/specification/2025-06-18/basic/transports#backwards-compatibility, https://github.com/sipeed/picoclaw/issues/2782, https://github.com/sipeed/picoclaw/issues/2546, internal/tools/mcp_http.go, internal/tools/mcp_client.go, internal/tools/mcp_oauth_store.go, internal/tools/managed_tool_gateway.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Dynamic agent identity inheritance regression matrix

- Phase: 9 / 9.F
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Add a dynamic-agent identity regression matrix that proves spawned or delegated agents keep explicit parent linkage while receiving their own SOUL/persona, tool policy, AGENTS.md scope, and memory/search scope, so child agents do not silently inherit the root agent role as their own identity.
- Trust class: operator, child-agent, system
- Ready when: Goncho-backed dynamic agent registry, deterministic subagent runtime, durable job ledger, and tool allowlist policy remain complete., Tests can use fake child runners, temp Goncho state, and synthetic AGENTS.md/SOUL.md fixtures; no real Codex, Claude, opencode, or provider child process is required.
- Not ready when: The slice changes live coding-agent execution, starts child CLIs, or introduces agent-to-agent messaging semantics beyond identity and scope fixtures., The implementation treats child agents as fully independent users or erases parent lineage in the durable job ledger.
- Degraded mode: If a child identity cannot be resolved, delegation refuses with child_identity_unresolved evidence and leaves the parent turn untouched rather than running as an ambiguous root persona.
- Fixture: `internal/subagent/picoclaw_identity_regression_test.go`
- Write scope: `internal/subagent/picoclaw_identity_regression_test.go`, `internal/subagent/`, `internal/goncho/`, `internal/agent/`, `internal/skills/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/subagent ./internal/goncho ./internal/agent ./internal/skills -run '^TestDynamicAgentIdentity_' -count=1`, `go test ./internal/subagent ./internal/goncho ./internal/agent ./internal/skills -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Dynamic-agent identity fixtures prove child persona, AGENTS scope, explicit tool-policy inheritance, and child-aware memory scope without launching live child CLIs.
- Acceptance: TestDynamicAgentIdentity_ChildPersonaOverridesRootRole proves child SOUL/persona is rendered as the active identity while parent identity remains lineage metadata., TestDynamicAgentIdentity_AGENTSScopesDoNotBleedAcrossChild proves child AGENTS.md/project context is selected by child workspace scope and does not overwrite the parent turn., TestDynamicAgentIdentity_ToolPolicyInheritedOnlyWhenExplicit proves allow/deny/glob tool policy inheritance is explicit and visible in child launch evidence., TestDynamicAgentIdentity_MemoryScopeIsChildAware proves child memory/search scope names both child agent ID and parent lineage without merging unrelated root-agent facts.
- Source refs: https://github.com/sipeed/picoclaw/issues/1934, https://github.com/sipeed/picoclaw/issues/2148, https://github.com/sipeed/picoclaw/issues/2775, https://github.com/sipeed/picoclaw/issues/294, https://github.com/sipeed/picoclaw/issues/284, internal/subagent/, internal/goncho/dynamic_agents.go, internal/agent/middleware.go, internal/skills/
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Native TUI /model slash command binding over the existing model picker

- Phase: 5 / 5.Q
- Owner: `tui`
- Size: `large`
- Status: `planned`
- Priority: `P1`
- Contract: The native Bubble Tea TUI treats `/model` (and the `/m` prefix) as a local operator command, not prompt text: dispatching it opens the already-implemented ModelPicker overlay (internal/tui/model_picker.go RenderModelPicker/UpdateModelPicker — a TUI-LOCAL overlay, unlike the kernel-driven Approval/Clarify/Secret panels, so it needs its own Model overlay state + update.go key routing + view.go render slot), clears the editor, never calls Submitter; confirming applies an IN-SESSION model switch; cancel returns unchanged. BLOCKED: builder-pass 2026-05-15 established there is NO in-session model-switch seam in the local kernel path — PlatformEventKind is {Submit,Cancel,Quit,ResetSession,Steer} with no model override; kernel.go SetModel is construction-only; the completed 5.O picker is config-TOML-persist only; SessionModelOverride is gateway-server-only and not wired to the local Bubble Tea kernel. This row therefore depends on the new 'Kernel in-session model-switch seam for the native TUI' prerequisite. The picker render/key engine already exists and MUST be reused, not reimplemented; the missing piece is the apply seam plus a model-catalog -> internal/tui data seam.
- Trust class: -
- Ready when: SATISFIED — the 'Kernel in-session model-switch seam for the native TUI' prerequisite row is COMPLETE: PlatformEventSetModel + kernel.SetSessionModel(provider,model) exist and are fixture-proven for the same-provider in-session switch (the /model picker's primary affordance). Cross-provider client swap is a separate non-blocking follow-up row, not a /model blocker., Catalog seam: SATISFIED — the picker is populated from existing internal/hermes.ListPickerProviders() (internal/tui already imports internal/hermes); no cmd/gormes catalog import needed., The native TUI slash registry consumes recognized commands before kernel submit (5.Q 'Native TUI slash-command dispatch table' complete, satisfied).
- Not ready when: The slice ignores the shipped kernel.SetSessionModel/PlatformEventSetModel apply seam (e.g. only persists to config TOML or no-ops on confirm) and ships a non-functional /model that fails acceptance., The slice reimplements RenderModelPicker/UpdateModelPicker instead of reusing internal/tui/model_picker.go., The slice binds the local TUI to the gateway-only SessionModelOverride instead of the local kernel seam., Unknown or failing `/model` invocations leak raw slash text to the model.
- Degraded mode: If the model catalog is unavailable, `/model` is consumed with `model: ...` status evidence instead of forwarding the slash text to the model or silently dropping it; the picker is not opened with an empty/invalid catalog.
- Fixture: `internal/tui/slash_model_test.go; cmd/gormes/tui_model_slash_test.go`
- Write scope: `internal/tui/model.go`, `internal/tui/update.go`, `internal/tui/view.go`, `internal/tui/slash_dispatch.go`, `internal/tui/slash_model.go`, `internal/tui/slash_model_test.go`, `cmd/gormes/main.go`, `cmd/gormes/tui_model_slash_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui -run 'TestModelSlash\|TestHermesSlashDispatchBehavior\|ModelPicker' -count=1`, `go test ./cmd/gormes -run TestTUIModelSlash -count=1`, `go run ./cmd/progress validate`
- Done signal: Native TUI `/model` dispatch is fixture-proven over the reused ModelPicker engine, consumes slash text instead of leaking it, applies the model switch through the existing seam, and the 'recognized but unavailable' fallback no longer fires for /model.
- Acceptance: TUI fixtures prove `/model` and the `/m` prefix are handled by the default slash registry, clear the editor, and never call Submitter., Fixtures prove dispatch opens the reused ModelPicker overlay populated from the model catalog and that key events route to UpdateModelPicker while it is active., Fixtures prove confirming a selection applies the model switch to the active session via the existing override seam and that cancel leaves the model unchanged., Failure fixtures prove catalog/seam errors surface as `model: ...` status evidence without raw slash leakage, and that `/model` no longer produces the 'recognized but unavailable in the native TUI' message.
- Source refs: internal/tui/model_picker.go (ModelPickerState/ProviderEntry/ModelEntry/ModelPickerResult/modelPickerConfirmedMsg/RenderModelPicker/UpdateModelPicker — reuse engine, do not reimplement), internal/tui/slash_dispatch.go (NewDefaultSlashRegistry; slashFallbackResult/slashKnownUnhandledStatus produce today's 'recognized but unavailable'), internal/tui/update.go (Model.Update key routing — local overlay must intercept keys when active), internal/tui/view.go (render slot — the local picker overlay is NOT a kernel panel, RenderActivePanel does not cover it), internal/tui/model.go (new local overlay state field), internal/kernel/frame.go (PlatformEventKind {Submit,Cancel,Quit,ResetSession,Steer} — NO model override; proves the missing seam), internal/gateway/model_picker.go (SessionModelOverride — gateway-server-only, NOT the local kernel seam; do not bind the local TUI to this), cmd/gormes/model.go (5.O picker — config-TOML-persist only, NOT a live session switch), ./hermes-agent/ui-tui/src/app/slash/registry.ts (slash dispatch parity reference, as cited by the completed /kanban row), progress 5.O 'Gormes model interactive provider/model picker' (config-only, complete); prerequisite row 'Kernel in-session model-switch seam for the native TUI'
- Unblocks: Native TUI slash handler-port coverage
- Why now: Unblocks Native TUI slash handler-port coverage.

## 7. Termux storage and path safety audit

- Phase: 1 / 5.X
- Owner: `orchestrator`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Audit and test Gormes path selection under synthetic Termux env so config, dotenv secrets, sessions, gateway state, SQLite/Goncho, browser temp dirs, and generated files land only under configured GORMES_HOME/XDG/HOME locations while install publication remains $PREFIX/bin/gormes. No runtime code may hardcode desktop workspace paths such as /home/xel or workspace-mineru.
- Trust class: operator, system
- Ready when: Termux runtime doctor check is complete., Existing config path helpers can be tested with temp HOME/XDG/GORMES_HOME and synthetic PREFIX., Tests use temp dirs only and never inspect the developer's live ~/.gormes or Termux state.
- Not ready when: Tests depend on /data/data/com.termux existing on the host., Any command writes outside temp HOME/XDG/GORMES_HOME or the synthetic $PREFIX/bin install target., The implementation hardcodes workspace-mineru, /home/xel, or desktop-only paths.
- Degraded mode: If an Android path is unavailable, commands must return typed path/readiness warnings instead of writing into unexpected shared storage or desktop-only paths.
- Fixture: `internal/config Termux path fixtures plus cmd/gormes doctor/config/goncho smoke fixtures`
- Write scope: `internal/config/`, `internal/store/`, `internal/goncho/`, `cmd/gormes/`, `internal/installtest/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/config ./cmd/gormes -run 'Termux\|GatewayRuntimeStatusPath\|ConfigPath\|Goncho' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Synthetic Termux path fixtures prove runtime state stays under configured homes and never hardcodes desktop checkout paths.
- Acceptance: Synthetic Termux tests prove ConfigPath, EnvPath, GatewayRuntimeStatusPath, memory DB paths, and Goncho DB paths stay under temp HOME/XDG/GORMES_HOME., Doctor/config smoke fixtures under synthetic Termux do not create files outside the allowed roots., Install dry-run remains the only path that targets $PREFIX/bin/gormes., No Termux runtime path code depends on root permissions or shared Android storage.
- Source refs: internal/config/config.go, internal/store/, internal/goncho/, cmd/gormes/goncho.go, cmd/gormes/doctor.go, install.sh
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Termux gateway foreground tmux lifecycle

- Phase: 1 / 5.X
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Gateway lifecycle commands and docs present a Termux-specific foreground/tmux model: Telegram/Discord/Slack gateways are supported from a foreground shell or tmux session, systemd/Windows service assumptions are not advertised, and doctor/status guidance names termux-wake-lock plus Android battery settings as best-effort survival aids. The implementation must preserve the same gateway command names and JSON contracts as desktop Linux.
- Trust class: operator, gateway, system
- Ready when: Termux runtime doctor check is complete., Gateway command tests can run with temp GORMES_HOME and fake runtime status stores., Termux install docs identify foreground/tmux as the supported local gateway model.
- Not ready when: The command tries to install systemd units, Android services, or Termux:Boot entries by default., Doctor/status claims guaranteed unattended background uptime on Android., Gateway command names, flags, or JSON shapes diverge from desktop Linux only for Termux.
- Degraded mode: If Termux lacks tmux or termux-wake-lock, gateway startup remains possible but doctor/status emits WARN guidance. Android process death is treated as recoverable operator environment behavior, not a Gormes crash.
- Fixture: `cmd/gormes gateway/doctor fixtures under synthetic Termux env`
- Write scope: `cmd/gormes/`, `internal/gateway/`, `internal/doctor/`, `webpages/docs/content/install/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes -run 'Test.*Termux.*Gateway\|TestDoctorCommand_JSONIncludesTermuxRuntimeWhenDetected' -count=1`, `go test ./internal/gateway -count=1`, `go run ./cmd/progress validate`
- Done signal: Gateway fixtures and docs prove Termux uses the same operator CLI with foreground/tmux lifecycle guidance and bounded Android process-survival claims.
- Acceptance: Synthetic Termux doctor/status output includes foreground/tmux and wake-lock guidance., Gateway start/status/stop command surfaces keep the same names and JSON contracts under synthetic Termux env., Termux docs explain tmux, termux-wake-lock, battery optimization, and Termux:Boot as operator-managed aids., No test starts live Telegram/Discord/Slack connections or Android services.
- Source refs: cmd/gormes/gateway.go, cmd/gormes/gateway_status.go, cmd/gormes/doctor.go, internal/gateway/status.go, internal/doctor/termux.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Termux notification bridge via termux-api

- Phase: 1 / 5.X
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Add an optional Termux notification adapter that shells out to termux-notification only when Termux and the command are detected. Gateway/long-run status can emit Android notifications through this adapter, while non-Termux hosts and Termux hosts without Termux:API degrade to structured no-op/WARN evidence. The adapter must redact secrets and never make termux-api a hard dependency.
- Trust class: operator, gateway, system
- Ready when: Termux runtime doctor check is complete., A small notification sender interface can be injected into gateway/status paths without changing core gateway contracts., Tests can fake command lookup and command execution.
- Not ready when: The adapter invokes live termux-notification in tests., Missing Termux:API fails doctor, gateway, or long-running tasks., Notification text can include provider tokens, bot tokens, prompts containing secrets, or raw command output without redaction.
- Degraded mode: Missing termux-notification or missing Termux:API app returns optional_notification_unavailable evidence; Gormes continues normally without Android notifications.
- Fixture: `internal/gateway or internal/tools Termux notification adapter tests with fake exec runner`
- Write scope: `internal/gateway/`, `internal/tools/`, `internal/doctor/`, `cmd/gormes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway ./internal/tools -run 'Termux.*Notification\|Notification.*Termux' -count=1`, `go test ./cmd/gormes -run 'Termux\|Notification' -count=1`, `go run ./cmd/progress validate`
- Done signal: Optional termux-api notification adapter sends through fake exec under Termux and degrades cleanly everywhere else.
- Acceptance: Fake-exec tests prove Termux notification sends title/body through termux-notification with bounded arguments., Non-Termux and missing-command tests return structured no-op/WARN evidence., Doctor/status output references notification availability without requiring Termux:API., Secret redaction tests prove tokens are not passed into notification bodies.
- Source refs: internal/doctor/termux.go, internal/tools/voice_mode_env.go:termux-api detection precedent, internal/gateway/, cmd/gormes/kanban_notify_test.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. Termux real-device smoke evidence

- Phase: 1 / 5.X
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Capture a dated real-device no-root Android Termux smoke record for the current release: install via repo-root install.sh release asset, run gormes version, gormes doctor --offline --json, gormes config check, initialize SQLite/Goncho state, and run a provider-backed gormes chat -q "hello from Termux" when a test credential is available. The evidence must record Android/Termux versions, device arch, install method, and any caveats without leaking credentials.
- Trust class: operator, system
- Ready when: Termux runtime doctor check is complete., Termux install and release smoke guide is complete., A real no-root Android arm64/aarch64 Termux environment is available to the operator.
- Not ready when: The evidence is only CI simulation or local Linux fake TERMUX_VERSION output., The smoke transcript includes raw provider keys, bot tokens, device-private paths beyond normal Termux paths, or personal chat IDs., The smoke uses source build as the primary install path unless the release asset is explicitly unavailable.
- Degraded mode: If no provider credential is available, record provider-backed oneshot as skipped with credential-unavailable evidence; local install/version/doctor/config/Goncho smoke remains required.
- Fixture: `webpages/docs/content/install/termux-smoke.md or release evidence note`
- Write scope: `webpages/docs/content/install/`, `docs/content/building-gormes/architecture_plan/progress.json`, `README.md`
- Test commands: -
- No test required: Manual real-device evidence row; CI simulation cannot replace the Android smoke transcript.
- Done signal: A dated redacted real-device Termux smoke record is checked in and linked from the install docs/progress row.
- Acceptance: Evidence records exact date, device arch, Android version, Termux version, and Gormes version/commit., Evidence shows install.sh release-binary path into $PREFIX/bin/gormes., Evidence includes gormes version, gormes doctor --offline --json, gormes config check, and SQLite/Goncho initialization outputs or redacted summaries., Provider-backed gormes chat -q succeeds or is explicitly skipped for missing test credential., The public compatibility claim remains bounded to the proven support matrix.
- Source refs: install.sh, cmd/gormes/version.go, cmd/gormes/doctor.go, cmd/gormes/config.go, cmd/gormes/goncho.go, internal/doctor/termux.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
