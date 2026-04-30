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
## 1. Feishu transport/bootstrap layer

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

## 2. Clarify

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
