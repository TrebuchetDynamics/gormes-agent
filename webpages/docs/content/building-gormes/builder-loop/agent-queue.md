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

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Hermes gateway platform strict-fidelity source-pair expansion

- Phase: 2 / 2.B.12
- Owner: `docs`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Expand strict-fidelity mappings for Hermes gateway platform implementations and TUI gateway bridge files. The pass must classify platform adapters, platform helper modules, API-server platform surface, TUI gateway websocket/render/protocol bridge, and platform docs into completed channel rows, planned adapter rows, or explicit exclusions.
- Trust class: operator, system
- Ready when: `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json` is generated for the current Hermes SHA., The row uses exact Hermes files/tests as evidence, not only broad directory globs., The pass is allowed to add source-pair entries, progress source_refs, planned child rows, or explicit exclusions, but not to mark runtime behavior covered without tests.
- Not ready when: The implementation treats low-confidence taxonomy matches as proof of coverage., The implementation creates a side backlog outside progress.json or mutates hundreds of rows without feature-module grouping., The implementation copies unsupported Hermes Python/TypeScript runtime code into Gormes instead of classifying the Go contract first.
- Degraded mode: Until this strict-fidelity bucket is classified, Gormes must continue treating the matching Hermes source/docs/tests as unmapped blockers and avoid claiming complete Hermes parity for this surface.
- Fixture: `internal/channels strict-fidelity gateway platform mapping fixtures`
- Write scope: `internal/channels`, `internal/gateway`, `internal/tuigateway`, `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels ./internal/gateway ./internal/tuigateway -count=1`, `go run ./cmd/repoctl hermes-source-pairs validate`, `go run ./cmd/repoctl hermes-contract-inventory --repo-root .`
- Done signal: Strict-fidelity blockers for this bucket are classified into the canonical backlog/source-pair evidence with no side queue and no unsupported full-parity claim.
- Acceptance: The relevant Hermes files/tests no longer appear as anonymous examples in the strict-fidelity unmapped bucket; they are linked to rows, source pairs, planned child rows, explicit exclusions, or owned-divergence notes., `go run ./cmd/repoctl hermes-contract-inventory --repo-root .` regenerates JSON and Markdown with this bucket broken into actionable evidence., `go run ./cmd/repoctl hermes-source-pairs validate` passes after any source-pair edits., `go run ./cmd/progress validate` passes and generated docs show the row in the correct module.
- Source refs: hermes-agent/gateway/platforms/base.py, hermes-agent/gateway/platforms/api_server.py, hermes-agent/gateway/platforms/telegram.py, hermes-agent/gateway/platforms/yuanbao.py, hermes-agent/tui_gateway/server.py, hermes-agent/tui_gateway/render.py, internal/channels, internal/gateway
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. Hermes agent runtime strict-fidelity source-pair expansion

- Phase: 4 / 4.I
- Owner: `docs`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Expand source-pair and progress mappings for unmapped Hermes `agent` runtime files before treating them as runtime implementation gaps. The pass must classify transports, LSP helpers, context/compression helpers, prompt caching, retry/rate diagnostics, conversation loop helpers, tool dispatch helpers, and safety/redaction helpers into existing Gormes provider/runtime/tool rows or new builder rows.
- Trust class: operator, system
- Ready when: `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json` is generated for the current Hermes SHA., The row uses exact Hermes files/tests as evidence, not only broad directory globs., The pass is allowed to add source-pair entries, progress source_refs, planned child rows, or explicit exclusions, but not to mark runtime behavior covered without tests.
- Not ready when: The implementation treats low-confidence taxonomy matches as proof of coverage., The implementation creates a side backlog outside progress.json or mutates hundreds of rows without feature-module grouping., The implementation copies unsupported Hermes Python/TypeScript runtime code into Gormes instead of classifying the Go contract first.
- Degraded mode: Until this strict-fidelity bucket is classified, Gormes must continue treating the matching Hermes source/docs/tests as unmapped blockers and avoid claiming complete Hermes parity for this surface.
- Fixture: `internal/fidelity agent runtime strict-fidelity mapping fixture`
- Write scope: `internal/runtime`, `internal/provider`, `internal/tools`, `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/runtime ./internal/provider ./internal/tools -count=1`, `go run ./cmd/repoctl hermes-source-pairs validate`, `go run ./cmd/repoctl hermes-contract-inventory --repo-root .`
- Done signal: Strict-fidelity blockers for this bucket are classified into the canonical backlog/source-pair evidence with no side queue and no unsupported full-parity claim.
- Acceptance: The relevant Hermes files/tests no longer appear as anonymous examples in the strict-fidelity unmapped bucket; they are linked to rows, source pairs, planned child rows, explicit exclusions, or owned-divergence notes., `go run ./cmd/repoctl hermes-contract-inventory --repo-root .` regenerates JSON and Markdown with this bucket broken into actionable evidence., `go run ./cmd/repoctl hermes-source-pairs validate` passes after any source-pair edits., `go run ./cmd/progress validate` passes and generated docs show the row in the correct module.
- Source refs: hermes-agent/agent/conversation_loop.py, hermes-agent/agent/tool_executor.py, hermes-agent/agent/context_engine.py, hermes-agent/agent/transports/codex.py, hermes-agent/agent/transports/chat_completions.py, hermes-agent/agent/lsp/manager.py, hermes-agent/tests/agent/lsp/test_lifecycle.py, internal/runtime, internal/provider
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Hermes plugin catalog strict-fidelity classifier

- Phase: 5 / 5.I
- Owner: `docs`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Classify Hermes first-party plugin families into Gormes plugin/provider/channel/memory rows or explicit exclusions. The classifier must cover model-provider plugin manifests, memory plugins, web/browser/image/video plugins, platform plugins, Google Meet, Teams pipeline, Spotify, and plugin docs so strict fidelity can distinguish runtime gaps from catalog-only compatibility evidence.
- Trust class: operator, system
- Ready when: `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json` is generated for the current Hermes SHA., The row uses exact Hermes files/tests as evidence, not only broad directory globs., The pass is allowed to add source-pair entries, progress source_refs, planned child rows, or explicit exclusions, but not to mark runtime behavior covered without tests.
- Not ready when: The implementation treats low-confidence taxonomy matches as proof of coverage., The implementation creates a side backlog outside progress.json or mutates hundreds of rows without feature-module grouping., The implementation copies unsupported Hermes Python/TypeScript runtime code into Gormes instead of classifying the Go contract first.
- Degraded mode: Until this strict-fidelity bucket is classified, Gormes must continue treating the matching Hermes source/docs/tests as unmapped blockers and avoid claiming complete Hermes parity for this surface.
- Fixture: `internal/plugins hermes plugin catalog strict-fidelity fixtures`
- Write scope: `internal/plugins`, `internal/provider`, `internal/channels`, `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/plugins ./internal/provider ./internal/channels -count=1`, `go run ./cmd/repoctl hermes-source-pairs validate`, `go run ./cmd/repoctl hermes-contract-inventory --repo-root .`
- Done signal: Strict-fidelity blockers for this bucket are classified into the canonical backlog/source-pair evidence with no side queue and no unsupported full-parity claim.
- Acceptance: The relevant Hermes files/tests no longer appear as anonymous examples in the strict-fidelity unmapped bucket; they are linked to rows, source pairs, planned child rows, explicit exclusions, or owned-divergence notes., `go run ./cmd/repoctl hermes-contract-inventory --repo-root .` regenerates JSON and Markdown with this bucket broken into actionable evidence., `go run ./cmd/repoctl hermes-source-pairs validate` passes after any source-pair edits., `go run ./cmd/progress validate` passes and generated docs show the row in the correct module.
- Source refs: hermes-agent/plugins/model-providers/openrouter/plugin.yaml, hermes-agent/plugins/model-providers/openai-codex/plugin.yaml, hermes-agent/plugins/memory/honcho/plugin.yaml, hermes-agent/plugins/platforms/simplex/adapter.py, hermes-agent/plugins/google_meet/meet_bot.py, internal/plugins, internal/provider
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Long-term plan: profile fleet supervisor and single control-plane gateway

- Phase: 5 / 5.O
- Owner: `orchestrator`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: Define Gormes' long-term profile-fleet runtime so operators get one control surface for all named profiles while preserving Hermes-compatible profile state separation. The near-term per-profile gateway services remain a compatibility bridge; the target is a fleet supervisor that can enumerate configured profiles, start/stop/restart profile-scoped workers or a proven profile-scoped in-process equivalent, validate token ownership, surface per-profile health, and coordinate update/restart-all flows without sharing config, auth, sessions, memory, tool state, or kernels across profiles.
- Trust class: operator, gateway, system
- Ready when: The current per-profile gateway-service bridge is documented as migration/runtime compatibility, not the final operator model., Gormes-owned profile workspace/channel config and token-scoped gateway locks are available as inputs., The implementation shape chooses either isolated worker processes or a tested profile-scoped in-process runtime, with the same operator-facing fleet contract.
- Not ready when: The design treats a single gateway process as permission to reuse one GORMES_HOME, one auth store, one session DB, one memory DB, or one kernel across multiple named profiles., The slice deletes or disables the per-profile service bridge before the fleet supervisor can prove profile/token isolation and restart-all behavior., Tests require live Telegram tokens, live systemd units, or Juan's real profile directories.
- Degraded mode: If fleet supervision is unavailable, Gormes must keep the Hermes-compatible per-profile service/process bridge and report exact per-profile service state instead of collapsing profiles into the default GORMES_HOME.
- Fixture: `internal/gateway/fleet_supervisor_test.go; cmd/gormes/gateway_fleet_test.go`
- Write scope: `cmd/gormes/gateway.go`, `cmd/gormes/gateway_fleet_test.go`, `internal/gateway/fleet_supervisor.go`, `internal/gateway/fleet_supervisor_test.go`, `internal/config/agents.go`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`, `webpages/docs/content/building-gormes/modules/profiles.md`
- Test commands: `go test ./internal/gateway -run 'TestFleetSupervisor\|TestGatewayFleet' -count=1`, `go test ./cmd/gormes -run 'TestGatewayFleet' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: The profiles module documents one operator-facing fleet gateway/supervisor target, preserves profile isolation as non-negotiable, and makes the current per-profile services an explicit compatibility bridge rather than silent architecture drift.
- Acceptance: Fleet status JSON lists every configured profile with desired channels, runtime owner, version, health, last error, and token-lock evidence., Start/stop/restart-all paths operate on all configured profiles while preserving isolated GORMES_HOME, config, auth, session, memory, and tool state per profile., A duplicate Telegram token across profiles is detected and reported as a per-profile conflict rather than racing two pollers., Update/release restart hooks can target the fleet through one operator-facing command or service instead of requiring hand-managed unit names., Regression tests use fake profile roots and fake supervisors only; no live systemd, Telegram, or provider credentials are required.
- Source refs: webpages/docs/content/upstream-hermes/developer-guide/architecture.md:Profile isolation, webpages/docs/content/upstream-hermes/developer-guide/gateway-internals.md:profile-scoped process tracking, webpages/docs/content/upstream-hermes/reference/cli-commands.md:gateway --all, webpages/docs/content/upstream-hermes/reference/faq.md:multiple profiles and bot tokens, cmd/gormes/gateway.go:gatewayManagerConfig, internal/config/agents.go:AgentDefaultsCfg, internal/gateway/manager.go:ManagerConfig.ContextFilesProfile
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Hermes ui-tui strict-fidelity action matrix

- Phase: 5 / 5.Q
- Owner: `docs`
- Size: `large`
- Status: `planned`
- Priority: `P1`
- Contract: Map the unmapped Hermes `ui-tui` source and test surface into Gormes-native TUI rows, owned-divergence notes, or explicit exclusions. The matrix must cover command dispatch, viewport/history stores, RPC/gateway client events, terminal modes, clipboard/OSC52, provider/model UI, approval actions, and state isolation before the strict-fidelity report can stop treating `ui-tui` as an undifferentiated blocker bucket.
- Trust class: operator, system
- Ready when: `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json` is generated for the current Hermes SHA., The row uses exact Hermes files/tests as evidence, not only broad directory globs., The pass is allowed to add source-pair entries, progress source_refs, planned child rows, or explicit exclusions, but not to mark runtime behavior covered without tests.
- Not ready when: The implementation treats low-confidence taxonomy matches as proof of coverage., The implementation creates a side backlog outside progress.json or mutates hundreds of rows without feature-module grouping., The implementation copies unsupported Hermes Python/TypeScript runtime code into Gormes instead of classifying the Go contract first.
- Degraded mode: Until this strict-fidelity bucket is classified, Gormes must continue treating the matching Hermes source/docs/tests as unmapped blockers and avoid claiming complete Hermes parity for this surface.
- Fixture: `internal/tui hermes-ui-tui strict-fidelity matrix fixtures`
- Write scope: `internal/tui`, `internal/tuigateway`, `internal/repoctl`, `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui ./internal/tuigateway -count=1`, `go run ./cmd/repoctl hermes-contract-inventory --repo-root .`, `go run ./cmd/progress validate`
- Done signal: Strict-fidelity blockers for this bucket are classified into the canonical backlog/source-pair evidence with no side queue and no unsupported full-parity claim.
- Acceptance: The relevant Hermes files/tests no longer appear as anonymous examples in the strict-fidelity unmapped bucket; they are linked to rows, source pairs, planned child rows, explicit exclusions, or owned-divergence notes., `go run ./cmd/repoctl hermes-contract-inventory --repo-root .` regenerates JSON and Markdown with this bucket broken into actionable evidence., `go run ./cmd/repoctl hermes-source-pairs validate` passes after any source-pair edits., `go run ./cmd/progress validate` passes and generated docs show the row in the correct module.
- Source refs: hermes-agent/ui-tui/src/__tests__/slashParity.test.ts, hermes-agent/ui-tui/src/__tests__/gatewayClient.test.ts, hermes-agent/ui-tui/src/__tests__/terminalParity.test.ts, hermes-agent/ui-tui/src/__tests__/stateIsolation.test.ts, hermes-agent/ui-tui/src/__tests__/approvalAction.test.ts, internal/tui, internal/tuigateway
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Hermes web dashboard strict-fidelity contract map

- Phase: 5 / 5.Q
- Owner: `docs`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Classify Hermes `web/src` dashboard behavior into Gormes API/TUI gateway contracts, owned public-site divergence, or explicit exclusions. The map must connect chat, sessions, profiles, plugins, OAuth/provider panels, model picker, cron/admin pages, i18n, theme/plugin slots, and gateway client event shapes to Gormes runtime rows before dashboard parity is claimed.
- Trust class: operator, system
- Ready when: `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json` is generated for the current Hermes SHA., The row uses exact Hermes files/tests as evidence, not only broad directory globs., The pass is allowed to add source-pair entries, progress source_refs, planned child rows, or explicit exclusions, but not to mark runtime behavior covered without tests.
- Not ready when: The implementation treats low-confidence taxonomy matches as proof of coverage., The implementation creates a side backlog outside progress.json or mutates hundreds of rows without feature-module grouping., The implementation copies unsupported Hermes Python/TypeScript runtime code into Gormes instead of classifying the Go contract first.
- Degraded mode: Until this strict-fidelity bucket is classified, Gormes must continue treating the matching Hermes source/docs/tests as unmapped blockers and avoid claiming complete Hermes parity for this surface.
- Fixture: `internal/apiserver dashboard contract fixtures; internal/tuigateway gateway-client fixtures`
- Write scope: `internal/apiserver`, `internal/tuigateway`, `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/apiserver ./internal/tuigateway -count=1`, `go run ./cmd/repoctl hermes-contract-inventory --repo-root .`, `go run ./cmd/progress validate`
- Done signal: Strict-fidelity blockers for this bucket are classified into the canonical backlog/source-pair evidence with no side queue and no unsupported full-parity claim.
- Acceptance: The relevant Hermes files/tests no longer appear as anonymous examples in the strict-fidelity unmapped bucket; they are linked to rows, source pairs, planned child rows, explicit exclusions, or owned-divergence notes., `go run ./cmd/repoctl hermes-contract-inventory --repo-root .` regenerates JSON and Markdown with this bucket broken into actionable evidence., `go run ./cmd/repoctl hermes-source-pairs validate` passes after any source-pair edits., `go run ./cmd/progress validate` passes and generated docs show the row in the correct module.
- Source refs: hermes-agent/web/src/lib/gatewayClient.ts, hermes-agent/web/src/pages/ChatPage.tsx, hermes-agent/web/src/pages/ProfilesPage.tsx, hermes-agent/web/src/pages/PluginsPage.tsx, hermes-agent/web/src/components/ModelPickerDialog.tsx, hermes-agent/web/src/plugins/registry.ts, internal/apiserver, internal/tuigateway
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. Hermes skill catalog strict-fidelity classifier

- Phase: 6 / 6.C
- Owner: `docs`
- Size: `large`
- Status: `planned`
- Priority: `P1`
- Contract: Classify Hermes bundled and optional skill catalog files into Gormes skill-store rows, catalog-copy rows, unsupported-skill exclusions, or owned-divergence notes. The classifier must preserve SKILL.md metadata, support-file/reference layout, DESCRIPTION.md category docs, optional-skill install status, and docs-site generation boundaries without blindly copying Python-only examples into runtime packages.
- Trust class: operator, system
- Ready when: `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json` is generated for the current Hermes SHA., The row uses exact Hermes files/tests as evidence, not only broad directory globs., The pass is allowed to add source-pair entries, progress source_refs, planned child rows, or explicit exclusions, but not to mark runtime behavior covered without tests.
- Not ready when: The implementation treats low-confidence taxonomy matches as proof of coverage., The implementation creates a side backlog outside progress.json or mutates hundreds of rows without feature-module grouping., The implementation copies unsupported Hermes Python/TypeScript runtime code into Gormes instead of classifying the Go contract first.
- Degraded mode: Until this strict-fidelity bucket is classified, Gormes must continue treating the matching Hermes source/docs/tests as unmapped blockers and avoid claiming complete Hermes parity for this surface.
- Fixture: `internal/skills optional skill strict-fidelity catalog fixtures`
- Write scope: `internal/skills`, `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/skills -count=1`, `go run ./cmd/repoctl hermes-source-pairs validate`, `go run ./cmd/repoctl hermes-contract-inventory --repo-root .`
- Done signal: Strict-fidelity blockers for this bucket are classified into the canonical backlog/source-pair evidence with no side queue and no unsupported full-parity claim.
- Acceptance: The relevant Hermes files/tests no longer appear as anonymous examples in the strict-fidelity unmapped bucket; they are linked to rows, source pairs, planned child rows, explicit exclusions, or owned-divergence notes., `go run ./cmd/repoctl hermes-contract-inventory --repo-root .` regenerates JSON and Markdown with this bucket broken into actionable evidence., `go run ./cmd/repoctl hermes-source-pairs validate` passes after any source-pair edits., `go run ./cmd/progress validate` passes and generated docs show the row in the correct module.
- Source refs: hermes-agent/skills/yuanbao/SKILL.md, hermes-agent/skills/creative/popular-web-designs/SKILL.md, hermes-agent/skills/productivity/notion/SKILL.md, hermes-agent/optional-skills/research/osint-investigation/SKILL.md, hermes-agent/optional-skills/devops/pinggy-tunnel/SKILL.md, internal/skills
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. SimpleX Chat platform plugin parity

- Phase: 7 / 7.E
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Port Hermes' SimpleX Chat platform plugin into Gormes behind the shared channel adapter contract: local daemon/WebSocket configuration, allowlist admission, opaque contact IDs, DM pairing, outbound delivery, command routing, and status/degraded evidence.
- Trust class: -
- Ready when: Gateway platform manifest already classifies SimpleX as row-backed., Shared channel adapter fixtures can run without a live SimpleX daemon.
- Not ready when: The slice requires a real SimpleX account, daemon, or network socket in tests., The slice bypasses shared gateway admission/delivery abstractions.
- Degraded mode: -
- Fixture: `internal/channels/simplex fake WebSocket fixtures`
- Write scope: `internal/channels/simplex`, `internal/gateway`, `cmd/gormes/gateway.go`
- Test commands: `go test ./internal/channels/simplex ./internal/gateway -run 'SimpleX\|PlatformManifest\|Connected' -count=1`, `go run ./cmd/progress validate`
- Done signal: SimpleX fake-daemon fixtures prove config/status, inbound admission, outbound delivery, DM pairing, and command routing without live credentials.
- Acceptance: Config/status checks distinguish disabled, missing ws_url, unauthorized, and connected fake-daemon states., Inbound fake events produce normalized PlatformEvent values with opaque contact identity preserved., Outbound fake delivery and DM pairing preserve Hermes-visible SimpleX behavior and degraded errors.
- Source refs: ../hermes-agent/plugins/platforms/simplex/plugin.yaml, ../hermes-agent/plugins/platforms/simplex/adapter.py, internal/gateway/platform_manifest.go, internal/gateway/platform_connected_checkers.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Agentic-porting-kit public repo scaffold

- Phase: 8 / 8.E
- Owner: `skills`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout.
- Trust class: operator
- Ready when: Agentic-porting-kit extraction spec is complete., GitHub authentication can create or push to TrebuchetDynamics/agentic-porting-kit, or the operator has created the empty repo., The public repo name is confirmed as agentic-porting-kit or an equivalent name before the first push.
- Not ready when: No authenticated path exists to create or update the public TrebuchetDynamics repo., The builder plans to edit Gormes' repo-local skills in place instead of copied kit skills., The standalone example still requires cloning Gormes or running cmd/progress.
- Degraded mode: Without the public scaffold, the methodology remains inspectable only inside Gormes and cannot be cited or reused by other teams.
- Fixture: `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json`
- Write scope: `(separate repo) README.md`, `(separate repo) LICENSE`, `(separate repo) schemas/progress.schema.json`, `(separate repo) scripts/validate-example.sh`, `(separate repo) skills/`, `(separate repo) examples/python-greeter-to-go/`, `README.md`, `docs/content/building-gormes/strategy/success-plan.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `cd ${AGENTIC_PORTING_KIT_REPO:-../agentic-porting-kit} && ./scripts/validate-example.sh`, `go run ./cmd/progress validate`, `go test ./webpages/docs -count=1`
- Done signal: Public repo URL, standalone validation output, and Gormes backlink updates are recorded in the completed row note.
- Acceptance: Public repo exists with README.md, LICENSE, schemas/progress.schema.json, scripts/validate-example.sh, skills/, and examples/python-greeter-to-go/., README.md explains the kit independent of Gormes/Hermes and includes Codex plus Claude Code loading instructions., Each copied skill uses the porting-* name from the extraction spec and replaces hard-coded Gormes paths with target-repo variables., scripts/validate-example.sh validates the example progress file and runs the example tests without cloning Gormes., Gormes README.md and success-plan.md record the public repo URL after the repo is reachable.
- Source refs: docs/content/building-gormes/strategy/agentic-porting-kit.md, docs/content/building-gormes/strategy/success-plan.md, webpages/docs/development-skills/gormes-planner/SKILL.md, webpages/docs/development-skills/gormes-builder/SKILL.md, webpages/docs/development-skills/gormes-tdd-slice/SKILL.md, webpages/docs/development-skills/gormes-parity-auditor/SKILL.md, webpages/docs/development-skills/gormes-references/SKILL.md, webpages/docs/development-skills/gormes-skill-manager/SKILL.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
