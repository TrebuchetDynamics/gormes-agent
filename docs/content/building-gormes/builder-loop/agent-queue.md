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
## 1. Security Audit Command

- Phase: 5 / 5.J
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Port OpenClaw's security audit: gormes security audit --deep --fix --json. Deep mode includes live gateway probe checks. Fix mode applies safe remediations and file-permission fixes. JSON mode produces machine-readable output. Audit categories: gateway auth status, state integrity, channel security warnings, shell blocklist coverage, filesystem scoping, credential redaction.
- Trust class: operator
- Ready when: Shell blocklist + filesystem scoping (5.J) is operational., Secrets runtime controls (5.J) is operational.
- Not ready when: The row performs destructive fixes without --fix flag., The row requires live gateway for basic audit checks.
- Degraded mode: Unauditable surfaces, missing probes, or unfixable issues report per-category status with severity level and recommended action rather than blocking the entire audit.
- Fixture: `internal/tools/security_audit_test.go`
- Write scope: `internal/tools/security_audit.go`, `internal/tools/security_audit_test.go`, `cmd/gormes/security.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestSecurityAudit -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes security audit ships with --deep, --fix, and --json modes.
- Acceptance: gormes security audit --deep checks gateway auth, state integrity, channel security, shell blocklist, filesystem scoping, and credential redaction., --fix applies safe remediations (file permissions, auth token generation)., --json produces machine-readable output with per-category pass/fail/warn.
- Source refs: openclaw security audit --deep --fix --json CLI surface, docs/content/building-gormes/openclaw-platform-parity-audit.md, docs/content/building-gormes/must-have-features.md
- Unblocks: Production security posture
- Why now: P0 handoff; needs contract proof before closeout.

## 2. ACP Client Bridge Mode

- Phase: 5 / 5.H
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Complete the ACP integration with client bridge mode: gormes acp client connects to the Go-native ACP server (5.H server side is validated) with session key/label resolution, reset-session capability, require-existing guard, provenance modes (off/meta/meta+receipt), and --no-prefix-cwd flag. Match OpenClaw's ACP bridge surface.
- Trust class: operator, system
- Ready when: ACP server side (5.H) is validated — server manifest, auth, session, tools, permissions, events., Gateway session store supports key/label resolution.
- Not ready when: The row requires a running Hermes Python process., The row exposes unauthenticated ACP endpoints.
- Degraded mode: Unsupported ACP provider, missing auth, session key not found, or permission prompt timeout returns explicit acp_client_row_backed evidence with available fallback modes.
- Fixture: `internal/acp/client_test.go`
- Write scope: `internal/acp/client.go`, `internal/acp/client_test.go`, `cmd/gormes/acp.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/acp -run TestClient -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes acp client ships with session key/label resolution and provenance modes.
- Acceptance: gormes acp client --session agent:main:main connects to local ACP server., Provenance mode meta+receipt writes signed receipts., --reset-session clears and reinitializes the session key., --require-existing fails when session does not exist.
- Source refs: openclaw acp client CLI surface, internal/acp/server_manifest_test.go, docs/content/building-gormes/openclaw-platform-parity-audit.md
- Unblocks: Multi-agent interoperability, Editor integrations
- Why now: Unblocks Multi-agent interoperability, Editor integrations.

## 3. Extension Lifecycle Hook System

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

## 4. System Events, Heartbeat, and Presence

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Port OpenClaw's system event surface: gormes system event enqueues a system event and optionally triggers a heartbeat; gormes system heartbeat shows and controls heartbeat state; gormes system presence lists system presence entries. Events are written to the audit ledger (JSONL) and surfaced in gormes status.
- Trust class: operator, system
- Ready when: Audit JSONL ledger (internal/audit/) is operational., Session health monitoring (5.N) provides heartbeat data.
- Not ready when: The row introduces a new event bus or message queue., The row depends on external monitoring services.
- Degraded mode: Missing audit ledger, event queue full, or heartbeat disabled reports system_unavailable with ledger path/error details.
- Fixture: `internal/tools/system_events_test.go`
- Write scope: `internal/tools/system_events.go`, `internal/tools/system_events_test.go`, `cmd/gormes/system.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestSystemEvents -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes system ships with event, heartbeat, and presence subcommands.
- Acceptance: gormes system event 'gateway restart' enqueues event with timestamp., gormes system heartbeat shows on/off state and last beat time., gormes system presence lists active components with last-seen times.
- Source refs: openclaw system event/heartbeat/presence CLI surface, internal/audit/ (JSONL audit log), docs/content/building-gormes/openclaw-platform-parity-audit.md
- Unblocks: Operator observability, Gateway discover/probe diagnostics
- Why now: Unblocks Operator observability, Gateway discover/probe diagnostics.

## 5. Gateway Discover and Probe

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Port OpenClaw's gateway network discovery: gormes gateway discover finds local gateways via Bonjour/mDNS; gormes gateway probe shows gateway reachability + discovery + health + status summary; gormes gateway usage-cost fetches usage cost summary from session logs.
- Trust class: operator
- Ready when: Gateway status and health commands are operational., Session store has usage/cost data accessible.
- Not ready when: The row requires Tailscale or CoreDNS for basic discovery., The row sends unauthenticated probe requests.
- Degraded mode: No gateways discovered, probe timeout, or usage data unavailable reports per-endpoint status with failure reason.
- Fixture: `internal/tools/gateway_discover_test.go`
- Write scope: `internal/tools/gateway_discover.go`, `internal/tools/gateway_discover_test.go`, `cmd/gormes/gateway_discover.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestGatewayDiscover -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes gateway discover/probe/usage-cost ships.
- Acceptance: gormes gateway discover lists local gateways with addresses and ports., gormes gateway probe shows reachability + discovery + health + status., gormes gateway usage-cost shows per-session and aggregate token costs.
- Source refs: openclaw gateway discover/probe/usage-cost CLI surface, docs/content/building-gormes/openclaw-platform-parity-audit.md
- Unblocks: Multi-instance fleet management
- Why now: Unblocks Multi-instance fleet management.

## 6. Channels Capabilities Introspection

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

## 7. Prompt Fragment Include System

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

## 8. Plan gate hook in agent turn loop

- Phase: 4 / 4.L
- Owner: `orchestrator`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Before tool execution, the agent loop invokes a plan-gate safety check. Unsafe plans are refused with explanation. Safe plans proceed. This mirrors MOSAIC (2025) plan->check->act/refuse pattern.
- Trust class: operator, system
- Ready when: Agent turn loop has a hook point before tool dispatch, Tool registry exposes safety-relevant metadata per tool
- Not ready when: Agent loop is not refactorable to add pre-tool hooks, Tool registry has no permission/safety metadata
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/hermes/safety_plan_gate.go`, `internal/hermes/safety_plan_gate_test.go`
- Test commands: `go test ./internal/hermes -run TestPlanGate -count=1`
- Done signal: Plan gate tests prove safe plans pass and unsafe plans are refused
- Acceptance: Plan gate evaluates agent plan before any tool executes, Unsafe plans are refused with structured explanation, Safe plans pass through with zero added latency >10ms P99, Plan gate is testable with mock tool sets
- Source refs: docs/content/papers/safety-and-deployment.md, internal/hermes/turn.go, internal/hermes/agent_loop.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Tool gate pre-execution validation

- Phase: 4 / 4.L
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Each individual tool invocation is checked against intent alignment before execution. This mirrors IntentGuard's two-gate architecture: plan gate (strategic) + tool gate (tactical).
- Trust class: operator, system
- Ready when: Plan gate exists (4.L row 1), Tool registry exposes permission model
- Not ready when: Plan gate not yet implemented
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/tools/safety_tool_gate.go`, `internal/tools/safety_tool_gate_test.go`
- Test commands: `go test ./internal/tools -run TestToolGate -count=1`
- Done signal: Tool gate tests prove intent-aligned calls pass and drift calls are blocked
- Acceptance: Tool gate evaluates every tool call before execution, Tool calls outside intent scope are blocked, Intent drift across multi-step tool chains is detected, Tool gate adds <5ms P99 overhead
- Source refs: docs/content/papers/safety-and-deployment.md, internal/tools/registry.go, internal/tools/executor.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. Refusal-as-action in ReAct cycle

- Phase: 4 / 4.L
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: The agent loop supports 'refuse' as a first-class action in the ReAct cycle. When safety gates reject a planned action, the agent can refuse and explain why, rather than silently failing or hallucinating a different action.
- Trust class: operator
- Ready when: Plan gate or tool gate exists (4.L rows 1-2)
- Not ready when: No safety gate to produce refusal signals
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/hermes/refuse_action.go`, `internal/hermes/refuse_action_test.go`
- Test commands: `go test ./internal/hermes -run TestRefuseAction -count=1`
- Done signal: Refusal tests prove the agent can refuse, explain, and recover
- Acceptance: ReAct cycle accepts RefuseAction alongside ToolAction, Refused actions produce user-visible explanation, Agent can recover and try alternative approach after refusal, Refusal does not count as an error in session stats
- Source refs: docs/content/papers/safety-and-deployment.md, internal/hermes/turn.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
