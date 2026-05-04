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

## 4. Circuit breaker per provider and API key

- Phase: 4 / 4.M
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Each provider connection gets an independent circuit breaker tracking consecutive failures. After threshold (default 5), breaker trips to OPEN and all calls fast-fail for cooldown period (default 30s). Half-open state allows single probe request.
- Trust class: system
- Ready when: Provider interface supports health status reporting
- Not ready when: No per-provider error tracking
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/hermes/circuit_breaker.go`, `internal/hermes/circuit_breaker_test.go`
- Test commands: `go test ./internal/hermes -run TestCircuitBreaker -count=1`
- Done signal: Circuit breaker tests prove CLOSED→OPEN→HALF_OPEN→CLOSED transitions with configurable thresholds
- Acceptance: Circuit breaker trips after N consecutive failures, OPEN state fast-fails without making network calls, Half-open probe succeeds → breaker resets to CLOSED, Half-open probe fails → breaker returns to OPEN, Each API key tracked independently, Breaker state transitions are logged at INFO level
- Source refs: docs/content/papers/safety-and-deployment.md, internal/hermes/fallback_chain.go, internal/hermes/provider.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. P95 latency-aware failover

- Phase: 4 / 4.M
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Provider selection considers P95 latency in addition to health status. Degraded-but-not-dead providers get reduced traffic weight rather than full exclusion. Rolling window tracks last N requests.
- Trust class: system
- Ready when: Provider interface supports latency tracking, Circuit breaker exists (4.M row 1)
- Not ready when: No latency data collection from providers
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/hermes/latency_router.go`, `internal/hermes/latency_router_test.go`
- Test commands: `go test ./internal/hermes -run TestLatencyRouter -count=1`
- Done signal: Latency router tests prove degraded providers get reduced weight while healthy ones get priority
- Acceptance: P95 latency tracked per provider in rolling window, Degraded providers receive reduced traffic weight, Non-degraded providers prioritized in selection, Latency window configurable (default: last 100 requests)
- Source refs: docs/content/papers/safety-and-deployment.md, internal/hermes/fallback_chain.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Capability-based model tier routing

- Phase: 4 / 4.M
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Route simple queries to cheap models and complex queries to capable models based on a fast classifier. Avoids sending 'hello' to Claude Opus and avoids sending multi-file refactors to a 7B model.
- Trust class: operator
- Ready when: Multiple model tiers configured in provider registry
- Not ready when: Only one model tier available
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/hermes/capability_router.go`, `internal/hermes/capability_router_test.go`
- Test commands: `go test ./internal/hermes -run TestCapabilityRouter -count=1`
- Done signal: Capability router tests prove simple queries hit cheap tier and complex queries hit capable tier
- Acceptance: Simple queries (greetings, single-file edits) route to cheap tier, Complex queries (multi-file refactor, architecture) route to capable tier, Classifier is fast (<100ms) and does not add LLM call overhead, Operator can override classifier with explicit model selection, Classification failures fall back to capable tier (safe default)
- Source refs: docs/content/papers/safety-and-deployment.md, internal/hermes/provider.go, Beluga AI model-switching docs
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. ACP bridge client compatibility

- Phase: 5 / 5.N
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Close the OpenClaw ACP bridge gap by adding Gormes client-side ACP connection/proxy behavior in addition to the existing server-facing package, with status/doctor evidence for connected, unavailable, and unsupported remote ACP endpoints.
- Trust class: operator, gateway
- Ready when: Current internal/acp server stubs are audited so the row can distinguish server support from outbound bridge/client support.
- Not ready when: The slice claims ACP parity from package existence without a client/proxy fixture.
- Degraded mode: Unavailable ACP endpoints report acp_bridge_unavailable with endpoint and auth source redacted; local Gormes operation continues without pretending ACP is connected.
- Fixture: `internal/acp/bridge_test.go`
- Write scope: `internal/acp/`, `cmd/gormes/acp.go`, `cmd/gormes/doctor.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/acp ./cmd/gormes -run 'ACP.*Bridge\|ACP.*Status' -count=1`, `go run ./cmd/progress validate`
- Done signal: ACP bridge fixtures prove outbound client/proxy behavior or explicitly degraded status.
- Acceptance: Gormes can configure an outbound ACP endpoint and report connection status., Bridge requests are covered by hermetic fake endpoint tests., Doctor/status distinguishes server-only support from client bridge support.
- Source refs: ../openclaw/src/acp/, internal/acp/, cmd/gormes/
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Gateway discover/probe command

- Phase: 5 / 5.N
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Add OpenClaw-style gateway discovery/probe commands for Gormes operators: discover local/remote gateway endpoints, probe auth/health/capabilities, and return redacted structured evidence for unavailable, unauthenticated, and mismatched gateways.
- Trust class: operator, gateway
- Ready when: Gateway status and doctor endpoints are stable enough to probe without live network dependencies in tests.
- Not ready when: The slice requires a live Telegram or Discord gateway to pass tests.
- Degraded mode: Probe failures show endpoint, status code, and auth source classification without leaking bearer tokens or gateway passwords.
- Fixture: `cmd/gormes/gateway_discover_test.go`
- Write scope: `cmd/gormes/gateway.go`, `cmd/gormes/gateway_discover.go`, `cmd/gormes/gateway_discover_test.go`, `internal/apiserver/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes ./internal/apiserver -run 'Gateway.*Discover\|Gateway.*Probe' -count=1`, `go run ./cmd/progress validate`
- Done signal: Gateway discover/probe gives operators redacted connection evidence without reading source code.
- Acceptance: gormes gateway discover reports candidate local endpoints and active PID/status evidence., gormes gateway probe --url uses fake HTTP tests for success, unauthorized, unavailable, and malformed response., Output redacts tokens/passwords and includes exact probe counts.
- Source refs: ../openclaw/src/cli/gateway-secret-options.ts, ../openclaw/src/security/audit-gateway-auth-selection.test.ts, cmd/gormes/gateway.go, internal/apiserver/server.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Pre-execution command classification

- Phase: 5 / 5.U
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Classify every tool command as safe (whitelist), unsafe (blacklist), or uncertain (needs sandbox snapshot) before execution. Safe commands run directly. Unsafe commands are blocked. Uncertain commands trigger snapshot/rollback wrapper.
- Trust class: system
- Ready when: Tool execution path is interceptable, Sandbox filesystem supports snapshots
- Not ready when: Tool execution cannot be intercepted, No sandbox filesystem available
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/tools/command_classifier.go`, `internal/tools/command_classifier_test.go`, `internal/tools/transactional_executor.go`, `internal/tools/transactional_executor_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestCommandClassifier -count=1`, `go test ./internal/tools -run TestTransactionalExecutor -count=1`, `go run ./cmd/progress validate`
- Done signal: Classifier tests prove safe/unsafe/uncertain classification for representative command sets
- Acceptance: Commands classified as safe/unsafe/uncertain before execution, Safe commands execute directly with <1ms overhead, Unsafe commands are blocked with audit log entry, Uncertain commands trigger snapshot before execution, Classification rules are configurable per session
- Source refs: docs/content/papers/safety-and-deployment.md, arXiv:2512.12806 (Fault-Tolerant Sandboxing 2025), internal/tools/executor.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. Transactional tool execution with snapshot/rollback

- Phase: 5 / 5.U
- Owner: `tools`
- Size: `large`
- Status: `planned`
- Priority: `P1`
- Contract: Wrap each uncertain tool call as an atomic transaction with ACID properties. Filesystem snapshot before execution; rollback on failure, error, or policy violation. Guarantees system consistency regardless of agent behavior.
- Trust class: system
- Ready when: Command classifier exists (5.U row 1), Sandbox filesystem supports snapshot/rollback
- Not ready when: No snapshot mechanism available on target OS
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/tools/transactional_executor.go`, `internal/tools/transactional_executor_test.go`, `internal/tools/snapshot.go`
- Test commands: `go test ./internal/tools -run TestTransactionalExecutor -count=1`, `go test ./internal/tools -run TestSnapshot -count=1`
- Done signal: Transactional executor tests prove rollback on failure and commit on success with filesystem integrity
- Acceptance: Uncertain tool calls create filesystem snapshot before execution, Failed tool calls roll back to pre-execution state, Successful tool calls commit snapshot changes, Rollback is atomic — no partial state after failure, Snapshot overhead <2s per transaction on typical project, Audit log records snapshot/rollback/commit events
- Source refs: docs/content/papers/safety-and-deployment.md, arXiv:2512.12806 (Fault-Tolerant Sandboxing 2025), internal/tools/executor.go, internal/tools/sandbox.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
