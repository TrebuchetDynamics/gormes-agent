---
title: "Next Slices"
weight: 30
aliases:
  - /building-gormes/next-slices/
---

# Next Slices

This page is generated from the canonical progress file and lists the highest
leverage contract-bearing roadmap rows to execute next.

The ordering is:

1. unblocked `P0` handoffs;
2. active `in_progress` rows;
3. `fixture_ready` rows;
4. unblocked rows that unblock other slices;
5. remaining `draft` contract rows.

Use this page when choosing implementation work. If a row is too broad, split
the row in `progress.json` before assigning it.

If no slices are listed, the next correct action is planner work: choose one
planned row from `progress.json` or a phase page and add enough contract detail
for it to appear here. Do not infer that an empty generated list means the
roadmap is complete.

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 4 / 4.K | Hermes fallback activation + classifier carve-outs | Gormes extends the existing provider fallback chain from generic dispatch into Hermes' current agent-loop behavior: fallback_model accepts a single object or ordered list, ignores missing provider/model entries without panics, normalizes supported fallback providers through the provider registry, activates fallback after empty-response retry exhaustion, updates compressor/model context after activation, and treats malformed provider JSON as retryable transport corruption rather than a local validation error. | operator, system | `internal/hermes/fallback_activation_test.go; internal/kernel/provider_fallback_test.go` | Unblocks Provider-tool-memory golden transcript suite, Release readiness e2e gate. |
| 5 / 5.I | Extension Lifecycle Hook System | Port agent-zero extension lifecycle hook system: register extensions at 8+ lifecycle points (agent_init, monologue_start/end, message_loop_start/end, before_main_llm_call, prompt_before/after, stream_chunk, tool_before/after, context_deleted). Extension chain executes in registration order with per-extension timeout and panic isolation. | operator, system | `internal/kernel/extensions_test.go` | Unblocks Plugin ecosystem, Skill injection pipeline. |
| 5 / 5.M | Hermes Kanban production worker process binding | Gormes binds the fakeable Kanban dispatcher spawner to a production worker launcher that resolves Gormes profiles, builds the native gormes worker argv/env with Kanban context pins and the kanban-worker skill, redirects stdout/stderr to per-task logs with bounded rotation, records worker PID/run metadata, detects crashed worker PIDs, enforces per-task max-runtime caps through injected process controls, and reports worker_spawn_failed, worker_crashed, worker_timed_out, or task_circuit_open evidence without reading live Hermes config. | operator, gateway, child-agent, system | `internal/kanban/process_spawner_test.go; internal/kanban/worker_lifecycle_test.go; internal/gateway/kanban_dispatcher_test.go` | Unblocks Hermes Kanban multi-board, workspace, and run-history parity, Hermes Kanban slash/gateway/dashboard surfaces. |
| 5 / 5.N | Channels Capabilities Introspection | Port OpenClaw's channels capabilities: gormes channels capabilities shows provider capabilities (intents/scopes + supported features) for each configured channel. Enables operators to understand what each channel adapter supports before configuring it. | operator | `internal/channels/capabilities_test.go` | Unblocks Channel configuration UX. |
| 5 / 5.N | Prompt Fragment Include System | Port agent-zero prompt fragment system: prompts stored as fragments with {{include filename.md}} directives, priority search order (agent profile > user > plugin > default), {{include original}} chains through hierarchy, variables substituted at render time. | operator, system | `internal/hermes/prompt_fragments_test.go` | Unblocks Agent profile customization, Plugin prompt injection. |
| 5 / 5.N | ACP bridge client compatibility | Close the OpenClaw ACP bridge gap by adding Gormes client-side ACP connection/proxy behavior in addition to the existing server-facing package, with status/doctor evidence for connected, unavailable, and unsupported remote ACP endpoints. | operator, gateway | `internal/acp/bridge_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | Gateway discover/probe command | Add OpenClaw-style gateway discovery/probe commands for Gormes operators: discover local/remote gateway endpoints, probe auth/health/capabilities, and return redacted structured evidence for unavailable, unauthenticated, and mismatched gateways. | operator, gateway | `cmd/gormes/gateway_discover_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.U | Transactional tool execution with snapshot/rollback | Wrap each uncertain tool call as an atomic transaction with ACID properties. Filesystem snapshot before execution; rollback on failure, error, or policy violation. Guarantees system consistency regardless of agent behavior. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.U | Sandbox isolation depth selection | Operator can select sandbox isolation depth: process-level (fast, weaker isolation), container-level (Docker/gVisor, balanced), or VM-level (Firecracker, strongest isolation). Default is process-level with transactional rollback. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.V | Gateway channel adapters publish to event bus | Each gateway channel adapter (Telegram, Discord, Slack, WhatsApp, WeChat) publishes incoming messages as standardized events on the bus, and subscribes to outgoing message events. Channel-specific translation lives in adapters; the bus carries channel-neutral events. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
