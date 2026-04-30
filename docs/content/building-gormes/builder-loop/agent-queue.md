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

<!-- PROGRESS:START kind=agent-queue -->
## 1. Hermes memory tool over Goncho/local durable store

- Phase: 3 / 3.F
- Owner: `memory`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Expose the Hermes-visible `memory` tool with add, replace, and remove actions over memory/user targets, backed by Goncho or local durable USER.md/MEMORY.md storage, while preserving safe write responses, redaction, locks, and prompt-insertion semantics.
- Trust class: system, operator
- Ready when: Hermes memory tool actions and Gormes Goncho/local durable context seams have been source-mapped., Temp durable memory fixtures can prove add/replace/remove behavior without reading live USER.md/MEMORY.md or external Honcho.
- Not ready when: -
- Degraded mode: If durable memory storage is unavailable, the tool returns memory_store_unavailable evidence and does not mutate prompt context or transcripts.
- Fixture: `internal/tools/memory_tool_test.go + internal/memory`
- Write scope: `cmd/gormes/registry.go`, `cmd/gormes/registry_test.go`, `internal/tools/memory_tool.go`, `internal/tools/memory_tool_test.go`, `internal/memory/`, `internal/goncho/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/tools ./internal/memory ./internal/goncho -run 'MemoryTool\|DurableUserContext\|Goncho' -count=1`, `GOCACHE=/tmp/gormes-go-cache go test ./cmd/gormes -run Registry -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Hermes-compatible `memory` tool fixtures prove safe durable add/replace/remove behavior over Goncho/local memory stores.
- Acceptance: Default tool registry exposes a Hermes-compatible `memory` descriptor., Add, replace, and remove actions mutate only temp durable memory fixtures in tests., memory and user targets map to the intended durable stores without renaming Goncho internally., Injection/exfiltration scans and redaction prevent unsafe content from entering provider-visible memory., Tool responses are bounded and match Hermes-compatible success/error shapes.
- Source refs: ../hermes-agent/tools/memory_tool.py:222-513, ../hermes-agent/tools/memory_tool.py:105-124, cmd/gormes/registry.go, internal/gonchotools/honcho_tools.go, internal/hermes/durable_user_context.go, internal/memory/
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Gateway stream/tool trace formatting fixture matrix

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Channel-neutral stream rendering has source-backed fixtures for Hermes/Sidon text deltas, tool progress, errors, and final answer separation, with Telegram MarkdownV2 escaping and compact labels for memory/search/read/patch/terminal/browser actions.
- Trust class: gateway, operator
- Ready when: Hermes/Sidon stream and tool-trace rendering examples are captured as source-backed fixture expectations., The renderer can be tested with pure gateway events and Telegram formatting fixtures without provider or live channel calls.
- Not ready when: -
- Degraded mode: Unknown tool events render as bounded generic tool_progress evidence instead of raw provider payloads or dropped traces.
- Fixture: `internal/gateway/render_test.go`
- Write scope: `internal/gateway/render.go`, `internal/gateway/render_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run 'Render\|FormatStream\|ToolTrace' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Renderer snapshot/table fixtures prove compact channel-neutral stream and tool trace parity.
- Acceptance: Renderer fixtures cover streaming text, final answer separation, provider errors, and tool progress., Telegram renderer escapes MarkdownV2 while preserving code blocks and compact labels., Memory/search/read/patch/terminal/browser traces match the parity matrix examples., No renderer emits tokens, raw credential values, or unbounded payloads.
- Source refs: ../hermes-agent/gateway/stream_consumer.py:482-508, ../hermes-agent/gateway/run.py, internal/gateway/render.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Gateway slash registry parity sweep (recognized-name expansion)

- Phase: 2 / 2.F.1
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: internal/gateway/commands.go::CommandRegistry recognizes every Hermes/Sidon command from cmd/gormes/hermes_cli_parity_test.go's manifest, even when the handler is not yet implemented, so unknown-command replies only fire on actual non-Hermes inputs. Each newly-recognized command lands with ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable and a friendly description, mirroring the existing /retry, /undo, /title, /branch, /compress pattern. Aliases (reset for new, set-home for sethome, gateway for platforms, etc.) resolve to the canonical command. Handler ports remain owned by the 49-file CLI tree port umbrella; this row only changes recognition.
- Trust class: gateway, operator
- Ready when: cmd/gormes/hermes_cli_parity_test.go::TestHermesCLIParityManifest is complete and enumerates the canonical Hermes command tree., internal/gateway/commands.go::buildRecognizedUnavailableSlashCommands already wires the unavailable-but-recognized fallback for 13 commands., Tests can drive the expanded set without touching handlers.
- Not ready when: The slice attempts to implement any handler beyond what the existing CommandRegistry pattern provides., The slice silently rewires existing aliases (reset, snap, fork, bg, q, tasks) — those remain on their current canonical commands., The slice changes the active-turn policy of any already-recognized command., The slice introduces new commands not present in Hermes COMMAND_REGISTRY.
- Degraded mode: When an expanded recognized command is sent and no handler exists, the bot replies '/<cmd> is recognized but unavailable' instead of 'unknown command', matching the existing buildRecognizedUnavailableSlashCommands pathway. No live runtime side effects.
- Fixture: `internal/gateway/commands_test.go + cmd/gormes/hermes_cli_parity_test.go`
- Write scope: `internal/gateway/commands.go`, `internal/gateway/commands_test.go`, `internal/gateway/active_turn_command_bypass_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`, `www.gormes.ai/internal/site/data/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run 'CommandRegistry\|ActiveTurn\|TelegramBotCommands\|Slack' -count=1`, `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway ./cmd/gormes -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Every Hermes command from the parity manifest resolves via ResolveCommand; unknown-command replies only fire for actual non-Hermes inputs.
- Acceptance: TestCommandRegistry_RecognizesAllHermesCommands: every command name and alias from cmd/gormes/hermes_cli_parity_test.go's Hermes inventory resolves via ResolveCommand., TestCommandRegistry_UnimplementedHermesCommandsAreUnavailable: every newly-added entry has ActiveTurnPolicy CommandActiveTurnPolicyUnavailable., TestCommandRegistry_UnknownCommandRepliesUnchanged: '/does-not-exist' still produces 'unknown command' (regression guard)., TestSlackSubcommandMap_StaysCanonical: SlackSubcommandMap output is byte-stable for existing entries while gaining the new ones., TestTelegramBotCommands_RegistryOrder: TelegramBotCommands() lists in registry order and includes the new entries., go test ./internal/gateway -run 'CommandRegistry\|ActiveTurn\|TelegramBotCommands' -count=1 passes.
- Source refs: ../hermes-agent/hermes_cli/commands.py:59-175, internal/gateway/commands.go:38-90, internal/gateway/commands.go:107-119, cmd/gormes/hermes_cli_parity_test.go
- Unblocks: 49-file CLI tree port
- Why now: Unblocks 49-file CLI tree port.

## 4. Stateful tool migration queue

- Phase: 5 / 5.A
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Contract: Gormes defines the migration queue and execution guard for stateful Hermes tools before exposing write-capable tools to the native loop: file, session, checkpoint, and process tools declare state domains, XDG roots, rollback/audit behavior, concurrency policy, and degraded evidence; the first implementation is a registry/read-model contract that lets builders add one stateful tool at a time without bypassing path isolation.
- Trust class: operator
- Ready when: Side-effect-light tool rows such as Todo and Debug helpers remain separately planned so this queue can focus only on write/process state contracts., Environment and path-denial contracts under internal/tools are green., The slice does not port the full file/process/checkpoint tools; it only freezes queue metadata and guard decisions for future rows.
- Not ready when: The slice implements write_file, patch, restore, terminal process spawning, live checkpoint restoration, or shell execution., A tool can declare write/process behavior without a state domain, rollback policy, and focused test command., Paths outside the injected Gormes roots can be accepted in tests.
- Degraded mode: Stateful tools without a validated queue entry return tool_state_contract_missing, tool_path_denied, tool_rollback_unavailable, or tool_concurrency_blocked evidence instead of mutating files, sessions, checkpoints, or processes.
- Fixture: `internal/tools/stateful_migration_queue_test.go`
- Write scope: `internal/tools/stateful_migration_queue.go`, `internal/tools/stateful_migration_queue_test.go`, `internal/tools/registry.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestStatefulToolMigrationQueue -count=1`, `go run ./cmd/progress validate`
- Done signal: Stateful tool queue fixtures prove domains, path isolation, rollback requirements, serialized write policy, and no hidden runtime mutations.
- Acceptance: TestStatefulToolMigrationQueueRegistersDomains proves file/session/checkpoint/process tool plans declare state domain, root policy, rollback policy, concurrency policy, and owner row., TestStatefulToolMigrationQueueRejectsMissingRollback proves write-capable tools cannot become selectable without rollback/audit evidence., TestStatefulToolMigrationQueuePathIsolation proves injected XDG roots are the only accepted mutation roots and traversal/absolute foreign paths return tool_path_denied., TestStatefulToolMigrationQueueSerializedWrites proves write-domain tools run through one deterministic queue while read-only tools can remain concurrent., TestStatefulToolMigrationQueueNoRuntimePort proves the queue contract does not execute shell/file/process mutations.
- Source refs: ../hermes-agent/tools/file_tools.py, ../hermes-agent/tools/terminal_tool.py, ../hermes-agent/tools/process_registry.py, ../hermes-agent/tools/checkpoint_manager.py, ../hermes-agent/tests/tools/test_file_tools.py, ../hermes-agent/tests/tools/test_watch_patterns.py, internal/tools/registry.go, internal/tools/environment_contract.go, internal/cli/pty_bridge.go, references/go-agent-os/engram/internal/mcp/write_queue.go, references/go-agent-os/axe/internal/artifact/tracker.go, references/go-agent-os/nanobot/pkg/tools/flows.go
- Unblocks: File write/patch tool port, Checkpoint restore tool port, Terminal process execution port
- Why now: Unblocks File write/patch tool port, Checkpoint restore tool port, Terminal process execution port.

## 5. Debug helpers

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Contract: Gormes ports Hermes DebugSession as shared tool debug infrastructure: tool-specific env vars enable a per-tool session ID, log entries remain in memory until explicit save, save writes deterministic JSON under an injected debug log directory, disabled debug mode is a no-op, get_session_info returns enabled/session/path/count evidence, and sensitive arguments are redacted before persistence.
- Trust class: operator, system
- Ready when: Native tool registry can share helper packages without enabling debug writes globally., Tests can inject env vars, fake clocks/UUIDs, and temp log directories., The slice defines redaction policy before MOA or web tools write debug logs.
- Not ready when: The slice writes logs when the tool-specific debug env var is unset/false., The slice stores raw tokens, API keys, bearer headers, cookies, file contents, or provider request bodies in debug JSON., The slice implements debug-share uploads, paste sweeps, support archives, or live doctor commands.
- Degraded mode: When debug mode is disabled or the log directory is unavailable, tool calls return debug_disabled or debug_log_unavailable evidence without hidden writes or tool execution failure.
- Fixture: `internal/tools/debug_helpers_test.go`
- Write scope: `internal/tools/debug_helpers.go`, `internal/tools/debug_helpers_test.go`, `internal/tools/registry.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestDebugSession -count=1`, `go run ./cmd/progress validate`
- Done signal: Debug helper fixtures prove disabled no-op behavior, enabled JSON save, redaction, session info, and save-failure degradation under temp directories only.
- Acceptance: TestDebugSessionDisabledNoops proves log/save/session-info do not create files when the env var is absent, false, or mixed-case false., TestDebugSessionEnabledWritesJSON proves true/True/TRUE enables logging, assigns a deterministic session ID, and saves a JSON file named <tool>_debug_<session>.json under the injected log dir., TestDebugSessionRedactsSensitiveArgs proves token, api_key, authorization, cookie, password, and raw provider body fields are redacted before save., TestDebugSessionInfoReportsPathAndCount proves get_session_info returns enabled, session_id, log_path, and call count without absolute home leakage beyond the injected temp path., TestDebugSessionSaveFailureDegrades proves directory/write failures return debug_log_unavailable evidence without panics.
- Source refs: ../hermes-agent/tools/debug_helpers.py:DebugSession, ../hermes-agent/tests/tools/test_debug_helpers.py, ../hermes-agent/tools/mixture_of_agents_tool.py:_debug, references/go-agent-os/engram/internal/mcp/activity.go, references/go-agent-os/axe/internal/artifact/tracker.go, references/go-agent-os/nanobot/pkg/tools/flows.go
- Unblocks: Multi-model coordination, Debug share paste sweep scheduler contract, Web/search tool debug logging
- Why now: Unblocks Multi-model coordination, Debug share paste sweep scheduler contract, Web/search tool debug logging.

## 6. Feishu transport/bootstrap layer

- Phase: 7 / 7.E
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Contract: Gormes adds a fakeable Feishu/Lark transport bootstrap boundary before live SDK binding: config resolves app credentials, connection_mode selects webhook vs websocket, webhook URL verification and signature checks are pure helpers, websocket event handlers register message/reaction/card/customized processors, inbound events queue until the adapter loop is ready, and rich-text/card send failures return typed SendResult evidence with redacted tokens.
- Trust class: gateway, operator, system
- Ready when: Shared gateway event and delivery contracts are validated enough to host a Feishu channel package., Tests can inject fake webhook bodies, fake websocket events, fake event handlers, and fake send clients., No live Feishu app credentials, SDK, websocket, or HTTP server is required.
- Not ready when: The slice imports the live Feishu SDK, opens network sockets, starts an HTTP server, or sends messages to Feishu., Inbound events arriving before the adapter loop is ready are dropped or spawn one drainer per event., Signature failures or send errors leak app_secret, verification_token, tenant token, or raw oversized provider bodies.
- Degraded mode: Feishu status reports feishu_config_missing, feishu_signature_invalid, feishu_loop_not_ready, feishu_ws_unavailable, or feishu_send_failed without opening live websocket or webhook network calls in unit tests.
- Fixture: `internal/channels/feishu/bootstrap_test.go`
- Write scope: `internal/channels/feishu/bootstrap.go`, `internal/channels/feishu/bootstrap_test.go`, `internal/channels/feishu/testdata/`, `internal/config/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels/feishu ./internal/config -run TestFeishuBootstrap -count=1`, `go run ./cmd/progress validate`
- Done signal: Feishu bootstrap fixtures prove connection mode, webhook verification/signature checks, handler registration, loop-not-ready queue drain, and redacted send failure evidence with fake clients only.
- Acceptance: TestFeishuBootstrapConnectionModeSelectsWebhookOrWebsocket proves config/env select the intended lifecycle without live clients., TestFeishuWebhookURLVerificationAndSignature proves challenge responses and invalid signatures are deterministic and redacted., TestFeishuEventHandlerRegistration proves message, reaction, card-action, chat-entered, and customized drive-comment handlers register on a fake builder., TestFeishuLoopNotReadyQueuesAndDrainsOnce proves events received before loop readiness are queued, drained in order, and not lost., TestFeishuRichTextSendFailureEvidence proves fake send failures return typed evidence and do not leak credentials.
- Source refs: ../hermes-agent/gateway/platforms/feishu.py:FeishuAdapter, ../hermes-agent/tests/gateway/test_feishu.py:test_build_event_handler_registers_reaction_and_card_processors, ../hermes-agent/tests/gateway/test_feishu.py:TestLoopNotReadyRace, ../hermes-agent/tests/gateway/test_feishu.py:test_webhook_signature, ../hermes-agent/tests/gateway/test_feishu.py:test_url_verification, ../hermes-agent/tests/gateway/test_feishu.py:test_normalize_interactive_card_preserves_title_body_and_actions, internal/channels/feishu/, internal/gateway/event.go, references/go-agent-os/trpc-agent-go/agent/callbacks.go, references/go-agent-os/engram/internal/mcp/activity.go
- Unblocks: Feishu drive-comment rule + pairing seam, Feishu drive-comment reply workflow, Feishu live SDK binding
- Why now: Unblocks Feishu drive-comment rule + pairing seam, Feishu drive-comment reply workflow, Feishu live SDK binding.

## 7. Gateway auto-title generation wiring

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: After an assistant turn finalizes (kernel.PhaseIdle), the gateway invokes session.PerformAutoTitle once per turn against the MetadataTitleStore adapter, using internal/hermes.GenerateTitle as the provider boundary, so untitled sessions get one auto-generated title from the first user+assistant exchange while manual titles, already-titled sessions, blank generator output, and provider failures all surface evidence without retry storms or transcript mutation.
- Trust class: gateway, operator, system
- Ready when: Session metadata manual-title protection flag row is complete and merged on dev (provides MetadataTitleStore adapter and TitleManuallySet flag)., internal/hermes.GenerateTitle and TitleModelFunc seam are validated on dev (4.F[0] complete)., Auxiliary-failure callback wiring (4.F[1]) is validated on dev so failure evidence routing is reusable., The slice runs with hermetic in-memory session.Map + a fake TitleModelFunc; no live provider, Telegram, or profile access required.
- Not ready when: The slice tries to capture user prompts or assistant replies from a future transcript store that does not yet exist on dev — must build []session.TitleTurn from in-process gateway state (last inbound text + PhaseIdle render frame text)., The slice changes session.Metadata schema, mergeMetadata semantics, or the SessionTitleStore interface signature., The slice introduces background goroutines or retry loops; PerformAutoTitle is one synchronous call per turn with bounded evidence.
- Degraded mode: If the configured TitleModelFunc is nil, PerformAutoTitle returns AutoTitleCodeProviderFailed evidence and the gateway routes it through the existing auxiliary-failure callback (4.F[1]) so operators see why no title was set. Provider errors and blank model output surface auto_title_provider_failed and auto_title_blank_result via the same channel without crashing the foreground turn.
- Fixture: `internal/gateway/auto_title_wiring_test.go`
- Write scope: `internal/gateway/auto_title_wiring.go`, `internal/gateway/auto_title_wiring_test.go`, `internal/gateway/manager.go`, `cmd/gormes/gateway.go`, `docs/content/building-gormes/architecture_plan/progress.json`, `www.gormes.ai/internal/site/data/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run 'AutoTitleWiring\|AutoTitle' -count=1`, `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway ./internal/session ./internal/hermes ./cmd/gormes -count=1`, `GOCACHE=/tmp/gormes-go-cache go test ./... -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Gateway dispatchFrame PhaseIdle case fires PerformAutoTitle once per turn against MetadataTitleStore + an injected TitleModelFunc, hermetic fixtures cover all evidence codes (generated, skipped_manual, skipped_titled, provider_failed, blank_result, missing_session), production cmd/gormes/gateway.go injects a real provider-backed TitleModelFunc, and the auxiliary-failure callback surfaces failure evidence without crashing the foreground turn.
- Acceptance: TestAutoTitleWiring_FirstUserAssistantPairTriggersGeneration drives a full turn through the gateway with an untitled session, a hermetic TitleModelFunc returning "Friendly Test Title", and asserts MetadataTitleStore.Title returns the generated title after the PhaseIdle frame is dispatched., TestAutoTitleWiring_ManualTitledSessionShortCircuits seeds a session with TitleManuallySet=true and asserts the TitleModelFunc is never invoked and the persisted title is unchanged., TestAutoTitleWiring_AlreadyTitledNonManualSessionShortCircuits seeds a session with Title set and TitleManuallySet=false, asserts AutoTitleCodeSkippedTitled evidence and no generator call., TestAutoTitleWiring_ProviderFailureRoutesAuxiliaryEvidence injects a TitleModelFunc that returns an error and asserts title_provider_failed evidence reaches the auxiliary-failure callback wired in 4.F[1]., TestAutoTitleWiring_BlankResultRoutesEvidence injects a TitleModelFunc returning "" and asserts title_blank_result evidence (no metadata write)., TestAutoTitleWiring_OneCallPerTurnNoDoubleFire dispatches two PhaseIdle frames in quick succession for the same session and asserts the generator is invoked at most once on the second turn (because the first turn wrote the title)., TestAutoTitleWiring_NilTitleModelFuncRecordsEvidence proves the manager defaults safely with no panic and routes AutoTitleCodeProviderFailed evidence when ManagerConfig.TitleModel is nil., Production wiring in cmd/gormes/gateway.go injects a real TitleModelFunc backed by the configured provider; a smoke fixture (build + ManagerConfig assertion only, no live LLM) proves the seam is non-nil in production builds.
- Source refs: internal/gateway/manager.go:864-938 (dispatchFrame, PhaseIdle case at 896), internal/session/auto_title.go:94 (PerformAutoTitle), internal/session/title_store.go (MetadataTitleStore, shipped in bffe3c518), internal/hermes/title_generator.go:91 (GenerateTitle, TitleModelFunc), internal/kernel/title_failure_test.go (auxiliary-failure routing, 4.F[1] complete), ../hermes-agent/agent/title_generator.py@4a2ee6c1:maybe_auto_title, ../hermes-agent/gateway/run.py (auto-title invocation site)
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Placeholder edit-failure fallback hardening

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: When the coalescer's edit path fails (Telegram returns an error mid-stream or at finalize), the gateway falls back to a plain Send of the final text exactly once, records redacted edit_failed_fallback evidence, and never produces duplicate final messages even under concurrent flushImmediate / flushImmediateFinal races.
- Trust class: gateway, operator
- Ready when: Existing coalescer tests on dev cover the happy path so regression is detectable.
- Not ready when: -
- Degraded mode: If both edit and Send fail, the gateway records send_final_failed evidence and the turn ends without a delivered final message rather than panicking or retrying in a hot loop.
- Fixture: `internal/gateway/coalesce_failure_test.go`
- Write scope: `internal/gateway/coalesce.go`, `internal/gateway/coalesce_failure_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run 'Coalescer\|Coalesce' -count=1`, `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`
- Done signal: Coalescer hardening fixtures prove edit-failure fallback to plain Send, finalize-race idempotency, both-failed evidence, and fresh-final-delete preservation.
- Acceptance: TestCoalescer_EditFailure_PlainSendFallback proves an edit error mid-stream causes one plain Send with the final text + edit_failed_fallback evidence., TestCoalescer_FinalizeRace_NoDuplicateMessage uses concurrent flushImmediate + flushImmediateFinal calls and asserts at most one final message reaches the channel., TestCoalescer_BothEditAndSendFail records send_final_failed evidence and does not panic., TestCoalescer_FreshFinalAfter_StillRespected proves the fresh-final-delete behavior is unchanged by the new fallback paths.
- Source refs: internal/gateway/coalesce.go (flushImmediate, flushImmediateFinal, editCoalescedMessage), internal/gateway/manager.go:864-938 (dispatchFrame finalize path)
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Telegram dynamic BotCommand menu wiring

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Telegram startup registers the core command set plus enabled plugin/skill slash commands through the existing dynamic BotCommand helper, respecting Telegram's 100-command cap and emitting hidden-count evidence.
- Trust class: gateway, system, operator
- Ready when: Core Telegram BotCommand registration is already present and the dynamic command helper surface is identified., The slice can use fake Bot API request capture and deterministic skill/plugin fixtures without reading live tokens.
- Not ready when: -
- Degraded mode: If dynamic command discovery fails, the bot registers the core menu and logs redacted dynamic_menu_unavailable evidence.
- Fixture: `internal/channels/telegram/bot_commands_dynamic_test.go`
- Write scope: `internal/channels/telegram/`, `internal/gateway/commands.go`, `internal/skills/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/channels/telegram ./internal/gateway ./internal/skills -run 'BotCommand\|TelegramCommands' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Telegram runtime registration uses the dynamic command helper with deterministic cap/fallback fixtures.
- Acceptance: Runtime `setMyCommands` receives core plus enabled dynamic commands in deterministic order., Aliases are omitted and command names are Telegram-safe., More than 100 commands are capped with hidden-count evidence., Core-only fallback remains available.
- Source refs: ../hermes-agent/hermes_cli/commands.py:558-589, ../hermes-agent/gateway/platforms/telegram.py:822-837, internal/gateway/commands.go:TelegramBotCommandsWith, internal/channels/telegram/bot.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. Active Hermes/Sidon profile context root resolver for live turns

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Live-turn context discovery resolves explicit Gormes overrides first, then active Hermes profile roots such as `HERMES_HOME=/home/xel/.hermes` + profile `mineru` or `sidon`, then workspace ancestor SOUL/USER/MEMORY files, without unit tests reading live profile state.
- Trust class: gateway, system
- Ready when: Current live-turn context discovery seams are identified in internal/hermes and internal/gateway., Temp HERMES_HOME/profile/workspace fixtures can prove resolution order without reading Juan's live profile.
- Not ready when: -
- Degraded mode: Missing profile files render missing-context evidence and continue the turn; unsafe paths are rejected with redacted evidence.
- Fixture: `internal/hermes/context_root_resolver_test.go + internal/gateway/live_turn_prompt_test.go`
- Write scope: `internal/hermes/`, `internal/gateway/live_turn_prompt.go`, `internal/gateway/live_turn_prompt_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/hermes ./internal/gateway -run 'ContextRoot\|SOUL\|DurableUserContext\|LiveTurn' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Hermetic profile-root resolver fixtures prove production discovery order without live profile reads.
- Acceptance: Temp-dir tests prove Gormes override wins over Hermes profile discovery., `HERMES_HOME=/tmp/.hermes` plus active profile resolves `/tmp/.hermes/profiles/<name>/SOUL.md` and memory files., Workspace ancestor SOUL/USER/MEMORY fallback is covered., Unit tests do not read `/home/xel/.gormes` or `/home/xel/.hermes`.
- Source refs: ../hermes-agent/hermes_constants.py, ../hermes-agent/run_agent.py, internal/hermes/context_files.go, internal/hermes/durable_user_context.go, internal/gateway/live_turn_prompt.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
