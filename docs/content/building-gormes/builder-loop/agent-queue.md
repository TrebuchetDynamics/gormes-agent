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

## 2. Gateway slash registry parity sweep (recognized-name expansion)

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

## 3. Stateful tool migration queue

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

## 4. Transcription tool contract

- Phase: 5 / 5.E
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Contract: Native STT/transcription tool helper validates local audio input and provider selection before gateway media hooks call it: files must exist, be regular files, use supported audio suffixes, and stay under configured max bytes; explicit provider selection among local, local_command, groq, openai, mistral, and xai never silently falls back; auto mode chooses Hermes order local, groq, openai, mistral, xai from injected availability; model defaults and overrides are normalized per provider; tool results return transcript/provider/model/language on success or typed redacted error evidence on failure.
- Trust class: operator, gateway, system
- Ready when: internal/tools has the shared tool descriptor/result surface and can host a pure transcription helper with injected provider clients., Tests can use t.TempDir files with small fake audio bytes and fake provider clients; no ffmpeg, faster-whisper, OpenAI, Groq, Mistral, xAI, managed gateway, or network call is required., Gateway voice/message attachment rows can remain blocked until this helper exposes a stable result envelope.
- Not ready when: The slice shells out to ffmpeg or local STT binaries, imports cloud SDKs, calls live provider APIs, downloads attachment media, or wires gateway channel handlers., Explicit provider errors fall back to another provider instead of returning a typed unavailable/error result., Provider errors expose API keys, managed-gateway tokens, raw HTTP response bodies, or local command output without redaction/truncation.
- Degraded mode: Disabled STT, missing files, directories, unsupported formats, oversized audio, missing provider credentials, local-command failures, and provider API failures return evidence codes such as stt_disabled, audio_not_found, audio_not_file, unsupported_audio_format, audio_too_large, stt_provider_unavailable, and stt_api_error with secret/redaction guards.
- Fixture: `internal/tools/transcription_tool_test.go`
- Write scope: `internal/tools/transcription_tool.go`, `internal/tools/transcription_tool_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run '^TestTranscription' -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Transcription fixtures prove audio validation, provider selection/no-fallback behavior, model normalization, redacted result envelope, and tool descriptor shape with fake providers only.
- Acceptance: TestTranscriptionValidateAudioInput proves missing files, directories, unsupported suffixes, and max-size violations return stable evidence and no provider is called., TestTranscriptionProviderSelection proves explicit local/local_command/groq/openai/mistral/xai selection is honored without fallback and auto mode follows local > groq > openai > mistral > xai availability., TestTranscriptionModelNormalization proves provider defaults and overrides normalize like Hermes, including local and local_command model aliases., TestTranscriptionResultEnvelope proves success includes transcript/provider/model/language and failures include redacted typed error evidence., TestTranscriptionToolDescriptor proves the tool schema exposes audio path, provider, model, language, and optional format fields without requiring gateway media plumbing.
- Source refs: ../hermes-agent/tools/transcription_tools.py:_load_stt_config, ../hermes-agent/tools/transcription_tools.py:is_stt_enabled, ../hermes-agent/tools/transcription_tools.py:_get_provider, ../hermes-agent/tools/transcription_tools.py:_validate_audio_file, ../hermes-agent/tools/transcription_tools.py:_normalize_local_model, ../hermes-agent/tools/transcription_tools.py:_normalize_local_command_model, ../hermes-agent/tools/transcription_tools.py:transcribe_audio, ../hermes-agent/tests/tools/test_transcription_tools.py, internal/tools/tool.go, references/go-agent-os/nanobot/pkg/tools/flows.go
- Unblocks: TTS synthesis + voice-mode state, Gateway media transcription hooks, Voice attachment handling for Signal and QQ Bot
- Why now: Unblocks TTS synthesis + voice-mode state, Gateway media transcription hooks, Voice attachment handling for Signal and QQ Bot.

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

## 7. Telegram reply_to_mode and reply-context parity

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Telegram replies honor Hermes-style reply mode configuration, fall back cleanly if a target message was deleted, and inbound Telegram reply text can be attached to session context without leaking raw slash commands to the model.
- Trust class: gateway, operator
- Ready when: Hermes Telegram reply-mode behavior has been mapped to Gormes Telegram adapter and gateway context seams., Fake Telegram client fixtures can cover outbound reply, fallback, and inbound reply-context behavior without live Telegram.
- Not ready when: -
- Degraded mode: If reply metadata is unavailable, Gormes sends a normal message with reply_context_missing evidence rather than failing the turn.
- Fixture: `internal/channels/telegram/reply_mode_test.go + internal/gateway/manager_test.go`
- Write scope: `internal/channels/telegram/`, `internal/gateway/`, `internal/config/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/channels/telegram ./internal/gateway -run 'Reply\|Telegram' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Telegram reply-mode fixtures prove quoting, fallback, and inbound reply context parity.
- Acceptance: Outbound placeholder, final, error, and `/status` messages obey reply mode., Deleted reply target errors fall back to non-reply send with evidence., Inbound replied-to message text reaches channel-neutral session context only when Hermes would include it., Unit tests use fake Telegram clients only.
- Source refs: ../hermes-agent/gateway/platforms/telegram.py:904-922, ../hermes-agent/gateway/platforms/telegram.py:1022-1032, ../hermes-agent/gateway/platforms/telegram.py:2935-2959, internal/channels/telegram/bot.go, internal/gateway/manager.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Telegram typing action + placeholder lifecycle parity

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Telegram turn progress matches Hermes/Sidon lifecycle: typing action or placeholder appears while work runs, stale hourglass messages are deleted or finalized, duplicate ghost replies collapse, and final answers remain readable.
- Trust class: gateway, operator
- Ready when: Hermes placeholder/typing lifecycle source references have been mapped to Gormes channel/gateway render seams., Fake Telegram lifecycle tests can simulate edit/delete failures and final-message cleanup without live Telegram.
- Not ready when: -
- Degraded mode: If Telegram edit/delete fails, Gormes sends one bounded final message and logs redacted cleanup evidence instead of leaving stale placeholders.
- Fixture: `internal/channels/telegram/placeholder_lifecycle_test.go + internal/gateway/coalesce_test.go`
- Write scope: `internal/channels/telegram/`, `internal/gateway/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/channels/telegram ./internal/gateway -run 'Placeholder\|Typing\|Coalesce\|Final' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Telegram placeholder/typing lifecycle fixtures prove no stale hourglass or duplicate final replies.
- Acceptance: Fake Telegram tests prove sendChatAction or placeholder behavior for long turns., Final answer cleanup deletes or edits the placeholder exactly once., Failure paths do not produce duplicate final messages., Fresh-final delete behavior remains covered.
- Source refs: ../hermes-agent/gateway/platforms/base.py:1718-1724, ../hermes-agent/gateway/platforms/base.py:1976-1986, ../hermes-agent/gateway/platforms/telegram.py:1909-1935, internal/gateway/coalesce.go, internal/channels/telegram/bot.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Gateway stream/tool trace formatting fixture matrix

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

## 10. Telegram dynamic BotCommand menu wiring

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

<!-- PROGRESS:END -->
