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
autonomous implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared unattended-loop facts live in [Builder Loop Handoff](../builder-loop-handoff/):
the main entrypoint, orchestrator plan, candidate source, generated docs,
tests, and candidate policy. Keep those control-plane facts in
`meta.builder_loop`, and keep row-specific execution facts in `progress.json`.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Gateway fresh-final eligibility helper

- Phase: 2 / 2.B.5
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: Gateway coalescer exposes a deterministic fresh-final eligibility decision for stale editable previews without changing send/edit behavior yet
- Trust class: operator, gateway, system
- Ready when: Gateway stream consumer for agent-event fan-out and Non-editable gateway progress/commentary send fallback are complete on main., internal/gateway/coalesce.go already owns pending message id, last edit time, final flush, and the editable preview lifecycle., The worker can add a fake-clock eligibility helper in internal/gateway without touching manager wiring, Telegram config, SDK delete calls, or provider streaming.
- Not ready when: The slice changes manager outbound dispatch, adds a Send/Delete path, or edits Telegram/config packages., The slice changes provider streaming, kernel.RenderFrame phases, non-editable channel send fallback, or Slack/Discord channel implementations., The tests sleep or depend on wall-clock time instead of an injected now function.
- Degraded mode: Until this helper lands, fresh-final cannot be tested with a fake clock and all channels keep the legacy edit-in-place finalization path.
- Fixture: `internal/gateway/coalesce_fresh_final_test.go::TestFreshFinalEligibility`
- Write scope: `internal/gateway/coalesce.go`, `internal/gateway/coalesce_fresh_final_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway -run 'TestFreshFinalEligibility\|TestCoalescer' -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: Gateway coalescer fixtures prove fresh-final eligibility with a fake clock while all final sends still use the existing edit-in-place path.
- Acceptance: Coalescer state records the preview creation time when SendPlaceholder succeeds and keeps zero-value state for no-preview cases., A pure helper or coalescer method returns false when the threshold is zero, the preview id is empty, preview creation time is missing, or the preview is younger than the threshold., The same helper returns true when the preview age is equal to or greater than the threshold., Focused tests use a fake now function and do not sleep., Existing coalescer edit/finalize behavior remains unchanged because this row does not add the fresh send path.
- Source refs: ../hermes-agent/gateway/stream_consumer.py@b16f9d43:GatewayStreamConsumer._should_send_fresh_final, ../hermes-agent/tests/gateway/test_stream_consumer_fresh_final.py@b16f9d43:test_disabled_by_default_still_edits_in_place, ../hermes-agent/tests/gateway/test_stream_consumer_fresh_final.py@b16f9d43:test_short_lived_preview_edits_in_place, ../hermes-agent/tests/gateway/test_stream_consumer_fresh_final.py@b16f9d43:test_long_lived_preview_sends_fresh_final, ../hermes-agent/tests/gateway/test_stream_consumer_fresh_final.py@b16f9d43:test_no_edit_sentinel_is_not_affected, internal/gateway/coalesce.go, internal/gateway/channel.go
- Unblocks: Gateway fresh-final send/delete fallback
- Why now: P0 handoff; needs contract proof before closeout.

## 2. BlueBubbles iMessage bubble formatting parity

- Phase: 7 / 7.E
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes
- Trust class: gateway, system
- Ready when: The first-pass BlueBubbles adapter already owns Send, markdown stripping, cached GUID resolution, and home-channel fallback in internal/channels/bluebubbles.
- Not ready when: The slice attempts to add live BlueBubbles HTTP/webhook registration, attachment download, reactions, typing indicators, or edit-message support.
- Degraded mode: BlueBubbles remains a usable first-pass adapter, but long replies may still arrive as one stripped text send until paragraph splitting and suffix-free chunking are fixture-locked.
- Fixture: `internal/channels/bluebubbles/bot_test.go`
- Write scope: `internal/channels/bluebubbles/bot.go`, `internal/channels/bluebubbles/bot_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels/bluebubbles -count=1`
- Done signal: BlueBubbles adapter tests prove paragraph-to-bubble sends, suffix-free chunking, and no edit/placeholder capability.
- Acceptance: Send splits blank-line-separated paragraphs into separate SendText calls while preserving existing chat GUID resolution and home-channel fallback., Long paragraph chunks omit `(n/m)` pagination suffixes and concatenate back to the stripped original text., Bot does not implement gateway.MessageEditor or gateway.PlaceholderCapable, preserving non-editable iMessage semantics.
- Source refs: ../hermes-agent/gateway/platforms/bluebubbles.py@f731c2c2, ../hermes-agent/tests/gateway/test_bluebubbles.py@f731c2c2, internal/channels/bluebubbles/bot.go, internal/gateway/channel.go
- Unblocks: BlueBubbles iMessage session-context prompt guidance
- Why now: Unblocks BlueBubbles iMessage session-context prompt guidance.

<!-- PROGRESS:END -->
