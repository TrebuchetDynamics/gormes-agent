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
## 1. Non-editable gateway progress/commentary send fallback

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: Channels without placeholder/edit capabilities receive progress-safe interim or final assistant messages through the plain Send path without EditMessage calls
- Trust class: gateway, system
- Ready when: A fake channel can implement only gateway.Channel while the manager render path is exercised from an originating inbound event.
- Not ready when: The slice adds platform-specific BlueBubbles logic to gateway.Manager instead of testing capability-based behavior shared by all non-editable channels.
- Degraded mode: Non-editable channels continue to receive final responses, but quick commentary/interim updates may be suppressed until the send-fallback fixture proves no edit path is attempted.
- Fixture: `internal/gateway/manager_test.go`
- Write scope: `internal/gateway/manager.go`, `internal/gateway/manager_test.go`, `internal/gateway/fake_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway -run NonEditable -count=1`, `go test ./internal/gateway -run Manager_Outbound -count=1`
- Done signal: Gateway manager tests cover both editable streaming and Channel-only send fallback without platform-specific branches.
- Acceptance: A Channel-only fake receives terminal assistant output through Send with the original chat target., The same fixture proves no SendPlaceholder or EditMessage method is required for non-editable channels., Editable channel streaming/coalescing tests keep their existing placeholder/edit behavior.
- Source refs: ../hermes-agent/tests/gateway/test_run_progress_topics.py@f731c2c2, internal/gateway/manager.go, internal/gateway/channel.go, internal/gateway/fake_test.go
- Unblocks: BlueBubbles iMessage session-context prompt guidance
- Why now: Unblocks BlueBubbles iMessage session-context prompt guidance.

## 2. GBrain minion-orchestrator routing policy

- Phase: 2 / 2.E.1
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Durable-job routing separates deterministic restart-survivable work from live LLM subagents, following GBrain's unified minion-orchestrator skill while keeping Gormes Go-native subagent APIs
- Trust class: operator, child-agent, system
- Ready when: Phase 2.E.0 and 2.E.1 subagent runtime slices are complete and append-only run logs exist for child executions.
- Not ready when: The slice ports GBrain's Postgres/PGLite queue, shell-job executor, pause/resume runtime, or supervisor instead of only fixture-locking routing decisions and trust boundaries.
- Degraded mode: Until the policy lands, Gormes exposes in-memory subagents plus append-only run logs only; durable minion routing is documented as unavailable in status/doctor surfaces.
- Fixture: `internal/subagent/minion_policy_test.go`
- Write scope: `internal/subagent/`, `internal/cron/`, `internal/config/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/subagent ./internal/cron ./internal/config -count=1`
- Done signal: Routing policy tests prove which work enters durable orchestration, which trust classes may submit it, and that existing delegate_task names remain stable.
- Acceptance: Policy fixtures classify deterministic shell/cron-like work separately from judgment-heavy LLM subagents., Child-agent trust cannot submit privileged deterministic shell jobs through the durable-job route., Existing delegate_task and subagent manager APIs remain Go-native and are not renamed to Minions.
- Source refs: ../gbrain/skills/minion-orchestrator/SKILL.md, ../gbrain/skills/conventions/subagent-routing.md, docs/content/upstream-gbrain/architecture.md, internal/subagent/manager.go, internal/cron/executor.go
- Unblocks: Durable subagent/job ledger
- Why now: Unblocks Durable subagent/job ledger.

## 3. Interrupted-turn memory sync suppression

- Phase: 3 / 3.E.7
- Owner: `memory`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Interrupted or cancelled turns cannot flush partial observations into GONCHO or external Honcho-compatible memory
- Trust class: system
- Ready when: The turn-finalization path can tell a normal completion from an interrupt, cancellation, or client disconnect.
- Not ready when: The slice rewrites extraction, recall, or provider-plugin storage instead of only gating sync/finalization on interrupted turns.
- Degraded mode: Memory status reports skipped or interrupted sync attempts without promoting partial facts to recall.
- Fixture: `internal/memory/interrupted_sync_test.go`
- Write scope: `internal/kernel/`, `internal/memory/`, `internal/goncho/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/kernel ./internal/memory ./internal/goncho -count=1`
- Done signal: Interrupted-turn fixtures prove partial memory observations are skipped while completed turns still sync.
- Acceptance: Interrupted turns record a skipped-sync reason and do not create new GONCHO conclusions., Completed turns still sync/extract normally., Operator status can distinguish skipped interrupted sync from extractor failures.
- Source refs: ../hermes-agent/agent/memory_manager.py, ../hermes-agent/tests/run_agent/test_memory_sync_interrupted.py, docs/content/building-gormes/architecture_plan/phase-3-memory.md
- Unblocks: Honcho host integration compatibility fixtures, Cross-chat operator evidence
- Why now: Unblocks Honcho host integration compatibility fixtures, Cross-chat operator evidence.

## 4. Honcho-compatible scope/source tool schema

- Phase: 3 / 3.E.7
- Owner: `memory`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Honcho-compatible tool schemas expose GONCHO scope and source allowlist controls without renaming public tools
- Trust class: operator, system
- Ready when: internal/goncho SearchParams and ContextParams already accept scope and sources fields.
- Not ready when: The slice renames public honcho_* tools, changes internal goncho storage, or bundles deny-path/operator evidence work.
- Degraded mode: Memory status and tool schema evidence show when user-scope or source-filtered recall is unavailable.
- Fixture: `internal/tools/honcho_tools_test.go`
- Write scope: `internal/tools/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools ./internal/goncho -count=1`
- Done signal: Honcho-compatible tool schema tests prove scope and sources are optional, discoverable, and routed through existing GONCHO params.
- Acceptance: honcho_search and honcho_context JSON Schemas include optional scope and sources fields., scope and sources are not required and omitted calls preserve same-chat default behavior., The internal implementation package and tables remain named goncho while external tool names remain honcho_*.
- Source refs: docs/content/upstream-hermes/gormes-takeaways.md, docs/content/building-gormes/architecture_plan/phase-3-memory.md, ../honcho/docs/v3/guides/integrations/hermes.mdx
- Unblocks: Cross-chat deny-path fixtures, Honcho host integration compatibility fixtures
- Why now: Unblocks Cross-chat deny-path fixtures, Honcho host integration compatibility fixtures.

## 5. parent_session_id lineage for compression splits

- Phase: 3 / 3.E.8
- Owner: `memory`
- Size: `small`
- Status: `planned`
- Contract: Session metadata records compression/fork lineage and can resolve the live descendant without rewriting ancestor history
- Trust class: operator, gateway, system
- Ready when: internal/session.Metadata already persists source, chat_id, user_id, and updated_at on bbolt-backed sessions.
- Not ready when: The slice implements compression itself or rewrites existing session IDs instead of adding append-only lineage metadata.
- Degraded mode: Session mirrors and status show missing, orphaned, or looped lineage instead of silently resuming stale roots.
- Fixture: `internal/session/lineage_test.go`
- Write scope: `internal/session/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/session -count=1`
- Done signal: Session lineage fixtures prove append-only parent metadata, loop rejection, and operator-visible orphan status without implementing compression.
- Acceptance: session.Metadata carries parent_session_id and lineage_kind while root sessions remain parentless by default., Self-parent and trivial parent loops are rejected in BoltMap and MemMap tests., SessionIndexMirror or an equivalent read model exposes parent/child/orphan lineage for operator audit.
- Source refs: ../hermes-agent/hermes_state.py, ../hermes-agent/docs/developer-guide/session-storage.md, ../hermes-agent/tests/gateway/test_resume_command.py, docs/content/building-gormes/architecture_plan/phase-3-memory.md
- Unblocks: Gateway resume follows compression continuation, Lineage-aware source-filtered search hits
- Why now: Unblocks Gateway resume follows compression continuation, Lineage-aware source-filtered search hits.

## 6. DeepSeek/Kimi reasoning_content echo for tool-call replay

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

## 7. Bedrock Converse payload mapping (no AWS SDK)

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Contract: Pure Bedrock Converse request mapping over the shared provider message/tool contract
- Trust class: system
- Ready when: Provider interface + stream fixture harness and tool-call continuation contract are complete.
- Not ready when: The slice imports AWS SDK clients or signs live requests before pure request-body mapping is fixture-locked.
- Degraded mode: Provider status reports Bedrock as unavailable until request mapping fixtures pass and credential wiring lands.
- Fixture: `internal/hermes/bedrock_converse_mapping_test.go`
- Write scope: `internal/hermes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -count=1`
- Done signal: Bedrock request-body golden fixtures prove Converse mapping without AWS credentials or SDK clients.
- Acceptance: System, user, assistant, and tool-result messages map to Bedrock Converse roles and content blocks., Tool definitions map to Bedrock toolSpec inputSchema without dropping required fields., Golden request fixtures pin max_tokens, temperature, cache/reasoning passthrough, and empty-content placeholders.
- Source refs: ../hermes-agent/agent/bedrock_adapter.py, ../hermes-agent/agent/transports/bedrock.py, ../hermes-agent/tests/agent/test_bedrock_adapter.py, docs/content/building-gormes/architecture_plan/phase-4-brain-transplant.md
- Unblocks: Bedrock stream event decoding (SSE fixtures)
- Why now: Unblocks Bedrock stream event decoding (SSE fixtures).

## 8. Codex Responses pure conversion harness

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

## 9. Tool-call argument repair + schema sanitizer

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

## 10. Provider-enforced context-length resolver

- Phase: 4 / 4.D
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Contract: Displayed and budgeted context windows prefer provider-enforced limits over raw models.dev metadata
- Trust class: operator, system
- Ready when: Provider status and model metadata can be tested as pure functions without live provider credentials.
- Not ready when: The slice implements routing/fallback policy or pulls live models.dev/network data during unit tests.
- Degraded mode: Model status reports whether the context length came from provider-specific caps, models.dev fallback, or an unknown model.
- Fixture: `internal/hermes/model_context_resolver_test.go`
- Write scope: `internal/hermes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -count=1`
- Done signal: Context resolver fixtures prove provider caps, models.dev fallback, and unknown-model reporting are deterministic without network calls.
- Acceptance: openai-codex gpt-5.5 displays and budgets the provider cap (272000 tokens) instead of the raw models.dev 1050000-token window., Provider-specific caps for Codex OAuth, Copilot, and Nous win over model-info fallbacks when present., Unknown resolver failures fall back to fixture model metadata and report unknown when both sources are empty.
- Source refs: upstream Hermes 05d8f110, ../hermes-agent/hermes_cli/model_switch.py, ../hermes-agent/cli.py, ../hermes-agent/gateway/run.py, ../hermes-agent/tests/hermes_cli/test_model_switch_context_display.py
- Unblocks: Compression token-budget trigger + summary sizing, Routing policy and fallback selector
- Why now: Unblocks Compression token-budget trigger + summary sizing, Routing policy and fallback selector.

<!-- PROGRESS:END -->
