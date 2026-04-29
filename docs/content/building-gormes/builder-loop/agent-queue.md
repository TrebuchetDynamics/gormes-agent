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
## 1. Gateway-handled slash commands bypass active-session guard

- Phase: 2 / 2.F.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: Active-turn command bypass contract
- Trust class: operator, gateway
- Ready when: CommandRegistry and cli.CommandRegistry already classify ported gateway slash commands with active-turn policies., The channel-neutral TurnAdapter path is validated, so tests can exercise active-turn bypass with fixture channels and no live Telegram, Slack, Discord, WhatsApp, or BlueBubbles dependency., Mid-run steer injection and queued steer fallback are validated, so /steer can remain a safe bypass/drain command without interrupting the running turn.
- Not ready when: The implementation adds a Telegram-only active-turn bypass instead of using shared gateway command policy., Unknown, unavailable, CLI-only, or busy_reject commands can leak into the model prompt while a turn is active., The slice changes provider/runtime behavior or starts, shells out to, imports, or requires hermes-agent runtime services.
- Degraded mode: If a command is unknown, unavailable, or marked busy_reject while a turn is active, the gateway returns the existing sanitized busy/unavailable evidence and does not enqueue the slash text into the running model turn.
- Fixture: `internal/gateway/active_turn_command_bypass_test.go`
- Write scope: `internal/gateway/`, `internal/cli/command_registry.go`, `internal/cli/*command*_test.go`, `internal/channels/fakes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway ./internal/cli ./internal/channels/... -run 'ActiveTurn\|CommandBypass\|SlashCommand\|TurnAdapter' -count=1`, `go test ./internal/gateway ./internal/cli ./internal/channels/... -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Gateway active-turn fixture tests prove safe registry commands bypass the active-session guard across channel-neutral adapters while denied/unknown commands remain sanitized and do not reach the model prompt.
- Acceptance: While a session has an active turn, registry-resolved gateway commands with bypass/drain policy such as /help, /stop, /usage, and /steer dispatch through their gateway handlers immediately instead of being rejected by the active-session guard., Busy-reject mutators such as /new, unavailable commands such as /status, and unknown slash commands still return structured safe evidence and never enter the model prompt or queued turn text., The behavior is proven through the shared gateway/turn adapter seam with both a Telegram-shaped fixture and a non-Telegram fixture so channel parity is preserved.
- Source refs: docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#gateway--platform-control, docs/content/upstream-hermes/reference/slash-commands.md, internal/gateway/commands.go, internal/cli/command_registry.go, internal/gateway/turn_adapter.go
- Why now: P0 handoff; needs contract proof before closeout.

<!-- PROGRESS:END -->
