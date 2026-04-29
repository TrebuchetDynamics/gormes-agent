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

<!-- PROGRESS:START kind=agent-queue -->
## 1. Mid-run steer injection between tool calls

- Phase: 2 / 2.F.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: Gateway /steer guidance can be delivered into an in-flight native Gormes turn after the current tool batch, preserving provider message-role alternation by appending a clear user-guidance marker to the last tool-result message before the next provider request; no Telegram-only path, hermes-agent runtime call, or next-turn duplicate is introduced.
- Trust class: operator, gateway, system
- Ready when: The shipped /steer queue fallback row remains complete and provides no-active-turn/degraded behavior., internal/kernel has a single-owner event mailbox and a tool-call loop where a steer event can be drained after executeToolCallsInterruptible returns results but before the next OpenStream call., Focused tests can use hermetic fake streams/tools and fixture channels only — no live Telegram, provider, or hermes-agent runtime service is required.
- Not ready when: The slice starts, shells out to, imports, or depends on hermes-agent runtime services., The slice handles only Telegram instead of routing through the shared gateway CommandRegistry, Manager, and kernel PlatformEvent seams., The slice injects `/steer` as a fresh user message between assistant/tool messages, breaking provider message-role ordering., The slice duplicates guidance by both appending it to a tool result and leaving the same text queued as a follow-up turn., The slice changes /queue, /busy, /stop cancellation, provider adapter behavior, or platform-specific channel identity rules outside the named write_scope.
- Degraded mode: If no tool-result boundary is available, /steer keeps the existing queue/degraded fallback evidence instead of silently dropping guidance or injecting a malformed provider message.
- Fixture: `internal/kernel/tool_interrupt_test.go and internal/gateway/steer_queue_test.go`
- Write scope: `internal/kernel/frame.go`, `internal/kernel/kernel.go`, `internal/kernel/tool_interrupt_test.go`, `internal/gateway/manager.go`, `internal/gateway/steer_queue_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/kernel -run 'Steer\|ToolInterrupt\|Tool' -count=1`, `go test ./internal/gateway -run '^TestSteerCommandRegistry_' -count=1`, `go test ./internal/kernel ./internal/gateway -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Kernel and gateway tests prove /steer can inject guidance after a tool batch in the native Go runtime, while fallback queue/degraded behavior remains intact and channel-neutral.
- Acceptance: A running kernel accepts a steer PlatformEvent while a turn is in-flight and stores bounded guidance without changing turn phase to failed or canceled., After executeToolCallsInterruptible returns one or more tool results, pending steer guidance is appended to the last tool-result message with an explicit user-guidance marker before the next provider OpenStream request., Multiple pending steer events concatenate deterministically and drain exactly once., If the turn finishes or is canceled before another tool-result batch exists, the gateway keeps the existing queue/degraded fallback behavior rather than silently dropping or duplicating guidance., Fixture coverage proves the path through Manager/EventSteer is channel-neutral and does not special-case Telegram.
- Source refs: ../hermes-agent/run_agent.py:4109, ../hermes-agent/run_agent.py:4145, ../hermes-agent/run_agent.py:4161, ../hermes-agent/cli.py:5597, ../hermes-agent/cli.py:6307, internal/gateway/commands.go, internal/gateway/manager.go:617, internal/gateway/steer_command.go, internal/gateway/steer_queue_test.go, internal/kernel/frame.go:75, internal/kernel/kernel.go:370, internal/kernel/kernel.go:483, internal/kernel/kernel.go:500, internal/kernel/tool_interrupt_test.go
- Why now: P0 handoff; needs contract proof before closeout.

<!-- PROGRESS:END -->
