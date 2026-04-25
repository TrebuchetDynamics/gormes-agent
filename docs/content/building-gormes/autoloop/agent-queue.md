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
## 1. Operator-auditable search evidence

- Phase: 3 / 3.E.8
- Owner: `memory`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Goncho/Honcho-compatible search surfaces operator evidence for user scope, source filters, and session lineage on every widened cross-source search hit
- Trust class: operator, system
- Ready when: Lineage-aware source-filtered search hits are validated on main., GONCHO user-scope search/context parameters and Honcho-compatible scope/source tool schemas are validated on main., The slice can be tested from seeded SQLite/Bolt fixtures without provider calls or external Honcho services.
- Not ready when: The slice changes search ranking, widens default same-chat recall, implements Honcho v3 HTTP/SDK parity, or adds new filters beyond reporting existing scope/source/lineage evidence.
- Degraded mode: Search output reports missing session-directory evidence, same-chat fallback, unavailable lineage, orphaned lineage, and source allowlists instead of implying cross-source recall was safe by default.
- Fixture: `cmd/gormes/goncho_search_evidence_test.go`
- Write scope: `cmd/gormes/goncho.go`, `cmd/gormes/goncho_search_evidence_test.go`, `internal/goncho/types.go`, `internal/goncho/service.go`, `internal/goncho/service_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes -run TestGonchoDoctorSearchEvidence -count=1`, `go test ./internal/goncho -run TestServiceSearch -count=1`, `go test ./internal/memory -run TestSessionCatalog_SearchHitsIncludeLineageContext -count=1`
- Done signal: Goncho doctor fixtures prove cross-source search evidence names source filters, user-scope decisions, and lineage status for widened hits without changing default recall fences.
- Acceptance: `gormes goncho doctor --peer <user> --session <source:chat> --scope user --sources <source>` text output includes scope decision, source allowlist, sessions considered, widened session count, search hit source/origin_source/session_key, and lineage status for each hit., JSON output includes the same search evidence fields so autoloop and docs tests can assert them without parsing text., Search evidence reports orphan/unavailable lineage explicitly and does not widen recall when scope=user is denied or session directory evidence is missing.
- Source refs: internal/memory/session_catalog.go, internal/memory/session_lineage_search_test.go, internal/goncho/service.go, cmd/gormes/goncho.go, docs/content/building-gormes/goncho_honcho_memory/04-agent-work-packets.md, ../honcho/docs/v3/documentation/features/advanced/search.mdx@e659b6b, ../honcho/docs/v3/documentation/features/advanced/using-filters.mdx@e659b6b
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. Per-turn model selection

- Phase: 4 / 4.D
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Kernel accepts a per-turn model override, sends it on exactly one hermes.ChatRequest, exposes the active model in RenderFrame during that turn, and falls back to the resident config model on the next turn
- Trust class: operator, gateway, system
- Ready when: Routing policy and fallback selector is validated on main., Kernel tests can use internal/hermes.MockClient to inspect ChatRequest.Model without live provider credentials., Current main has no PlatformEvent.Model field and still builds ChatRequest with k.cfg.Model, so the first test must be RED before implementation.
- Not ready when: The slice reintroduces Config.ModelSelector, changes provider routing policy, makes provider network calls, mutates cfg.Model globally, or wires context compression history pruning.
- Degraded mode: If no per-turn model is supplied or the override is blank, the kernel continues using cfg.Model and status frames make the resident model explicit.
- Fixture: `internal/kernel/per_turn_model_test.go`
- Write scope: `internal/kernel/frame.go`, `internal/kernel/kernel.go`, `internal/kernel/per_turn_model_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/kernel -run TestPerTurnModel -count=1`, `go test ./internal/kernel -count=1`
- Done signal: internal/kernel/per_turn_model_test.go proves override, fallback, and frame-model behavior without mutating cfg.Model or provider routing.
- Acceptance: A PlatformEventSubmit with Model="override-model" causes the outbound hermes.ChatRequest.Model to be override-model for that turn., A following PlatformEventSubmit with no Model uses the original cfg.Model, proving the override did not mutate resident kernel configuration., RenderFrame.Model reflects the active override while the turn is connecting/streaming and returns to cfg.Model after the turn settles., The test stays in internal/kernel with MockClient and does not add provider calls or routing selector changes.
- Source refs: internal/kernel/frame.go, internal/kernel/kernel.go, internal/hermes/model_routing.go, docs/content/upstream-hermes/user-guide/features/provider-routing.md, ../hermes-agent/run_agent.py@5401a008
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

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
