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

Shared unattended-loop facts live in [Autoloop Handoff](../autoloop-handoff/):
the main entrypoint, orchestrator plan, candidate source, generated docs,
tests, and candidate policy. Keep those control-plane facts in
`meta.autoloop`, and keep row-specific execution facts in `progress.json`.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Pairing read-model schema + atomic persistence

- Phase: 2 / 2.F.3
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Contract: Gateway pairing state is persisted as a Go-native XDG read model with atomic writes, deterministic pending/approved ordering, and per-platform paired/unpaired status before approval UX
- Trust class: gateway, operator
- Ready when: The shared gateway package already owns RuntimeStatusStore and can add a pairing read model without live adapter startup., No approval-code generation or unauthorized-DM response behavior is needed for this first persistence slice.
- Not ready when: The implementation wires Telegram/Discord/Slack pairing UX, generates codes, or starts channel transports instead of only freezing the persisted read model.
- Degraded mode: Gateway status reports missing, corrupt, or permission-denied pairing state without starting transports or accepting unknown users.
- Fixture: `internal/gateway/pairing_store_test.go`
- Write scope: `internal/gateway/pairing_store.go`, `internal/gateway/pairing_store_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway -run TestPairingStore -count=1`, `go run ./cmd/autoloop progress validate`
- Done signal: Pairing store fixtures prove atomic persistence, deterministic readout, corrupt-state degradation, and per-platform pending/approved separation without code generation or adapter UX.
- Acceptance: Pairing store fixtures create pending and approved user records per platform under an XDG-controlled state root., Writes use temp-file plus atomic rename, preserve valid old state on failed writes, and apply owner-only file mode where the platform supports chmod., Corrupt JSON and missing files return structured degraded evidence and deterministic empty read models., List output is sorted by platform, user_id, and code age so docs/status tests do not depend on map iteration order.
- Source refs: ../hermes-agent/gateway/pairing.py@5401a008, ../hermes-agent/tests/gateway/test_pairing.py@5401a008, internal/gateway/status.go, docs/content/building-gormes/architecture_plan/phase-2-gateway.md
- Unblocks: Pairing approval + rate-limit semantics, `gormes gateway status` read-only command
- Why now: Unblocks Pairing approval + rate-limit semantics, `gormes gateway status` read-only command.

## 2. Goncho dreaming scheduler contract

- Phase: 3 / 3.F
- Owner: `memory`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: Goncho models Honcho dream scheduling as local auditable work intent with cooldown, idle, threshold, and dedupe gates before any LLM dream execution
- Trust class: operator, system
- Ready when: Goncho configuration namespace, queue status read model, and directional representation scope fixtures are validated on main., The first slice only records dream work intent and degraded evidence; it does not need a model provider, worker daemon, or surprisal tree.
- Not ready when: The slice runs LLM deduction or induction, implements tree-based surprisal, starts background workers, or calls a hosted Honcho service.
- Degraded mode: When dreaming is disabled or no scheduler table exists, Goncho context and doctor output report dream_disabled or dream_unavailable evidence rather than implying background reasoning is active.
- Fixture: `internal/goncho/dream_scheduler_test.go`
- Write scope: `internal/goncho/`, `internal/memory/`, `internal/config/`, `cmd/gormes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/goncho ./internal/memory ./internal/config ./cmd/gormes -run TestGonchoDream -count=1`, `go run ./cmd/autoloop progress validate`
- Done signal: Dream scheduler fixtures prove threshold, cooldown, idle, dedupe, manual scheduling, cancellation, and doctor/status degraded evidence without LLM execution.
- Acceptance: Scheduler fixtures create at most one pending or in-progress dream per workspace/observer/observed tuple., A dream is eligible only after at least 50 new conclusions since the last dream, an eight-hour cooldown, and the configured idle timeout., New activity cancels or marks stale a pending dream for the observed peer without deleting prior dream history., Manual schedule requests are deduped and report whether they created, reused, or rejected a dream intent., Doctor/status output exposes dream_disabled, dream_pending, dream_in_progress, and dream_cooldown evidence without waiting for queue emptiness.
- Source refs: ../honcho/docs/v3/documentation/features/advanced/dreaming.mdx@e659b6b, ../honcho/docs/v3/documentation/core-concepts/reasoning.mdx@e659b6b, ../honcho/docs/v3/contributing/configuration.mdx@e659b6b, docs/content/building-gormes/goncho_honcho_memory/03-honcho-docs-study.md, docs/content/building-gormes/goncho_honcho_memory/04-agent-work-packets.md, internal/goncho/service.go, internal/config/config.go
- Unblocks: Goncho dream execution worker, Goncho dream status documentation
- Why now: Unblocks Goncho dream execution worker, Goncho dream status documentation.

## 3. BlueBubbles iMessage bubble formatting parity

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
