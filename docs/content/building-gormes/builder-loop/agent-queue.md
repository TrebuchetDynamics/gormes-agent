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
## 1. Telegram production live-turn provider payload golden

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: The actual `gormes telegram` entrypoint is exercised with fake Telegram ingress and a fake provider capture so the final provider ChatRequest for `What's your name?` contains Gormes SOUL identity, USER.md, MEMORY.md, AGENTS/project context, timestamp, model, provider, Telegram/session metadata, skill/tool guidance, and the user message before provider execution.
- Trust class: gateway, system, operator
- Ready when: Live-turn metadata production wiring (cmd/gormes -> Manager seams) is validated., The test can run with temp profile/memory dirs, a fake Telegram client, and a fake provider; no live Telegram, provider network, or Python Hermes runtime is required.
- Not ready when: The slice only tests helper output instead of the production `gormes telegram` manager construction path., The slice rewrites provider output text or replaces `ChatGPT` after the provider returns., The slice reads Juan's live ~/.gormes, ~/.hermes, provider tokens, or Telegram token.
- Degraded mode: If a context file is missing, the captured request still carries redacted missing-context evidence and never uses post-provider string replacement to force identity.
- Fixture: `cmd/gormes/telegram_test.go + internal/gateway/live_turn_prompt_test.go`
- Write scope: `cmd/gormes/telegram.go`, `cmd/gormes/telegram_test.go`, `internal/gateway/live_turn_prompt_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`, `www.gormes.ai/internal/site/data/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./cmd/gormes ./internal/gateway -run 'Telegram.*ProviderPayload\|LiveTurn' -count=1`, `GOCACHE=/tmp/gormes-go-cache go test ./cmd/gormes ./internal/gateway ./internal/hermes ./internal/runtime -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: The production Telegram entrypoint has a fake-provider golden proving identity and context reach the final provider payload before execution.
- Acceptance: A failing test first proves the production `gormes telegram` path can capture the final provider request before execution., The captured request contains Gormes identity/SOUL text before the user message., The captured request contains USER.md and MEMORY.md durable context from temp fixtures., The captured request contains AGENTS/project context, Telegram/session context, timestamp, model, and provider metadata., The test proves no output postprocessing is used to change provider identity text.
- Source refs: ../hermes-agent/run_agent.py:3667-3779, ../hermes-agent/agent/prompt_builder.py, ../hermes-agent/gateway/platforms/telegram.py, cmd/gormes/telegram.go:telegramManagerConfig, internal/gateway/live_turn_prompt.go, internal/gateway/live_turn_prompt_test.go
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Telegram /status Hermes-format closeout

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: `/status` renders Hermes-compatible field order and labels, always includes a real session title when a title exists or can be generated, quotes the triggering Telegram message, and remains a gateway command that never enters the provider/model path.
- Trust class: gateway, operator
- Ready when: Hermes/Sidon `/status` field order and Telegram reply behavior have been source-audited against gateway references., Existing status/session metadata seams can be exercised with fake gateway and Telegram fixtures only.
- Not ready when: -
- Degraded mode: If title generation cannot run, status returns structured title_unavailable evidence instead of silently omitting the Title field or rendering a hardcoded fake title.
- Fixture: `internal/gateway/status_command_test.go + internal/channels/telegram/bot_test.go`
- Write scope: `internal/gateway/status_command.go`, `internal/gateway/status_command_test.go`, `internal/channels/telegram/bot.go`, `internal/channels/telegram/bot_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway ./internal/channels/telegram -run 'Status\|Title\|Telegram\|Reply' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Status fixtures prove Hermes-compatible fields, title visibility, reply quoting, and provider bypass.
- Acceptance: `/status` output includes Session ID, Title, Created, Last Activity, Tokens, Agent Running, and Connected Platforms., Telegram status replies set ReplyToMessageID for the triggering `/status` message., A fake provider/model path capture proves `/status` is not submitted as user text., Formatting differences from Hermes are either removed or explicitly documented in the parity matrix.
- Source refs: ../hermes-agent/gateway/run.py:4646-4680, ../hermes-agent/hermes_cli/commands.py:267-290, internal/gateway/status_command.go, internal/channels/telegram/bot.go
- Why now: P0 handoff; needs contract proof before closeout.

## 3. Gateway /title manual session title command

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: Implement Hermes-compatible `/title` handling in the gateway: `/title` shows the current title, `/title <name>` sanitizes and stores a manual title, manual titles are not overwritten by auto-title, invalid titles return operator guidance, and the command never reaches the provider.
- Trust class: gateway, operator
- Ready when: Hermes `/title` command semantics and Gormes session metadata storage have been source-audited., The slice can run with fake gateway/session stores and no provider, Telegram, or live profile access.
- Not ready when: -
- Degraded mode: If the metadata store is unavailable, `/title` returns title_store_unavailable evidence and does not mutate transcript content.
- Fixture: `internal/gateway/title_command_test.go`
- Write scope: `internal/gateway/commands.go`, `internal/gateway/manager.go`, `internal/gateway/title_command.go`, `internal/gateway/title_command_test.go`, `internal/gateway/status_command_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run 'Title\|Status\|Command' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: `/title` set/show/error fixtures pass and `/status` preserves manual titles.
- Acceptance: `/title` on a session with a title returns that title., `/title Friendly Greeting with Juan` stores a manual title and `/status` renders it., Manual titles are not overwritten by auto-title., Empty, overlong, or unsafe titles return guidance and no mutation., A fake provider capture proves `/title` is not sent to the model.
- Source refs: ../hermes-agent/gateway/run.py:6697-6743, ../hermes-agent/tests/gateway/test_title_command.py, internal/gateway/commands.go, internal/session/auto_title.go
- Why now: P0 handoff; needs contract proof before closeout.

## 4. Telegram MarkdownV2 parse-mode rendering closeout

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: internal/channels/telegram/bot.go::Send and SendReply set msgCfg.ParseMode = tgbotapi.ModeMarkdownV2 so that the MarkdownV2-escaped output produced by internal/gateway/render.go::FormatStreamTelegram, FormatFinalTelegram, and FormatErrorTelegram renders as bold/italic/code/spoiler/strike on Telegram clients instead of literal backslashes. A parse-failure fallback resends the same body in plain text via msgCfg.ParseMode = '' when Telegram rejects the MarkdownV2 payload, mirroring Hermes' behavior at gateway/platforms/telegram.py:998-1003. Edit messages (EditMessage) and placeholder sends (SendPlaceholder/SendReplyPlaceholder) honor the same parse-mode policy.
- Trust class: gateway, operator
- Ready when: internal/gateway/render.go::FormatStreamTelegram, FormatFinalTelegram, FormatErrorTelegram already MarkdownV2-escape outbound text (verified by render_test.go)., internal/channels/telegram/bot.go uses tgbotapi which exposes tgbotapi.ModeMarkdownV2 as a parse mode constant., Tests can construct MessageConfig values and inspect ParseMode without contacting Telegram.
- Not ready when: The slice modifies internal/gateway/render.go to change escape behavior — render output stays byte-identical to today's tests., The slice removes the MarkdownV2 escape path or routes any string through Telegram unsanitized., The slice introduces parse-mode logic in any non-Telegram channel (Slack/Discord/etc.)., The slice attempts to render unescaped tool output, leaks secrets, or relaxes the existing FormatErrorTelegram sanitization.
- Degraded mode: If Telegram rejects MarkdownV2 (Bot API error 'can't parse entities'), the bot retries the same body once with parse_mode unset and emits redacted telemetry evidence. If the retry also fails, the original error is surfaced via HookOnError and the channel state machine treats the platform as failed for that turn.
- Fixture: `internal/channels/telegram/bot_test.go`
- Write scope: `internal/channels/telegram/bot.go`, `internal/channels/telegram/bot_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`, `www.gormes.ai/internal/site/data/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/channels/telegram -run 'MarkdownV2\|ParseMode\|Send\|Reply\|EditMessage' -count=1`, `GOCACHE=/tmp/gormes-go-cache go test ./internal/channels/telegram ./internal/gateway -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Bot fixtures prove ParseMode reaches the MessageConfig wire on Send/SendReply/EditMessage; parse-error fallback retries with empty ParseMode without leaking secrets; existing render tests stay green.
- Acceptance: TestBot_SendSetsMarkdownV2ParseMode: a fake telegram client captures the MessageConfig produced by Bot.Send and asserts ParseMode == 'MarkdownV2'., TestBot_SendReplySetsMarkdownV2ParseMode: same as above but for SendReply, and ReplyToMessageID stays correct., TestBot_EditMessageSetsMarkdownV2ParseMode: EditMessageText carries ParseMode 'MarkdownV2'., TestBot_SendPlaintextFallbackOnParseError: when the fake client returns a Telegram parse-error, the bot retries once with empty ParseMode and the body is byte-identical to the first attempt's text., TestBot_SendPlaintextFallbackEvidence: a redaction-safe telemetry hook fires HookOnError-equivalent evidence on the parse-error branch without leaking raw token strings., Render tests under internal/gateway/render_test.go continue to pass unchanged (escape behavior preserved).
- Source refs: ../hermes-agent/gateway/platforms/telegram.py:91-122, ../hermes-agent/gateway/platforms/telegram.py:973-1003, internal/gateway/render.go:48-81, internal/channels/telegram/bot.go:127-179
- Unblocks: Telegram /status Hermes-format closeout
- Why now: P0 handoff; needs contract proof before closeout.

## 5. Hermes memory tool over Goncho/local durable store

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

## 6. Gateway slash registry parity sweep (recognized-name expansion)

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

## 7. Stateful tool migration queue

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

## 8. Transcription tool contract

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

## 9. Debug helpers

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

## 10. Feishu transport/bootstrap layer

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

<!-- PROGRESS:END -->
