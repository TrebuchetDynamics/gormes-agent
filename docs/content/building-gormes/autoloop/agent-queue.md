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
autonomous implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared unattended-loop facts live in [Autoloop Handoff](../autoloop-handoff/):
the main entrypoint, orchestrator plan, candidate source, generated docs,
tests, and candidate policy. Keep those control-plane facts in
`meta.autoloop`, and keep row-specific execution facts in `progress.json`.

<!-- PROGRESS:START kind=agent-queue -->
## 1. DeepSeek/Kimi reasoning_content echo for tool-call replay

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Contract: Thinking-mode providers that require reasoning_content on assistant tool-call turns receive an echoed value during persistence and API replay
- Trust class: system
- Ready when: Shared provider continuation fixtures can serialize assistant tool-call messages and replay them without live provider credentials.
- Not ready when: The slice stores hidden reasoning text in ordinary assistant content or changes non-thinking providers' replay payloads.
- Degraded mode: Provider status explains when a thinking-mode provider requires reasoning echo padding and when a stored transcript was repaired for replay.
- Fixture: `internal/hermes/reasoning_content_echo_test.go`
- Write scope: `internal/hermes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -count=1`
- Done signal: Reasoning echo fixtures prove DeepSeek and Kimi tool-call replays include provider-required reasoning_content without mutating ordinary assistant content.
- Acceptance: DeepSeek is detected by provider name, model substring, or api.deepseek.com host and gets reasoning_content="" on assistant tool-call replay when no reasoning exists., Kimi/Moonshot detection keeps the existing reasoning_content padding behavior., Explicit reasoning_content or reasoning fields are preserved, while non-tool assistant turns and non-thinking providers are left untouched.
- Source refs: upstream Hermes d58b305a, ../hermes-agent/run_agent.py, ../hermes-agent/tests/run_agent/test_deepseek_reasoning_content_echo.py, docs/content/building-gormes/architecture_plan/phase-4-brain-transplant.md
- Unblocks: Cross-provider reasoning-tag sanitization, OpenRouter, Codex stream repair + tool-call leak sanitizer
- Why now: Unblocks Cross-provider reasoning-tag sanitization, OpenRouter, Codex stream repair + tool-call leak sanitizer.

## 2. Codex Responses pure conversion harness

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Contract: OpenAI Responses request/response conversion for Codex-compatible providers without live OAuth
- Trust class: system
- Ready when: Shared provider transcript and tool-call continuation fixtures are complete.
- Not ready when: The slice performs OAuth/device login, imports ~/.codex/auth.json, or opens a live Responses request.
- Degraded mode: Provider status reports Codex unavailable until Responses conversion fixtures pass and auth wiring is configured.
- Fixture: `internal/hermes/codex_responses_adapter_test.go`
- Write scope: `internal/hermes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -count=1`
- Done signal: Codex Responses fixtures convert chat input, tool schemas, output items, usage, and tool calls without live credentials.
- Acceptance: Chat messages and multimodal content parts convert to Responses input items deterministically., Function tools convert to Responses function-tool schemas with deterministic call IDs., Responses output items normalize back to shared provider events, messages, usage, and tool calls.
- Source refs: ../hermes-agent/agent/codex_responses_adapter.py, ../hermes-agent/tests/run_agent/test_run_agent_codex_responses.py, ../hermes-agent/tests/run_agent/test_tool_call_args_sanitizer.py, docs/content/building-gormes/architecture_plan/phase-4-brain-transplant.md
- Unblocks: Codex stream repair + tool-call leak sanitizer, Codex OAuth state + stale-token relogin
- Why now: Unblocks Codex stream repair + tool-call leak sanitizer, Codex OAuth state + stale-token relogin.

## 3. Tool-call argument repair + schema sanitizer

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Contract: Provider tool-call arguments are repaired or rejected against available tool schemas before execution
- Trust class: system, child-agent
- Ready when: Shared provider tool-call continuation fixtures are complete and tool descriptors expose required argument schemas.
- Not ready when: The slice silently invents missing required arguments or changes tool executor behavior instead of validating provider output.
- Degraded mode: Tool execution status reports schema-repair failures before a malformed provider call reaches the executor.
- Fixture: `internal/hermes/tool_call_argument_repair_test.go`
- Write scope: `internal/hermes/`, `internal/tools/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes ./internal/tools -count=1`
- Done signal: Tool-call repair fixtures prove malformed arguments are repaired or rejected before execution using the advertised schema.
- Acceptance: Malformed JSON argument fragments from streamed tool calls are repaired only when the repair is deterministic., Impossible repairs return a provider/tool-call error before execution., Repair decisions use the current advertised tool schema so disabled or unavailable tools cannot be hallucinated into execution.
- Source refs: ../hermes-agent/tests/run_agent/test_repair_tool_call_arguments.py, ../hermes-agent/tests/run_agent/test_streaming_tool_call_repair.py, ../hermes-agent/tests/run_agent/test_tool_call_args_sanitizer.py, ../hermes-agent/tools/schema_sanitizer.py
- Unblocks: Codex stream repair + tool-call leak sanitizer, OpenRouter, Bedrock stream event decoding (SSE fixtures)
- Why now: Unblocks Codex stream repair + tool-call leak sanitizer, OpenRouter, Bedrock stream event decoding (SSE fixtures).

## 4. BlueBubbles iMessage bubble formatting parity

- Phase: 7 / 7.E
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes
- Trust class: gateway, system
- Ready when: The first-pass BlueBubbles adapter already owns Send, markdown stripping, cached GUID resolution, and home-channel fallback in internal/channels/bluebubbles.
- Not ready when: The slice attempts to add live BlueBubbles HTTP/webhook registration, attachment download, reactions, typing indicators, or edit-message support.
- Degraded mode: BlueBubbles remains a usable first-pass adapter, but long replies may still arrive as one stripped text send until paragraph splitting and suffix-free chunking are fixture-locked.
- Fixture: `internal/channels/bluebubbles/bot_test.go`
- Write scope: `internal/channels/bluebubbles/bot.go`, `internal/channels/bluebubbles/bot_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels/bluebubbles -count=1`
- Done signal: BlueBubbles adapter tests prove paragraph-to-bubble sends, suffix-free chunking, and no edit/placeholder capability.
- Acceptance: Send splits blank-line-separated paragraphs into separate SendText calls while preserving existing chat GUID resolution and home-channel fallback., Long paragraph chunks omit `(n/m)` pagination suffixes and concatenate back to the stripped original text., Bot does not implement gateway.MessageEditor or gateway.PlaceholderCapable, preserving non-editable iMessage semantics.
- Source refs: ../hermes-agent/gateway/platforms/bluebubbles.py@f731c2c2, ../hermes-agent/tests/gateway/test_bluebubbles.py@f731c2c2, internal/channels/bluebubbles/bot.go, internal/gateway/channel.go
- Unblocks: BlueBubbles iMessage session-context prompt guidance
- Why now: Unblocks BlueBubbles iMessage session-context prompt guidance.

<!-- PROGRESS:END -->
