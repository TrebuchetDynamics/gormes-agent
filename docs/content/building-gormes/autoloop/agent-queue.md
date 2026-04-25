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
## 1. Durable pause/resume intent contract

- Phase: 2 / 2.E.3
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Gormes durable jobs record pause and resume as explicit operator/system control intents over the SQLite ledger without starting a GBrain worker daemon
- Trust class: operator, system
- Ready when: Durable worker supervisor status is complete and the SQLite ledger can already report worker liveness, cancellation intent, and replay availability.
- Not ready when: The slice starts external workers, implements shell execution, or ports GBrain Postgres/PGLite queue internals instead of only freezing pause/resume state transitions.
- Degraded mode: Doctor/status reports paused jobs, resume intent, and unsupported child-agent lifecycle control instead of silently leaving workers active.
- Fixture: `internal/subagent/durable_lifecycle_test.go`
- Write scope: `internal/subagent/`, `internal/doctor/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/subagent ./internal/doctor -count=1`
- Done signal: Durable lifecycle fixtures prove pause/resume state transitions, trust denial, and doctor/status counts over the Go SQLite ledger.
- Acceptance: Pause transitions waiting or active durable jobs into an observable paused state with actor, reason, and timestamp evidence., Resume returns paused jobs to the correct claimable state without losing progress_json, parent_id, timeout_at, or lock audit fields., Child-agent trust cannot pause or resume privileged deterministic shell/cron jobs., Doctor/status includes paused and resume-pending counts plus degraded evidence when lifecycle control is unsupported.
- Source refs: ../gbrain/skills/minion-orchestrator/SKILL.md@11abb24, ../gbrain/skills/conventions/subagent-routing.md@11abb24, ../gbrain/src/core/minions/queue.ts, internal/subagent/durable_ledger.go, internal/doctor/durable_ledger.go
- Unblocks: Durable replay and inbox message contract
- Why now: Unblocks Durable replay and inbox message contract.

## 2. SillyTavern persona and group-chat mapping fixtures

- Phase: 3 / 3.E.7
- Owner: `memory`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: Goncho maps Honcho SillyTavern peer modes, session naming, enrichment modes, and group-chat participants without widening recall or leaking the internal goncho name
- Trust class: operator, system
- Ready when: Honcho host integration compatibility fixtures already map baseline SillyTavern chat-instance sessions, recall modes, host-scoped config, and honcho_* external tool names.
- Not ready when: The slice adds a SillyTavern extension, imports Node/browser code, calls hosted Honcho, or changes Goncho persistence instead of fixture-locking host mapping decisions.
- Degraded mode: Host mapping evidence reports unsupported SillyTavern panel knobs or missing peer/session identifiers instead of silently merging personas or group characters.
- Fixture: `internal/goncho/sillytavern_mapping_test.go`
- Write scope: `internal/goncho/`, `internal/gonchotools/`, `docs/content/building-gormes/goncho_honcho_memory/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/goncho ./internal/gonchotools -count=1`
- Done signal: SillyTavern mapping fixtures prove persona/session/group-chat decisions and enrichment-mode degradation while external tools keep honcho_* names.
- Acceptance: Peer mode fixtures distinguish one shared user peer from separate persona-scoped peers without changing the default Goncho workspace., Session naming fixtures cover auto per-chat, per-character, and custom session names, including reset/orphan evidence for new chats., Group-chat fixtures register one peer per character and lazy-add mid-chat characters instead of collapsing to one group peer., Context-only, reasoning, and tool-call enrichment modes map to prompt context, honcho_chat-style reasoning, and tool exposure with explicit degraded evidence for unsupported modes.
- Source refs: ../honcho/docs/v3/guides/integrations/sillytavern.mdx@e659b6b, ../honcho/docs/docs.json@e659b6b, internal/goncho/host_integration.go, internal/goncho/host_integration_test.go, docs/content/building-gormes/goncho_honcho_memory/05-operator-playbook.md
- Unblocks: Cross-chat operator evidence
- Why now: Unblocks Cross-chat operator evidence.

## 3. Plugin SDK

- Phase: 5 / 5.I
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Gormes loads plugin manifests, capability metadata, version constraints, and disabled-state evidence without executing plugin runtime code
- Trust class: operator, system
- Ready when: The existing Go tool registry and dashboard status endpoint can expose disabled capability rows without invoking plugin code.
- Not ready when: The slice executes Python, JavaScript, or arbitrary plugin hooks; serves plugin assets; or registers Spotify/tool handlers before manifest validation and disabled-state reporting are fixture-locked.
- Degraded mode: Plugin status reports malformed manifests, unsupported capability kinds, missing credentials, and disabled execution before any tool or dashboard route is registered.
- Fixture: `internal/plugins/manifest_test.go`
- Write scope: `internal/plugins/`, `internal/tools/`, `internal/apiserver/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/plugins ./internal/tools ./internal/apiserver -count=1`
- Done signal: Plugin manifest fixtures prove metadata parsing, version/capability validation, disabled inventory, and no implicit runtime code execution.
- Acceptance: Manifest fixtures parse name, version, label/description, capability declarations, required env/auth, dashboard manifest entry/css/api, tab override, tab hidden, and slot declarations., Invalid names, unsupported capability kinds, missing required fields, and incompatible version constraints fail closed with structured status evidence., Capability registration can produce disabled tool/dashboard/backend-route inventory without executing plugin code., Project plugin discovery remains disabled unless an explicit config/env gate is present.
- Source refs: ../hermes-agent/plugins/spotify/plugin.yaml, ../hermes-agent/plugins/disk-cleanup/plugin.yaml, ../hermes-agent/plugins/example-dashboard/dashboard/manifest.json, ../hermes-agent/plugins/strike-freedom-cockpit/dashboard/manifest.json, ../hermes-agent/website/docs/user-guide/features/extending-the-dashboard.md@e5647d78, ../hermes-agent/hermes_cli/web_server.py, internal/tools/registry.go
- Unblocks: Dashboard theme/plugin extension status contract, First-party Spotify plugin fixture
- Why now: Unblocks Dashboard theme/plugin extension status contract, First-party Spotify plugin fixture.

## 4. BlueBubbles iMessage bubble formatting parity

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
