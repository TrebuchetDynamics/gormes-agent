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

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Extension Lifecycle Hook System

- Phase: 5 / 5.I
- Owner: `tools`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: Port agent-zero extension lifecycle hook system: register extensions at 8+ lifecycle points (agent_init, monologue_start/end, message_loop_start/end, before_main_llm_call, prompt_before/after, stream_chunk, tool_before/after, context_deleted). Extension chain executes in registration order with per-extension timeout and panic isolation.
- Trust class: operator, system
- Ready when: Kernel state machine transitions are well-defined., Plugin registry supports lifecycle callback registration.
- Not ready when: The row introduces Python dependency., The row adds hooks without timeout/panic recovery per extension.
- Degraded mode: Extension load failure, timeout, or panic reports per-extension status with degraded extension skipped. No single extension failure blocks the agent turn.
- Fixture: `internal/kernel/extensions_test.go`
- Write scope: `internal/kernel/extensions.go`, `internal/kernel/extensions_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/kernel -run TestExtensions -count=1`, `go run ./cmd/progress validate`
- Done signal: Extension lifecycle hook system ships with 8+ hook points and per-extension error isolation.
- Acceptance: Extensions register for monologue_start, message_loop_start, prompt_before, prompt_after, stream_chunk, tool_before, tool_after hooks., Extension chain executes in registration order., Extension timeout or panic does not crash agent turn., gormes extensions list shows registered extensions with hook points.
- Source refs: agent-zero helpers/extension.py (@extensible decorator), agent-zero agent.py (hook points), docs/content/building-gormes/agent-zero-feature-analysis.md, internal/kernel/kernel.go
- Unblocks: Plugin ecosystem, Skill injection pipeline
- Why now: Unblocks Plugin ecosystem, Skill injection pipeline.

## 2. Channels Capabilities Introspection

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Port OpenClaw's channels capabilities: gormes channels capabilities shows provider capabilities (intents/scopes + supported features) for each configured channel. Enables operators to understand what each channel adapter supports before configuring it.
- Trust class: operator
- Ready when: Channel adapters expose capability metadata., CLI channel command routing is defined.
- Not ready when: The row hardcodes per-channel capabilities., The row requires live channel connections for capability discovery.
- Degraded mode: Unconfigured channel, missing adapter, or capability query failure reports per-channel status with 'unknown' capability rather than crashing.
- Fixture: `internal/channels/capabilities_test.go`
- Write scope: `internal/channels/capabilities.go`, `internal/channels/capabilities_test.go`, `cmd/gormes/channels_capabilities.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels -run TestCapabilities -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes channels capabilities ships for all configured channel adapters.
- Acceptance: gormes channels capabilities --channel telegram shows intents, scopes, and supported features., Output includes media support, command support, and format limitations per channel., Unconfigured channels show 'not configured' status.
- Source refs: openclaw channels capabilities CLI surface, internal/channels/* (adapter implementations), docs/content/building-gormes/openclaw-platform-parity-audit.md
- Unblocks: Channel configuration UX
- Why now: Unblocks Channel configuration UX.

## 3. Prompt Fragment Include System

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Port agent-zero prompt fragment system: prompts stored as fragments with {{include filename.md}} directives, priority search order (agent profile > user > plugin > default), {{include original}} chains through hierarchy, variables substituted at render time.
- Trust class: operator, system
- Ready when: Extension lifecycle hooks (5.I) provide prompt_before/after hook points., Skill system supports profile-level prompt overrides.
- Not ready when: The row hardcodes prompt content in Go strings., The row bypasses existing prompt builder contract.
- Degraded mode: Missing fragment, circular include, or render failure reports prompt_fragment_error with chain trace.
- Fixture: `internal/hermes/prompt_fragments_test.go`
- Write scope: `internal/hermes/prompt_fragments.go`, `internal/hermes/prompt_fragments_test.go`, `prompts/*.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run TestPromptFragments -count=1`, `go run ./cmd/progress validate`
- Done signal: Prompt fragment system ships with {{include}}, {{include original}}, and variable substitution.
- Acceptance: {{include agent.system.main.role.md}} resolves through priority search., {{include original}} chains through profile > user > default hierarchy., Circular includes detected and reported., Fragment cache invalidates on file change.
- Source refs: agent-zero prompts/ (72 fragment files), agent-zero agent.py:prepare_prompt, docs/content/building-gormes/agent-zero-feature-analysis.md
- Unblocks: Agent profile customization, Plugin prompt injection
- Why now: Unblocks Agent profile customization, Plugin prompt injection.

## 4. Telegram forum thread fallback + send retry safety

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Telegram outbound send/typing behavior preserves Hermes forum and retry safety: forum General-topic inbound messages without message_thread_id retain synthetic thread context `1`; outbound sends to General omit message_thread_id=1; send and typing retry without message_thread_id when Telegram returns BadRequest 'message thread not found'; non-thread BadRequest errors fail immediately; transient NetworkError sends retry with bounded attempts; TimedOut sends do not retry to avoid duplicate visible messages; RetryAfter sleeps/backoffs then retries; and once a chunk clears an invalid thread ID, later chunks in the same long response use no thread ID directly.
- Trust class: operator, gateway, system
- Ready when: Telegram reply_to_mode and reply-context parity plus channel directory thread target fixtures are complete, so the row can focus on Telegram message_thread_id send semantics., Tests can use fake Telegram clients that return typed errors and capture Chattable fields; no live Telegram Bot API call is required.
- Not ready when: The slice changes generic gateway delivery routing, reconnect polling lifecycle, or media-group batching., TimedOut errors are retried, non-thread BadRequest is retried, General-topic sends include message_thread_id=1, or invalid thread fallback only fixes final text while dropping typing/tool-progress messages., The active Go Telegram SDK and gateway manager seams cannot expose inbound message_thread_id/is_forum or pass thread metadata through final send/typing paths without first adding a thread-aware channel delivery seam.
- Degraded mode: If thread fallback or retry classification is unavailable, Gormes reports telegram_thread_fallback_unavailable or telegram_send_retry_unavailable evidence and avoids duplicate retries for timeouts rather than silently dropping all streaming/tool-progress messages.
- Fixture: `internal/channels/telegram/thread_fallback_test.go; internal/channels/telegram/send_retry_test.go`
- Write scope: `internal/channels/telegram/bot.go`, `internal/channels/telegram/client.go`, `internal/channels/telegram/thread_fallback_test.go`, `internal/channels/telegram/send_retry_test.go`, `internal/gateway/delivery.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels/telegram -run 'TestTelegram(ThreadFallback\|SendRetry\|TypingRetry)' -count=1`, `go test ./internal/channels/telegram ./internal/gateway -run 'Telegram\|Delivery' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Telegram fake-client fixtures prove General-topic thread preservation/omission, thread-not-found fallback for send and typing, duplicate-safe timeout handling, retry-after behavior, and chunk-level invalid-thread clearing.
- Acceptance: Thread fallback fixtures prove General-topic inbound keeps thread context `1`, General-topic outbound omits thread ID, thread-not-found send/typing calls retry once without thread ID, and non-thread BadRequest fails immediately., Retry fixtures prove transient NetworkError retries are bounded, TimedOut does not retry, RetryAfter retry succeeds, and long chunked messages clear invalid thread metadata after the first failed chunk., Existing Telegram parse-mode, reply-mode, and media-send fixtures remain green.
- Source refs: ../hermes-agent/gateway/platforms/telegram.py@b816fd4e2:send, ../hermes-agent/gateway/platforms/telegram.py@b816fd4e2:send_typing, ../hermes-agent/tests/gateway/test_telegram_thread_fallback.py@b816fd4e2, ../hermes-agent/tests/gateway/test_telegram_reply_mode.py@b816fd4e2, internal/channels/telegram/bot.go, internal/channels/telegram/client.go, internal/gateway/delivery.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Sandbox isolation depth selection

- Phase: 5 / 5.U
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P3`
- Contract: Operator can select sandbox isolation depth: process-level (fast, weaker isolation), container-level (Docker/gVisor, balanced), or VM-level (Firecracker, strongest isolation). Default is process-level with transactional rollback.
- Trust class: operator
- Ready when: Transactional executor exists (5.U row 2)
- Not ready when: No sandbox backend available
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/tools/isolation_depth.go`, `internal/tools/isolation_depth_test.go`
- Test commands: `go test ./internal/tools -run TestIsolationDepth -count=1`
- Done signal: Isolation depth tests prove all three levels selectable and process-level works without Docker
- Acceptance: Process-level isolation is the default and requires zero setup, Docker/gVisor isolation selectable via config, Firecracker VM isolation selectable via config, Isolation depth is per-session configurable, Deeper isolation correctly fails if backend not available
- Source refs: docs/content/papers/safety-and-deployment.md, OpenSandbox (github.com/alibaba/OpenSandbox), internal/tools/sandbox.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Behavioral pattern extraction from session logs

- Phase: 6 / 6.K
- Owner: `orchestrator`
- Size: `large`
- Status: `planned`
- Priority: `P3`
- Contract: Mine session logs and tool execution audits for behavioral patterns: which tool sequences succeed vs fail, which reasoning patterns precede good outcomes, which response styles correlate with user satisfaction. Patterns feed into the self-evolution loop as candidate mutations.
- Trust class: operator
- Ready when: Session logs are structured and queryable, Tool execution audit log exists (Phase 3.E.2)
- Not ready when: No structured session data available, Tool audit log not yet implemented
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/hermes/pattern_extractor.go`, `internal/hermes/pattern_extractor_test.go`
- Test commands: `go test ./internal/hermes -run TestPatternExtractor -count=1`
- Done signal: Pattern extractor tests prove successful and failed patterns are correctly identified from log data
- Acceptance: Pattern extractor identifies tool sequences with >80% success rate, Identifies tool sequences with <30% success rate (anti-patterns), Extracts reasoning patterns preceding successful tool calls, Patterns stored in Goncho as structured behavioral knowledge, Pattern extraction is offline (does not run during agent turns)
- Source refs: docs/content/papers/agentic-os-design.md, Hermes Agent GEPA engine, Generative Agents reflection mechanism (Park et al. 2023), internal/goncho/extractor.go, internal/hermes/turn.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. Skill code execution runtime

- Phase: 6 / 6.L
- Owner: `skills`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: Skills are not just markdown instructions — they contain executable code that can be run in a sandboxed environment. This mirrors Voyager's code-as-action pattern: skills are validated, sandboxed, and can be composed by the agent at runtime.
- Trust class: operator, system
- Ready when: Skill loader parses structured skill files, Sandbox execution exists for tool calls
- Not ready when: Skill files are plain text only (no code blocks), No sandbox isolation available
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/skills/code_executor.go`, `internal/skills/code_executor_test.go`, `internal/skills/skill_runtime.go`
- Test commands: `go test ./internal/skills -run TestCodeExecutor -count=1`, `go test ./internal/skills -run TestSkillRuntime -count=1`
- Done signal: Code executor tests prove skills with code blocks execute in sandbox with input/output contract
- Acceptance: Skill files with code blocks are executable in sandbox, Execution is sandboxed with the same isolation as tool calls, Skill code has access to skill-defined dependencies, Execution timeout prevents runaway skills, Execution output is captured and returned to agent, Skill can define input parameters accepted from agent
- Source refs: docs/content/papers/foundational-architectures.md, Voyager (arXiv:2305.16291), internal/skills/loader.go, internal/skills/executor.go, internal/tools/sandbox.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Skill dependency resolution and composition

- Phase: 6 / 6.L
- Owner: `skills`
- Size: `medium`
- Status: `planned`
- Priority: `P3`
- Contract: Skills can declare dependencies on other skills. The runtime resolves the dependency graph before execution. The agent can compose skills by chaining: output of Skill A feeds into input of Skill B. Dependencies are validated at load time.
- Trust class: operator
- Ready when: Skill code execution exists (6.L row 1), Skills have structured metadata with dependency declarations
- Not ready when: Skills have no dependency model, Code execution runtime not available
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/skills/dependency_resolver.go`, `internal/skills/dependency_resolver_test.go`, `internal/skills/composer.go`
- Test commands: `go test ./internal/skills -run TestDependencyResolver -count=1`, `go test ./internal/skills -run TestComposer -count=1`
- Done signal: Dependency tests prove circular deps rejected and chained composition works with error attribution
- Acceptance: Skill dependency graph resolved at load time, Circular dependencies detected and rejected with clear error, Missing dependencies reported with skill name and missing dep, Agent can chain Skill A output → Skill B input, Composition failures surface which step in the chain failed, Load-time validation catches 100% of dependency errors before execution
- Source refs: docs/content/papers/foundational-architectures.md, Voyager skill library composition, internal/skills/loader.go, internal/skills/registry.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Skill validation on load with execution proof

- Phase: 6 / 6.L
- Owner: `skills`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: When a skill is loaded or created, run a lightweight validation: parse code blocks, execute in sandbox with a canary input, verify output contract. Skills that fail validation are marked as broken and not offered to the agent. Passing skills carry a 'validated' trust marker.
- Trust class: system
- Ready when: Skill code execution exists (6.L row 1)
- Not ready when: No sandbox execution available for validation
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/skills/validator.go`, `internal/skills/validator_test.go`
- Test commands: `go test ./internal/skills -run TestValidator -count=1`
- Done signal: Validator tests prove broken skills are caught at load time with clear error messages
- Acceptance: Skills validated on load before appearing in agent's tool list, Canary execution with minimal input verifies basic functionality, Broken skills marked with error details (not silently skipped), Validation is fast (<500ms per skill, runs in background goroutine), Operator can force-load a broken skill with explicit override flag, Validation results visible in skill registry status
- Source refs: docs/content/papers/foundational-architectures.md, Voyager iterative prompting with execution feedback, internal/skills/loader.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
