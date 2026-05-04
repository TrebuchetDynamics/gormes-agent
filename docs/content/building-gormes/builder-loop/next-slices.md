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
| 5 / 5.N | System Events, Heartbeat, and Presence | Port OpenClaw's system event surface: gormes system event enqueues a system event and optionally triggers a heartbeat; gormes system heartbeat shows and controls heartbeat state; gormes system presence lists system presence entries. Events are written to the audit ledger (JSONL) and surfaced in gormes status. | operator, system | `internal/tools/system_events_test.go` | Unblocks Operator observability, Gateway discover/probe diagnostics. |
| 5 / 5.N | Gateway Discover and Probe | Port OpenClaw's gateway network discovery: gormes gateway discover finds local gateways via Bonjour/mDNS; gormes gateway probe shows gateway reachability + discovery + health + status summary; gormes gateway usage-cost fetches usage cost summary from session logs. | operator | `internal/tools/gateway_discover_test.go` | Unblocks Multi-instance fleet management. |
| 5 / 5.N | Channels Capabilities Introspection | Port OpenClaw's channels capabilities: gormes channels capabilities shows provider capabilities (intents/scopes + supported features) for each configured channel. Enables operators to understand what each channel adapter supports before configuring it. | operator | `internal/channels/capabilities_test.go` | Unblocks Channel configuration UX. |
| 5 / 5.N | Prompt Fragment Include System | Port agent-zero prompt fragment system: prompts stored as fragments with {{include filename.md}} directives, priority search order (agent profile > user > plugin > default), {{include original}} chains through hierarchy, variables substituted at render time. | operator, system | `internal/hermes/prompt_fragments_test.go` | Unblocks Agent profile customization, Plugin prompt injection. |
| 4 / 4.L | Plan gate hook in agent turn loop | Before tool execution, the agent loop invokes a plan-gate safety check. Unsafe plans are refused with explanation. Safe plans proceed. This mirrors MOSAIC (2025) plan->check->act/refuse pattern. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.L | Tool gate pre-execution validation | Each individual tool invocation is checked against intent alignment before execution. This mirrors IntentGuard's two-gate architecture: plan gate (strategic) + tool gate (tactical). | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.L | Refusal-as-action in ReAct cycle | The agent loop supports 'refuse' as a first-class action in the ReAct cycle. When safety gates reject a planned action, the agent can refuse and explain why, rather than silently failing or hallucinating a different action. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.L | Safety loop end-to-end integration | Integration test proving plan-gate + tool-gate + refusal work together in a complete agent turn. Covers: safe turn passes, unsafe plan blocked, tool drift blocked, multi-step chain with mixed safe/unsafe steps. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.M | Circuit breaker per provider and API key | Each provider connection gets an independent circuit breaker tracking consecutive failures. After threshold (default 5), breaker trips to OPEN and all calls fast-fail for cooldown period (default 30s). Half-open state allows single probe request. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
