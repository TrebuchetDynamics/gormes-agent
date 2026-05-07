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
| 5 / 5.N | Prompt Fragment Include System | Port agent-zero prompt fragment system: prompts stored as fragments with {{include filename.md}} directives, priority search order (agent profile > user > plugin > default), {{include original}} chains through hierarchy, variables substituted at render time. | operator, system | `internal/hermes/prompt_fragments_test.go` | Unblocks Agent profile customization, Plugin prompt injection. |
| 2 / 2.B.5 | Telegram forum thread fallback + send retry safety | Telegram outbound send/typing behavior preserves Hermes forum and retry safety: forum General-topic inbound messages without message_thread_id retain synthetic thread context `1`; outbound sends to General omit message_thread_id=1; send and typing retry without message_thread_id when Telegram returns BadRequest 'message thread not found'; non-thread BadRequest errors fail immediately; transient NetworkError sends retry with bounded attempts; TimedOut sends do not retry to avoid duplicate visible messages; RetryAfter sleeps/backoffs then retries; and once a chunk clears an invalid thread ID, later chunks in the same long response use no thread ID directly. | operator, gateway, system | `internal/channels/telegram/thread_fallback_test.go; internal/channels/telegram/send_retry_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.U | Sandbox isolation depth selection | Operator can select sandbox isolation depth: process-level (fast, weaker isolation), container-level (Docker/gVisor, balanced), or VM-level (Firecracker, strongest isolation). Default is process-level with transactional rollback. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.K | Behavioral pattern extraction from session logs | Mine session logs and tool execution audits for behavioral patterns: which tool sequences succeed vs fail, which reasoning patterns precede good outcomes, which response styles correlate with user satisfaction. Patterns feed into the self-evolution loop as candidate mutations. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.L | Skill code execution runtime | Skills are not just markdown instructions — they contain executable code that can be run in a sandboxed environment. This mirrors Voyager's code-as-action pattern: skills are validated, sandboxed, and can be composed by the agent at runtime. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.L | Skill dependency resolution and composition | Skills can declare dependencies on other skills. The runtime resolves the dependency graph before execution. The agent can compose skills by chaining: output of Skill A feeds into input of Skill B. Dependencies are validated at load time. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.L | Skill validation on load with execution proof | When a skill is loaded or created, run a lightweight validation: parse code blocks, execute in sandbox with a canary input, verify output contract. Skills that fail validation are marked as broken and not offered to the agent. Passing skills carry a 'validated' trust marker. | system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
