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
## 1. Gateway slash registry parity sweep (recognized-name expansion)

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

## 2. Stateful tool migration queue

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

## 3. Debug helpers

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

## 4. Feishu transport/bootstrap layer

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

## 5. Gateway active-turn policy manifest closeout

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Compare the full Hermes gateway slash-command policy set against Gormes CommandDef policies and close remaining bypass, reject, drain, unavailable, and model-leak differences with a manifest fixture.
- Trust class: gateway, operator
- Ready when: Hermes slash-command policy sources and Gormes CommandDef registry are available for table-driven comparison., Gateway manager tests can prove interception decisions with fake provider captures and no live channels.
- Not ready when: -
- Degraded mode: Unknown policies default to reject-with-guidance during active turns and never leak slash text to the provider.
- Fixture: `internal/gateway/active_turn_command_bypass_test.go`
- Write scope: `internal/gateway/commands.go`, `internal/gateway/commands_test.go`, `internal/gateway/active_turn_command_bypass_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run 'ActiveTurn\|Command\|Slash' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Active-turn command policy manifest fixtures prove bypass/reject/drain parity and no slash leakage.
- Acceptance: A manifest table lists every Hermes slash command policy and the Gormes decision., Known control/info commands bypass active turns., Mutating commands reject or drain according to the manifest., Provider capture proves slash text never reaches the model when intercepted.
- Source refs: ../hermes-agent/hermes_cli/commands.py:267-290, ../hermes-agent/gateway/run.py:2950-3225, internal/gateway/commands.go, internal/gateway/manager.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Gateway conversational session metadata refresh

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Normal conversational Telegram turns create or refresh session metadata with Hermes-compatible session ID, created time, last activity time, connected platform evidence, and title eligibility before `/status` renders it.
- Trust class: gateway, operator
- Ready when: Gormes session metadata and status rendering seams are identified., Fake conversational gateway turns can create/update metadata without live Telegram or provider calls.
- Not ready when: -
- Degraded mode: If metadata persistence fails, `/status` reports session_metadata_unavailable instead of fabricating timestamps.
- Fixture: `internal/gateway/session_metadata_test.go`
- Write scope: `internal/gateway/`, `internal/session/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway ./internal/session -run 'SessionMetadata\|Status\|Telegram' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Conversational turn fixtures prove created/updated/session/platform metadata refresh semantics for `/status`.
- Acceptance: A normal user turn creates metadata with stable session ID, CreatedAt, UpdatedAt, and connected platform., A later turn updates Last Activity without changing CreatedAt., `/status` reflects the same metadata without model submission., Legacy `telegram:<chat_id>` identifiers are mapped or explicitly reported.
- Source refs: ../hermes-agent/gateway/session.py, ../hermes-agent/gateway/run.py:4646-4680, internal/gateway/status_command.go, internal/session/
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. Gateway session token accounting parity

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Accumulate per-session provider usage into session metadata so `/status` reports Hermes-compatible token totals rather than only the last usage frame.
- Trust class: gateway, system, operator
- Ready when: Provider usage surfaces and status rendering seams are identified., Fake provider usage frames can exercise accumulation and missing-usage behavior without live provider calls.
- Not ready when: -
- Degraded mode: If provider usage is missing, status renders zero or unknown with usage_missing evidence and never guesses token counts.
- Fixture: `internal/gateway/token_accounting_test.go`
- Write scope: `internal/gateway/`, `internal/session/`, `internal/hermes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway ./internal/session ./internal/hermes -run 'Token\|Usage\|Status' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Session token accounting fixtures prove accumulated usage appears in `/status` with redacted evidence.
- Acceptance: Two fake provider turns with usage values accumulate into one session total., `/status` renders the accumulated total., Missing usage does not erase existing totals., Usage evidence never includes raw provider payloads or credentials.
- Source refs: ../hermes-agent/gateway/run.py:4674, internal/gateway/status_command.go, internal/gateway/usage_command.go, internal/hermes/provider_transport.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Goncho memory provider lifecycle adapter

- Phase: 3 / 3.F
- Owner: `memory`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Create a native Goncho memory-provider lifecycle adapter covering initialize, prefetch, sync turn, pre-compress contribution, memory-write mirror, delegation, and shutdown evidence so Gormes matches Hermes MemoryManager behavior without a hosted Honcho dependency.
- Trust class: system
- Ready when: Hermes MemoryManager lifecycle hooks and Gormes Goncho/kernel seams have been source-mapped., Fake provider and temp SQLite/local stores can exercise lifecycle behavior without hosted Honcho or provider network.
- Not ready when: -
- Degraded mode: Unavailable providers return memory_provider_unavailable evidence and leave ordinary turn execution intact.
- Fixture: `internal/memory/provider_lifecycle_test.go + internal/goncho`
- Write scope: `internal/memory/`, `internal/goncho/`, `internal/kernel/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/memory ./internal/goncho ./internal/kernel -run 'MemoryProviderLifecycle\|ShutdownMemory\|PreCompress\|Goncho' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Goncho memory-provider lifecycle fixtures prove initialize, prefetch, sync, pre-compress, mirror, delegation, and shutdown behavior.
- Acceptance: Lifecycle tests prove initialize and prefetch run once per session/provider., Turn sync mirrors user/assistant exchanges into Goncho with redacted evidence., Pre-compress contributions are bounded and ordered before compression., Shutdown drains or reports pending memory work without blocking process exit., Tests use fake providers and SQLite temp stores only.
- Source refs: ../hermes-agent/agent/memory_manager.py:97, ../hermes-agent/agent/memory_manager.py:178, ../hermes-agent/agent/memory_manager.py:210, ../hermes-agent/agent/memory_manager.py:296, ../hermes-agent/agent/memory_manager.py:315, ../hermes-agent/agent/memory_manager.py:331, internal/memory/, internal/goncho/, internal/kernel/
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Prompt-cache capability guard

- Phase: 4 / 4.H
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Contract: Gormes applies Hermes prompt-cache markers only when provider, endpoint, API mode, and model policy allow them: native Anthropic uses native layout, OpenRouter Claude uses envelope layout, third-party Anthropic Claude gateways cache conservatively, Qwen on opencode/opencode-go/Alibaba gets envelope markers, and OpenAI-wire custom providers without an allow rule strip cache_control visibly.
- Trust class: operator, system
- Ready when: Provider status already exposes a prompt-cache capability slot and unsupported OpenAI-compatible cache_control stripping is validated., The worker can test policy decisions and message rewrites with synthetic provider/baseURL/apiMode/model tuples; no live provider, token store, or network call is required.
- Not ready when: The slice sends cache_control to every OpenAI-compatible provider, changes retry/rate-limit behavior, or relies on live provider probes., The slice only changes status text without proving request mapping for native, envelope, and stripped layouts.
- Degraded mode: Provider status reports prompt_cache_supported, prompt_cache_stripped, prompt_cache_provider_unknown, or prompt_cache_policy_unavailable instead of leaking unsupported cache_control fields into strict providers.
- Fixture: `internal/hermes/prompt_cache_policy_test.go`
- Write scope: `internal/hermes/prompt_cache_policy.go`, `internal/hermes/prompt_cache_policy_test.go`, `internal/hermes/status.go`, `internal/hermes/anthropic_client.go`, `internal/hermes/provider_status_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run 'TestPromptCachePolicy\|TestApplyPromptCacheControl' -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/progress validate`
- Done signal: Prompt-cache fixtures prove provider policy, native/envelope/stripped layouts, four-breakpoint rewrite behavior, and visible unsupported-provider status without live probes.
- Acceptance: Policy fixtures match Hermes for native Anthropic, Anthropic-host aliases, OpenRouter Claude, third-party Anthropic Claude gateways, OpenAI-wire custom Claude names, and Qwen opencode/opencode-go/Alibaba cases., Message rewrite fixtures deep-copy inputs, place at most four breakpoints, mark system plus last three non-system messages, preserve 1h TTL, and handle native Anthropic tool-role markers., OpenAI-wire providers without an allow rule strip cache_control before request serialization and expose a visible degraded capability reason., Provider status and request bodies agree: a supported policy serializes cache markers and an unsupported policy omits them.
- Source refs: ../hermes-agent/agent/prompt_caching.py:apply_anthropic_cache_control, ../hermes-agent/run_agent.py:_anthropic_prompt_cache_policy, ../hermes-agent/tests/agent/test_prompt_caching.py, ../hermes-agent/tests/run_agent/test_anthropic_prompt_cache_policy.py, references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md#quick-lookup-problem--donor-file, internal/hermes/status.go, internal/hermes/client.go, internal/hermes/anthropic_client.go, internal/hermes/provider_status_test.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. Browser artifact and console render contract

- Phase: 5 / 5.C
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Browser tool results expose bounded screenshot paths, DOM snapshots, console logs, page errors, and artifact metadata in a channel-neutral envelope; renderers show safe previews without raw bytes, base64 blobs, private URLs, CDP secrets, or unbounded provider output.
- Trust class: system, gateway, operator
- Ready when: Hermes browser tool artifact/result behavior and Gormes browser harness result envelopes are source-mapped., Pure internal/tools fixtures can prove artifact budgeting, console/error metadata, and redacted evidence without starting Chrome or cloud browser services.
- Not ready when: -
- Degraded mode: Oversized or missing artifacts render browser_artifact_unavailable or browser_artifact_truncated evidence and keep the turn readable.
- Fixture: `internal/tools/browser_artifact_test.go + internal/gateway/render_test.go`
- Write scope: `internal/tools/browser_contract.go`, `internal/tools/browser_harness_tools.go`, `internal/tools/result_budget.go`, `internal/tools/browser_artifact_test.go`, `internal/gateway/render.go`, `internal/gateway/render_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/tools ./internal/gateway -run 'Browser.*Artifact\|Browser.*Console\|Render' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Browser artifact fixtures prove bounded screenshot/snapshot/console/error envelopes and channel-safe rendering.
- Acceptance: Tool fixtures cover screenshot path, DOM snapshot excerpt, console logs, and page errors., Artifact budgets truncate large snapshots with evidence., Gateway renderer emits channel-safe artifact summaries without raw image bytes or base64., Secrets, cookies, CDP URLs, and private URLs are redacted.
- Source refs: ../hermes-agent/tools/browser_camofox.py:300, ../hermes-agent/tools/browser_camofox.py:493, ../hermes-agent/tools/browser_camofox.py:583, ../hermes-agent/tools/browser_tool.py, internal/tools/browser_contract.go, internal/tools/result_budget.go, internal/gateway/render.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
