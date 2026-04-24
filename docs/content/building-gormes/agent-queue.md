---
title: "Agent Queue"
weight: 34
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
autonomous implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared unattended-loop facts live in [Autoloop Handoff](./autoloop-handoff/):
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

## 2. BlueBubbles iMessage bubble formatting parity

- Phase: 2 / 2.B.10
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

## 3. GBrain minion-orchestrator routing policy

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

## 4. Interrupted-turn memory sync suppression

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

## 5. Honcho-compatible scope/source tool schema

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

## 6. Bedrock Converse payload mapping (no AWS SDK)

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

## 7. Codex Responses pure conversion harness

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

## 8. Tool-call argument repair + schema sanitizer

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

## 9. Skill preprocessing + dynamic slash commands

- Phase: 5 / 5.F
- Owner: `skills`
- Size: `small`
- Status: `planned`
- Contract: Skill content preprocessing and skill-backed slash commands are deterministic, disabled-skill aware, and prompt-safe
- Trust class: operator, gateway, system
- Ready when: Phase 2.G parser/store and inactive candidate flow are complete.
- Not ready when: Inline shell preprocessing can execute during prompt assembly or disabled skills remain invokable through slash commands.
- Degraded mode: Skill status reports disabled, missing-prerequisite, or preprocessing-failed skills without injecting them into prompts.
- Fixture: `internal/skills/preprocessing_commands_test.go`
- Write scope: `internal/skills/`, `internal/gateway/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/skills ./internal/gateway -count=1`
- Done signal: Skill preprocessing and slash-command fixtures prove disabled/incompatible skills do not enter prompt or command surfaces.
- Acceptance: Template variable preprocessing is deterministic and fixture-covered., Inline shell preprocessing is disabled by default and bounded when explicitly enabled., Skill slash commands skip disabled/incompatible skills and build stable user-message content.
- Source refs: ../hermes-agent/agent/skill_preprocessing.py, ../hermes-agent/agent/skill_commands.py, ../hermes-agent/tools/skills_tool.py, ../hermes-agent/tests/tools/test_skills_tool.py, ../hermes-agent/tests/agent/test_skill_commands.py
- Unblocks: Toolset-aware skills prompt snapshot, TUI + Telegram browsing
- Why now: Unblocks Toolset-aware skills prompt snapshot, TUI + Telegram browsing.

## 10. PTY bridge protocol adapter

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Contract: Dashboard/TUI PTY sessions expose bounded read, write, resize, close, and unavailable-state behavior through a testable adapter
- Trust class: operator
- Ready when: Deterministic CLI helper ports are understood and PTY behavior can be isolated from the web dashboard transport.
- Not ready when: The slice starts the web dashboard, opens network listeners, or binds to a real TUI process in unit tests.
- Degraded mode: Dashboard or CLI status reports PTY unavailable instead of falling back to unsafe shell execution.
- Fixture: `internal/cli/pty_bridge_test.go`
- Write scope: `internal/cli/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cli -count=1`
- Done signal: PTY bridge fixtures prove read/write/resize/close/unavailable behavior without network or live dashboard dependencies.
- Acceptance: Reads are bounded by timeout and chunk size., Writes and resize messages validate input before reaching the PTY., Unsupported platforms return PtyUnavailable-style errors without starting a shell.
- Source refs: ../hermes-agent/hermes_cli/pty_bridge.py, ../hermes-agent/tests/hermes_cli/test_pty_bridge.py, ../hermes-agent/hermes_cli/web_server.py
- Unblocks: SSE streaming to Bubble Tea TUI, Dashboard PTY chat sidecar contract
- Why now: Unblocks SSE streaming to Bubble Tea TUI, Dashboard PTY chat sidecar contract.

<!-- PROGRESS:END -->
