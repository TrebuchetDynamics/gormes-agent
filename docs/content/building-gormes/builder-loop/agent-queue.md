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
## 1. Home channel ownership resolver fixtures

- Phase: 2 / 2.F.4
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: Add a channel-neutral home-channel ownership resolver for platform-only delivery targets. The resolver must prefer an explicit target chat/thread, then a Hermes-compatible per-platform home_channel.chat_id/thread/name setting bridged through Gormes config, then a discovery/pairing-owned source only when discovery is explicitly enabled for that platform. It must be callable by delivery routing without Telegram-specific branches and must preserve explicit endpoint/source routing semantics.
- Trust class: operator, gateway
- Ready when: The builder restates the Hermes parity contract and confirms no dependency on hermes-agent runtime services before editing., Upstream Hermes refs are available at gateway/config.py HomeChannel/PlatformConfig and gateway/delivery.py DeliveryTarget.parse platform-name semantics., Current Gormes refs are available at internal/gateway/delivery.go ParseDeliveryTarget, internal/gateway/session_context.go SessionSource, internal/config/config.go platform/home-channel config bridge, and cmd/gormes/gateway.go manager channel registration., The slice is implemented through shared gateway/config code and fixture channels only; no live Telegram, Slack, Discord, WhatsApp, BlueBubbles, or future channel adapter is required.
- Not ready when: The implementation hard-codes Telegram, assumes one global chat id for all platforms, or bypasses shared DeliveryTarget/SessionSource routing., The implementation prints or commits secret-bearing Hermes profile values instead of using redacted/temp fixtures., The implementation expands notify-to fanout, channel-directory persistence, or periodic refresh before the home-channel resolver contract is tested.
- Degraded mode: If a platform has no configured home channel and first-run discovery has not supplied one, resolution must return a structured missing_home_channel error and leave delivery at local/origin fallback; it must not guess a chat, leak secrets, or special-case Telegram.
- Fixture: `internal/gateway/home_channel_resolver_test.go with temp config structs and fake SessionSource records only; no live platform SDK or Hermes runtime service.`
- Write scope: `internal/gateway/delivery.go`, `internal/gateway/home_channel_resolver_test.go`, `internal/config/config.go`, `internal/config/config_test.go`, `cmd/gormes/gateway.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway ./internal/config ./cmd/gormes -run 'HomeChannel\|DeliveryTarget\|Gateway.*Allowed\|ResolveHome' -count=1`, `go test ./internal/gateway ./internal/config ./cmd/gormes -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Home-channel resolver fixtures prove explicit override precedence, configured home_channel ownership, first-run discovery fallback, and missing-home-channel degradation without live platform SDKs., Progress evidence records that the next builder can implement notify-to fanout on top of a tested shared home-channel resolver.
- Acceptance: Platform-only targets such as telegram, slack, discord, whatsapp, bluebubbles, and future channel names resolve through one shared home-channel lookup API rather than channel-specific branches., Explicit platform:chat_id[:thread_id] targets remain explicit and bypass home-channel ownership lookup., Configured Hermes/Gormes home_channel.chat_id wins over first-run discovery fallback for that platform., Missing home-channel evidence returns a structured, user-safe error/fallback reason suitable for /status and cron delivery diagnostics.
- Source refs: docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#phase-2--gateway-and-platform-runtime, ../hermes-agent/gateway/config.py:HomeChannel, ../hermes-agent/gateway/config.py:PlatformConfig.home_channel, ../hermes-agent/gateway/delivery.py:DeliveryTarget.parse, internal/gateway/delivery.go:ParseDeliveryTarget, internal/gateway/session_context.go:SessionSource, internal/config/config.go:TelegramCfg, internal/config/config.go:DiscordCfg, cmd/gormes/gateway.go:runGateway
- Unblocks: Notify-to delivery routing, Channel directory atomic persistence + lookup, Manager remember-source hook
- Why now: P0 handoff; needs contract proof before closeout.

<!-- PROGRESS:END -->
