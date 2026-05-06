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
| 5 / 5.I | Extension Lifecycle Hook System | Port agent-zero extension lifecycle hook system: register extensions at 8+ lifecycle points (agent_init, monologue_start/end, message_loop_start/end, before_main_llm_call, prompt_before/after, stream_chunk, tool_before/after, context_deleted). Extension chain executes in registration order with per-extension timeout and panic isolation. | operator, system | `internal/kernel/extensions_test.go` | Unblocks Plugin ecosystem, Skill injection pipeline. |
| 5 / 5.M | Hermes Kanban production worker process binding | Gormes binds the fakeable Kanban dispatcher spawner to a production worker launcher that resolves Gormes profiles, builds the native gormes worker argv/env with Kanban context pins and the kanban-worker skill, redirects stdout/stderr to per-task logs with bounded rotation, records worker PID/run metadata, detects crashed worker PIDs, enforces per-task max-runtime caps through injected process controls, and reports worker_spawn_failed, worker_crashed, worker_timed_out, or task_circuit_open evidence without reading live Hermes config. | operator, gateway, child-agent, system | `internal/kanban/process_spawner_test.go; internal/kanban/worker_lifecycle_test.go; internal/gateway/kanban_dispatcher_test.go` | Unblocks Hermes Kanban multi-board, workspace, and run-history parity, Hermes Kanban slash/gateway/dashboard surfaces. |
| 5 / 5.N | Channels Capabilities Introspection | Port OpenClaw's channels capabilities: gormes channels capabilities shows provider capabilities (intents/scopes + supported features) for each configured channel. Enables operators to understand what each channel adapter supports before configuring it. | operator | `internal/channels/capabilities_test.go` | Unblocks Channel configuration UX. |
| 5 / 5.N | Prompt Fragment Include System | Port agent-zero prompt fragment system: prompts stored as fragments with {{include filename.md}} directives, priority search order (agent profile > user > plugin > default), {{include original}} chains through hierarchy, variables substituted at render time. | operator, system | `internal/hermes/prompt_fragments_test.go` | Unblocks Agent profile customization, Plugin prompt injection. |
| 5 / 5.N | Gateway probe auth/capability HTTP closeout | Close the remaining OpenClaw-style gateway probe gap after the completed TCP discover/probe slice: probe authenticated HTTP health/capabilities endpoints with redacted evidence for unavailable, unauthenticated, unsupported, and mismatched gateways. | operator, gateway | `cmd/gormes/gateway_probe_http_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.U | Transactional tool execution with snapshot/rollback | Wrap each uncertain tool call as an atomic transaction with ACID properties. Filesystem snapshot before execution; rollback on failure, error, or policy violation. Guarantees system consistency regardless of agent behavior. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.U | Sandbox isolation depth selection | Operator can select sandbox isolation depth: process-level (fast, weaker isolation), container-level (Docker/gVisor, balanced), or VM-level (Firecracker, strongest isolation). Default is process-level with transactional rollback. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.V | Gateway channel adapters publish to event bus | Each gateway channel adapter (Telegram, Discord, Slack, WhatsApp, WeChat) publishes incoming messages as standardized events on the bus, and subscribes to outgoing message events. Channel-specific translation lives in adapters; the bus carries channel-neutral events. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.K | Behavioral pattern extraction from session logs | Mine session logs and tool execution audits for behavioral patterns: which tool sequences succeed vs fail, which reasoning patterns precede good outcomes, which response styles correlate with user satisfaction. Patterns feed into the self-evolution loop as candidate mutations. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.L | Skill code execution runtime | Skills are not just markdown instructions — they contain executable code that can be run in a sandboxed environment. This mirrors Voyager's code-as-action pattern: skills are validated, sandboxed, and can be composed by the agent at runtime. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
