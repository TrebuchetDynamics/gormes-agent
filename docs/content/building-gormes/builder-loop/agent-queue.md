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
## 1. Stateful tool migration queue

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

## 2. Browser action contract + event transcript

- Phase: 5 / 5.C
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Contract: Gormes freezes the native browser tool contract before binding Chromedp or Rod: action schema, page-state transcript events, screenshot/result envelope, console/content-none guards, private-URL safety handoff, oversized artifact pointer behavior, and unavailable-backend errors are represented as pure Go types and fixtures.
- Trust class: operator, child-agent, system
- Ready when: Browser hybrid private-URL local sidecar routing and Browser SSRF quoted-false guard are validated on main., The worker can build pure action/result/transcript fixtures with fake screenshots and fake page state; no Chromedp, Rod, Browserbase, Firecrawl, Camofox, DNS, or live browser dependency is required.
- Not ready when: The slice starts a browser, follows redirects, opens network connections, chooses Chromedp versus Rod, or implements provider bridges., The slice bypasses the existing private-host and quoted-false SSRF guard helpers.
- Degraded mode: Browser status returns browser_backend_unavailable, browser_action_invalid, browser_result_truncated, or private_url_local_sidecar evidence instead of starting a browser, contacting cloud providers, or dumping screenshots/transcripts into model context.
- Fixture: `internal/tools/browser_contract_test.go`
- Write scope: `internal/tools/browser_contract.go`, `internal/tools/browser_contract_test.go`, `internal/tools/browser_hybrid_routing.go`, `internal/tools/browser_ssrf_guard.go`, `internal/tools/result_budget.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run 'TestBrowserContract\|TestBrowserResultEnvelope\|TestBrowserTranscript' -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Browser contract fixtures prove action validation, transcript/result envelopes, artifact-pointer behavior, unavailable backend errors, and reuse of private-URL guard decisions without starting a browser.
- Acceptance: Action-schema fixtures validate navigation, click/type, screenshot, content extraction, wait, and invalid-action errors without a browser backend., Transcript fixtures preserve page URL/title/text/console/error/screenshot metadata and handle None/empty content without panics., Screenshot and oversized text results route through a bounded artifact pointer shape instead of embedding unbounded bytes in tool output., Unavailable backend and private URL sidecar decisions produce typed degraded evidence and reuse the existing SSRF guard contracts.
- Source refs: ../hermes-agent/tools/browser_tool.py, ../hermes-agent/tests/tools/test_browser_content_none_guard.py, ../hermes-agent/tests/tools/test_browser_console.py, ../hermes-agent/tests/tools/test_browser_hardening.py, ../hermes-agent/tests/tools/test_browser_hybrid_routing.py, references/go-agent-os/nanobot/pkg/agents/truncate.go, references/go-agent-os/axe/internal/artifact/tracker.go, internal/tools/browser_hybrid_routing.go, internal/tools/browser_ssrf_guard.go, internal/tools/result_budget.go
- Unblocks: Chromedp, Rod, Browser provider bridge + Firecrawl fallback
- Why now: Unblocks Chromedp, Rod, Browser provider bridge + Firecrawl fallback.

## 3. Todo

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Contract: Gormes ports Hermes todo_tool as a small stateful native tool: items validate id/content/status, duplicate IDs keep the last entry while preserving list order, replace vs merge semantics match Hermes, only pending/in_progress items are injected into prompts, summary counts include pending/in_progress/completed/cancelled, and storage is deterministic under an injected per-session root.
- Trust class: operator
- Ready when: Native tool registry can register a schema-backed pure/stateful tool with fake store injection., The slice uses temp roots in tests and does not depend on the full file-tools port., Stateful tool migration queue is either complete or this row limits persistence to a small per-session store with explicit path tests.
- Not ready when: The slice implements unrelated task planning UX, file_tools, process tools, or cross-session syncing., Completed/cancelled todos are injected back into prompts., Duplicate ID behavior keeps the first entry or reorders the priority list differently from Hermes fixtures.
- Degraded mode: Invalid status/content, missing store, corrupt JSON, or state-root denial returns todo_invalid_args, todo_store_unavailable, or todo_store_corrupt evidence without touching file-tools or global session state.
- Fixture: `internal/tools/todo_tool_test.go`
- Write scope: `internal/tools/todo_tool.go`, `internal/tools/todo_tool_test.go`, `internal/tools/registry.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestTodoTool -count=1`, `go run ./cmd/progress validate`
- Done signal: Todo fixtures prove schema validation, replace/merge, duplicate last-wins, active-only injection, summary counts, and corrupt-store degradation without file-tools dependency.
- Acceptance: TestTodoToolValidateAndDedupe proves status normalization, empty-content rejection, duplicate ID last-wins behavior, and stable priority order., TestTodoToolReplaceAndMerge proves replace overwrites the store while merge updates existing items and appends new ones according to Hermes fixtures., TestTodoToolFormatForInjectionOnlyActive proves pending and in_progress items inject with Hermes markers while completed/cancelled items are omitted., TestTodoToolSummaryCounts proves JSON output contains todo list plus pending/in_progress/completed/cancelled counts., TestTodoToolStoreCorruptionDegrades proves malformed persisted JSON returns safe evidence and does not panic or delete the file.
- Source refs: ../hermes-agent/tools/todo_tool.py:TodoStore, ../hermes-agent/tools/todo_tool.py:todo_tool, ../hermes-agent/tools/todo_tool.py:TODO_SCHEMA, ../hermes-agent/tests/tools/test_todo_tool.py, internal/tools/registry.go, references/go-agent-os/engram/internal/mcp/write_queue.go, references/go-agent-os/nanobot/pkg/tools/flows.go
- Unblocks: TUI todo panel
- Why now: Unblocks TUI todo panel.

## 4. Debug helpers

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

## 5. Feishu transport/bootstrap layer

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

## 6. Tool-result pruning + protected head/tail summary

- Phase: 4 / 4.B
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Contract: Gormes freezes the pure context-compression pruning pass before kernel mutation: protect system and first-turn head messages, choose the recent tail by token budget with at least three messages, keep assistant tool_calls paired with their tool results, prune old oversized tool result content without cutting tool-call arguments or JSON payloads, and emit summary-prefix-compatible replacement messages.
- Trust class: operator, system
- Ready when: ContextEngine interface, compression token-budget sizing, auxiliary headroom, provider-aware cap, and single-prompt threshold rows are validated on main., The worker can test the pruning pass with synthetic message arrays, fake token counters, and existing tool-result budget helpers; no summarizer, provider call, or kernel history mutation is required.
- Not ready when: The slice calls a summarizer, changes provider routing, edits live kernel history, ports manual compression feedback, or rewrites persisted transcripts instead of freezing the pure pruning transform., The implementation trims assistant tool-call arguments as text, emits partial JSON, or leaves orphaned tool_result messages without their assistant call.
- Degraded mode: Context status reports pruning_skipped, prune_budget_unavailable, or invalid_tool_pair evidence instead of silently truncating JSON arguments, dropping required tool results, or mutating live history.
- Fixture: `internal/hermes/context_compressor_pruning_test.go`
- Write scope: `internal/hermes/context_compressor_pruning.go`, `internal/hermes/context_compressor_pruning_test.go`, `internal/hermes/context_compressor_budget.go`, `internal/tools/result_budget.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run TestContextCompressorPruning -count=1`, `go test ./internal/hermes ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Context-pruning fixtures prove token-budget tail selection, tool-call/result pairing, non-truncated JSON arguments, oversized-result pruning, and degraded evidence without provider calls.
- Acceptance: Fixtures prove oversized historical tool results are pruned while recent tail messages are selected by token budget and still preserve at least three messages., Assistant tool_calls and tool result messages remain paired after pruning; no tool result starts the tail without its matching assistant call., Tool-call argument JSON is never substring-truncated; invalid or unparseable argument boundaries cause visible degraded evidence instead of mutation., Summary replacement content uses the existing Hermes summary prefix rules and does not create impossible consecutive-role collisions.
- Source refs: ../hermes-agent/agent/context_compressor.py:_prune_old_tool_results, ../hermes-agent/agent/context_compressor.py:_find_tail_cut_by_tokens, ../hermes-agent/tests/agent/test_context_compressor.py:TestContextCompressorTokenBudget, ../hermes-agent/tests/agent/test_context_compressor.py:test_summarization_does_not_split_tool_call_pairs, references/go-agent-os/nanobot/pkg/agents/truncate.go, references/go-agent-os/nanobot/pkg/agents/tokencount.go, references/go-agent-os/axe/internal/budget/budget.go, internal/hermes/context_compressor_budget.go, internal/tools/result_budget.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. Prompt-cache capability guard

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

## 8. Clarify

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Contract: Gormes ports Hermes clarify as a schema-validated, interruptible user-reply tool: required question text, up to four trimmed choices, platform-added Other behavior, callback/resume routing for gateway and TUI, deterministic unavailable output in non-interactive cron/oneshot contexts, and one-shot resume-token cleanup after the next user reply.
- Trust class: operator, gateway, child-agent, system
- Ready when: Tool descriptor parity manifest, TUI clarify panel renderer, and oneshot noninteractive clarify policy are validated on main., The worker can test schema/callback/resume behavior with fake platform callbacks and fake session state; no live Telegram, TUI event loop, or stdin interaction is required.
- Not ready when: The slice implements only TUI rendering without tool execution/resume state, or only schema validation without gateway/TUI callback routing., The slice blocks cron or oneshot waiting for user input, or persists a pending reply route that is not cleared after one resume.
- Degraded mode: Clarify returns clarify_invalid_args, clarify_unavailable, clarify_timeout, or clarify_route_missing evidence instead of blocking cron/oneshot turns, reading stdin from a noninteractive context, or leaking a pending route into the wrong session.
- Fixture: `internal/tools/clarify_tool_test.go; internal/gateway/clarify_resume_test.go`
- Write scope: `internal/tools/clarify_tool.go`, `internal/tools/clarify_tool_test.go`, `internal/gateway/clarify_resume.go`, `internal/gateway/clarify_resume_test.go`, `internal/tui/`, `cmd/gormes/oneshot_safety_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestClarifyTool -count=1`, `go test ./internal/gateway -run TestClarifyResume -count=1`, `go test ./cmd/gormes -run TestOneshotClarify -count=1`, `go run ./cmd/progress validate`
- Done signal: Clarify fixtures prove Hermes schema validation, platform callback output, gateway/TUI one-shot resume routing, and noninteractive unavailable/timeout evidence without live UI.
- Acceptance: Tool fixtures match Hermes validation: empty questions error, choices must be lists, whitespace choices are stripped, non-string choices stringify, and more than four choices are trimmed., Callback fixtures return question, choices_offered, and stripped user_response for open-ended and multiple-choice prompts., Gateway/TUI resume fixtures persist a one-shot route for the awaiting session and clear it after the next user reply., Cron/oneshot fixtures return clarify_unavailable or clarify_timeout evidence and never wait for interactive input.
- Source refs: ../hermes-agent/tools/clarify_tool.py:clarify_tool, ../hermes-agent/tools/clarify_tool.py:CLARIFY_SCHEMA, ../hermes-agent/tests/tools/test_clarify_tool.py, ../hermes-agent/cli.py:_clarify_callback, ../hermes-agent/gateway/run.py:clarify callback handling, references/go-agent-os/trpc-agent-go/agent/await_user_reply.go, cmd/gormes/oneshot_safety_test.go, internal/tui/hermes_panels.go, internal/tools/testdata/upstream_tool_parity_manifest.json
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
