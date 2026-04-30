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

## 2. Feishu transport/bootstrap layer

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

## 3. Goncho memory provider lifecycle adapter

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

## 4. Prompt-cache capability guard

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

## 5. Telegram browser artifact rendering

- Phase: 5 / 5.C
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Telegram rendering for browser results is mobile-readable and Hermes/Sidon-compatible: screenshot/artifact pointers, DOM excerpts, console errors, and browser progress traces are MarkdownV2-safe, bounded, reply-threaded, and separate from final answers.
- Trust class: gateway, system, operator
- Ready when: The browser artifact result envelope and Telegram renderer seams are identified., Fake Telegram/render fixtures can prove screenshot/link/console presentation without live Telegram, Chrome, or cloud browser services.
- Not ready when: -
- Degraded mode: If Telegram media delivery is unavailable, Gormes sends a text artifact pointer with browser_artifact_text_fallback evidence.
- Fixture: `internal/gateway/telegram_browser_render_test.go + internal/tools/browser_artifact_test.go`
- Write scope: `internal/gateway/render.go`, `internal/gateway/telegram_browser_render_test.go`, `internal/channels/telegram/`, `internal/tools/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway ./internal/channels/telegram ./internal/tools -run 'Telegram.*Browser\|Browser.*Artifact\|Render' -count=1`, `GOCACHE=/tmp/gormes-go-cache go run ./cmd/progress validate`, `git diff --check`
- Done signal: Telegram browser rendering fixtures prove bounded, escaped, reply-safe artifact output.
- Acceptance: Telegram renderer escapes browser artifact text and code blocks correctly., Screenshots/artifact pointers are shown without raw bytes or local secret paths., Browser progress/tool traces remain distinct from final answer text., Reply metadata is preserved when browser output is sent as part of a Telegram turn.
- Source refs: ../hermes-agent/gateway/platforms/telegram.py, ../hermes-agent/tools/browser_tool.py, internal/gateway/render.go, internal/channels/telegram/bot.go, internal/tools/browser_harness_tools.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Clarify

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
