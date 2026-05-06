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

## 2. Hermes Kanban production worker process binding

- Phase: 5 / 5.M
- Owner: `orchestrator`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Gormes binds the fakeable Kanban dispatcher spawner to a production worker launcher that resolves Gormes profiles, builds the native gormes worker argv/env with Kanban context pins and the kanban-worker skill, redirects stdout/stderr to per-task logs with bounded rotation, records worker PID/run metadata, detects crashed worker PIDs, enforces per-task max-runtime caps through injected process controls, and reports worker_spawn_failed, worker_crashed, worker_timed_out, or task_circuit_open evidence without reading live Hermes config.
- Trust class: operator, gateway, child-agent, system
- Ready when: Hermes Kanban dispatcher and worker spawn loop is complete., The builder can inject fake process start, PID liveness, signal/kill, log filesystem, and clock seams; no unit test starts or kills a real worker process., Gormes profile name/root helpers are available for profile resolution without importing Hermes config.
- Not ready when: The native root command cannot yet accept the Hermes `-p/--profile` and `--skills kanban-worker` invocation that this row's worker argv acceptance asserts., The implementation shells out to `hermes`, reads HERMES_HOME or ~/.hermes as live config, or uses Hermes Python helpers., Unit tests depend on a real subprocess, real PID table, real signal delivery, or the operator's PATH., The slice changes Kanban worker tools, dashboard routes, board registry semantics, or slash-command parsing instead of only production worker process binding.
- Degraded mode: Missing gormes binaries, invalid profile names, unwritable workspaces/log paths, stale PIDs, process-kill failures, and max-runtime expiry return typed evidence and release or block the task according to the dispatcher failure policy without spawning Hermes Python or killing unrelated processes.
- Fixture: `internal/kanban/process_spawner_test.go; internal/kanban/worker_lifecycle_test.go; internal/gateway/kanban_dispatcher_test.go`
- Write scope: `internal/kanban/`, `internal/gateway/`, `internal/cli/`, `cmd/gormes/kanban.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/kanban -run 'TestKanban(ProcessSpawner\|WorkerLifecycle)' -count=1`, `go test ./internal/gateway -run TestManagerKanbanDispatcher -count=1`, `go run ./cmd/progress validate`
- Done signal: Kanban dispatcher production binding launches native Gormes workers through fakeable process seams, records PID/log/runtime evidence, reclaims crashed or timed-out workers, and never depends on Hermes Python or live subprocesses in tests.
- Acceptance: Spawner fixtures prove the production launcher builds native `gormes -p <profile> --skills kanban-worker chat -q ...` argv, GORMES_KANBAN_* env, cwd, and redacted logs without any HERMES_* leak., Log fixtures prove per-task stdout/stderr paths are under the Gormes Kanban log root and rotate before spawn when over the configured limit., PID fixtures prove spawned PIDs are recorded on task and run rows, crashed workers are reclaimed with worker_crashed evidence, and stale or reused PID evidence does not kill unrelated processes., Runtime-cap fixtures prove expired tasks receive injected TERM/KILL-style process controls, record worker_timed_out evidence, and return to ready or blocked according to the failure policy., Gateway wiring fixtures prove production dispatcher config can use the process spawner while tests keep using the fake spawner seam.
- Source refs: ./hermes-agent/hermes_cli/kanban_db.py@b816fd4e2:_default_spawn, ./hermes-agent/hermes_cli/kanban_db.py@b816fd4e2:detect_crashed_workers, ./hermes-agent/hermes_cli/kanban_db.py@b816fd4e2:enforce_max_runtime, ./hermes-agent/hermes_cli/kanban_db.py@b816fd4e2:worker_logs_dir, ./hermes-agent/hermes_cli/kanban_db.py@b816fd4e2:heartbeat_worker, ./hermes-agent/tests/hermes_cli/test_kanban_db.py@b816fd4e2:test_dispatcher_spawn_injects_kanban_db_and_workspaces_root, ./hermes-agent/tests/hermes_cli/test_kanban_db.py@b816fd4e2:test_dispatch_promotes_ready_and_spawns, ./hermes-agent/tests/hermes_cli/test_kanban_db.py@b816fd4e2:test_dispatch_spawn_failure_releases_claim, internal/kanban/dispatcher.go, internal/kanban/store.go, internal/cli/profile_root.go, cmd/gormes/profile.go
- Unblocks: Hermes Kanban multi-board, workspace, and run-history parity, Hermes Kanban slash/gateway/dashboard surfaces
- Why now: Unblocks Hermes Kanban multi-board, workspace, and run-history parity, Hermes Kanban slash/gateway/dashboard surfaces.

## 3. Channels Capabilities Introspection

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

## 4. Prompt Fragment Include System

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

## 5. ACP bridge client compatibility

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

## 6. Gateway discover/probe command

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

## 7. Transactional tool execution with snapshot/rollback

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

## 8. Sandbox isolation depth selection

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

## 9. Gateway channel adapters publish to event bus

- Phase: 5 / 5.V
- Owner: `gateway`
- Size: `large`
- Status: `planned`
- Priority: `P1`
- Contract: Each gateway channel adapter (Telegram, Discord, Slack, WhatsApp, WeChat) publishes incoming messages as standardized events on the bus, and subscribes to outgoing message events. Channel-specific translation lives in adapters; the bus carries channel-neutral events.
- Trust class: system
- Ready when: Event bus exists (5.V row 1), Channel adapters are refactorable to add bus hooks
- Not ready when: Event bus not yet implemented, Channel adapters too tightly coupled to refactor
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/channels/telegram/bus_adapter.go`, `internal/channels/discord/bus_adapter.go`, `internal/channels/slack/bus_adapter.go`, `internal/gateway/event_dispatch.go`, `internal/gateway/event_dispatch_test.go`
- Test commands: `go test ./internal/gateway -run TestEventDispatch -count=1`, `go test ./internal/channels/telegram -run TestBusAdapter -count=1`
- Done signal: Integration tests prove messages flow from channel→bus→agent and back through all adapter types
- Acceptance: Incoming Telegram message → MessageReceived event on bus, Incoming Discord message → MessageReceived event on bus, Incoming Slack message → MessageReceived event on bus, Outgoing reply event → channel-specific delivery by adapter subscriber, Channel adapter failures are isolated — one channel crash doesn't affect others, Message events carry channel provenance (source, channel_id, user_id)
- Source refs: docs/content/papers/agentic-os-design.md, internal/events/bus.go, internal/channels/telegram/adapter.go, internal/channels/discord/adapter.go, internal/channels/slack/adapter.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. Agent turn and tool execution events on bus

- Phase: 5 / 5.V
- Owner: `orchestrator`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Agent turns (start, thought, action, observation, complete, error) and tool executions (start, progress, complete, error) are published as structured events on the bus. Enables TUI, web dashboard, and audit log to observe agent activity without polling.
- Trust class: system
- Ready when: Event bus exists (5.V row 1), Agent loop has hook points for event emission
- Not ready when: Event bus not yet implemented, Agent loop cannot be refactored for hooks
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/hermes/turn_events.go`, `internal/hermes/turn_events_test.go`, `internal/tools/tool_events.go`
- Test commands: `go test ./internal/hermes -run TestTurnEvents -count=1`, `go test ./internal/tools -run TestToolEvents -count=1`
- Done signal: Event tests prove turn lifecycle emits all expected events with correct trace_id linking
- Acceptance: TurnStart event emitted when agent begins processing, Thought event emitted for each reasoning step, ToolCall event emitted when tool is invoked, ToolResult event emitted when tool completes, TurnComplete event emitted with summary, All events carry trace_id linking them to a session turn
- Source refs: docs/content/papers/agentic-os-design.md, internal/events/bus.go, internal/hermes/turn.go, internal/tools/executor.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
