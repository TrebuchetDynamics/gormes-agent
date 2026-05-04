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
| 5 / 5.N | Channels Capabilities Introspection | Port OpenClaw's channels capabilities: gormes channels capabilities shows provider capabilities (intents/scopes + supported features) for each configured channel. Enables operators to understand what each channel adapter supports before configuring it. | operator | `internal/channels/capabilities_test.go` | Unblocks Channel configuration UX. |
| 5 / 5.N | Prompt Fragment Include System | Port agent-zero prompt fragment system: prompts stored as fragments with {{include filename.md}} directives, priority search order (agent profile > user > plugin > default), {{include original}} chains through hierarchy, variables substituted at render time. | operator, system | `internal/hermes/prompt_fragments_test.go` | Unblocks Agent profile customization, Plugin prompt injection. |
| 4 / 4.M | Circuit breaker per provider and API key | Each provider connection gets an independent circuit breaker tracking consecutive failures. After threshold (default 5), breaker trips to OPEN and all calls fast-fail for cooldown period (default 30s). Half-open state allows single probe request. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.M | P95 latency-aware failover | Provider selection considers P95 latency in addition to health status. Degraded-but-not-dead providers get reduced traffic weight rather than full exclusion. Rolling window tracks last N requests. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.M | Capability-based model tier routing | Route simple queries to cheap models and complex queries to capable models based on a fast classifier. Avoids sending 'hello' to Claude Opus and avoids sending multi-file refactors to a 7B model. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | ACP bridge client compatibility | Close the OpenClaw ACP bridge gap by adding Gormes client-side ACP connection/proxy behavior in addition to the existing server-facing package, with status/doctor evidence for connected, unavailable, and unsupported remote ACP endpoints. | operator, gateway | `internal/acp/bridge_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | Gateway discover/probe command | Add OpenClaw-style gateway discovery/probe commands for Gormes operators: discover local/remote gateway endpoints, probe auth/health/capabilities, and return redacted structured evidence for unavailable, unauthenticated, and mismatched gateways. | operator, gateway | `cmd/gormes/gateway_discover_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.U | Pre-execution command classification | Classify every tool command as safe (whitelist), unsafe (blacklist), or uncertain (needs sandbox snapshot) before execution. Safe commands run directly. Unsafe commands are blocked. Uncertain commands trigger snapshot/rollback wrapper. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.U | Transactional tool execution with snapshot/rollback | Wrap each uncertain tool call as an atomic transaction with ACID properties. Filesystem snapshot before execution; rollback on failure, error, or policy violation. Guarantees system consistency regardless of agent behavior. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
