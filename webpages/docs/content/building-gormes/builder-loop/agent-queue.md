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
## 1. Hermes tool tail strict-fidelity source-pair expansion

- Phase: 5 / 5.A
- Owner: `docs`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Classify remaining unmapped Hermes tools into covered Gormes tool rows, focused builder rows, or explicit exclusions. The pass must cover web/search providers, voice/TTS/STT tools, video/image tools, environment backends, tool result storage, process/zombie guards, URL and website policy helpers, and x_search auth behavior without hiding them behind the existing broad 61-tool row.
- Trust class: operator, system
- Ready when: `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json` is generated for the current Hermes SHA., The row uses exact Hermes files/tests as evidence, not only broad directory globs., The pass is allowed to add source-pair entries, progress source_refs, planned child rows, or explicit exclusions, but not to mark runtime behavior covered without tests.
- Not ready when: The implementation treats low-confidence taxonomy matches as proof of coverage., The implementation creates a side backlog outside progress.json or mutates hundreds of rows without feature-module grouping., The implementation copies unsupported Hermes Python/TypeScript runtime code into Gormes instead of classifying the Go contract first.
- Degraded mode: Until this strict-fidelity bucket is classified, Gormes must continue treating the matching Hermes source/docs/tests as unmapped blockers and avoid claiming complete Hermes parity for this surface.
- Fixture: `internal/tools strict-fidelity tail mapping fixtures`
- Write scope: `internal/tools`, `internal/tools/safety`, `internal/tools/budget`, `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools ./internal/tools/safety ./internal/tools/budget -count=1`, `go run ./cmd/repoctl hermes-source-pairs validate`, `go run ./cmd/repoctl hermes-contract-inventory --repo-root .`
- Done signal: Strict-fidelity blockers for this bucket are classified into the canonical backlog/source-pair evidence with no side queue and no unsupported full-parity claim.
- Acceptance: The relevant Hermes files/tests no longer appear as anonymous examples in the strict-fidelity unmapped bucket; they are linked to rows, source pairs, planned child rows, explicit exclusions, or owned-divergence notes., `go run ./cmd/repoctl hermes-contract-inventory --repo-root .` regenerates JSON and Markdown with this bucket broken into actionable evidence., `go run ./cmd/repoctl hermes-source-pairs validate` passes after any source-pair edits., `go run ./cmd/progress validate` passes and generated docs show the row in the correct module.
- Source refs: hermes-agent/tools/x_search_tool.py, hermes-agent/tools/web_tools.py, hermes-agent/tools/tts_tool.py, hermes-agent/tools/transcription_tools.py, hermes-agent/tools/video_generation_tool.py, hermes-agent/tools/environments/vercel_sandbox.py, hermes-agent/tests/tools/test_x_search_tool.py, internal/tools
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. Strict-fidelity upstream test-suite classifier

- Phase: 8 / 8.C
- Owner: `docs`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Classify the strict-fidelity test blockers by upstream suite, exact Hermes source under test, current Gormes progress row or explicit exclusion. The row must turn the 1,206 unmapped upstream test files from one giant blocker count into deterministic report groups that builders can chase without creating a side backlog.
- Trust class: operator, system
- Ready when: `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json` is generated for the current Hermes SHA., The row uses exact Hermes files/tests as evidence, not only broad directory globs., The pass is allowed to add source-pair entries, progress source_refs, planned child rows, or explicit exclusions, but not to mark runtime behavior covered without tests.
- Not ready when: The implementation treats low-confidence taxonomy matches as proof of coverage., The implementation creates a side backlog outside progress.json or mutates hundreds of rows without feature-module grouping., The implementation copies unsupported Hermes Python/TypeScript runtime code into Gormes instead of classifying the Go contract first.
- Degraded mode: Until this strict-fidelity bucket is classified, Gormes must continue treating the matching Hermes source/docs/tests as unmapped blockers and avoid claiming complete Hermes parity for this surface.
- Fixture: `internal/repoctl/hermes_contract_inventory_test.go:TestWriteHermesContractInventoryWritesJSONAndMarkdown; webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md`
- Write scope: `internal/fidelity`, `internal/repoctl`, `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json`, `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md`
- Test commands: `go test ./internal/fidelity ./internal/repoctl -run 'TestHermesReport\|TestWriteHermesContractInventory' -count=1`, `go run ./cmd/repoctl hermes-contract-inventory --repo-root .`, `go run ./cmd/progress validate`
- Done signal: Strict-fidelity blockers for this bucket are classified into the canonical backlog/source-pair evidence with no side queue and no unsupported full-parity claim.
- Acceptance: The relevant Hermes files/tests no longer appear as anonymous examples in the strict-fidelity unmapped bucket; they are linked to rows, source pairs, planned child rows, explicit exclusions, or owned-divergence notes., `go run ./cmd/repoctl hermes-contract-inventory --repo-root .` regenerates JSON and Markdown with this bucket broken into actionable evidence., `go run ./cmd/repoctl hermes-source-pairs validate` passes after any source-pair edits., `go run ./cmd/progress validate` passes and generated docs show the row in the correct module.
- Source refs: webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json, internal/fidelity/report.go:buildUnmappedUpstreamInventory, internal/repoctl/hermes_contract_inventory.go:RenderHermesContractInventoryMarkdown, hermes-agent/tests/agent/lsp/test_workspace.py, hermes-agent/tests/tools/test_x_search_tool.py, hermes-agent/ui-tui/src/__tests__/slashParity.test.ts
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Hermes gateway platform strict-fidelity source-pair expansion

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

## 4. Gateway delivery evidence in operator run report

- Phase: 2 / 2.F.4
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Join gateway/cron delivery results into OperatorRunReport artifacts so unattended jobs record target platform, chat/thread identity, delivery path, delivered boolean, fallback path, and stable failure evidence for live adapter, standalone sender, and fallback sink paths. The slice must reuse existing cron delivery planning/outcome evidence and never require a live Telegram/Discord/Slack bot in tests.
- Trust class: operator, system
- Ready when: Durable operator run report for unattended jobs is complete., Existing cron delivery plan and delivery outcome fixtures cover live, standalone, fallback, local, and parse-failed paths.
- Not ready when: Tests require real channel credentials, live gateway processes, or network sends., Delivery evidence is only written to logs and not joined into the report artifact., The implementation treats local-only delivery as failure instead of an explicit local target.
- Degraded mode: If every delivery path fails, the report remains written with delivered=false, channel_degraded evidence, redacted target detail, and a recommended gateway status command.
- Fixture: `internal/cron/operator_delivery_report_test.go::TestOperatorRunReportIncludesDeliveryEvidence`
- Write scope: `internal/cron/operator_run_report.go`, `internal/cron/operator_delivery_report_test.go`, `internal/cron/delivery_plan.go`, `cmd/gormes/gateway_status.go`
- Test commands: `go test ./internal/cron -run 'TestOperatorRunReportIncludesDeliveryEvidence\|TestPlanCronDelivery\|TestCronDelivery' -count=1`, `go test ./cmd/gormes -run 'TestGatewayStatus' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Builder reports report fixtures for live/standalone/fallback/local delivery evidence, failed-delivery degradation, normalized targets, and gateway repair guidance.
- Acceptance: Report fixtures include delivered=true for successful live/standalone/fallback paths and delivered=false with stable evidence for adapter unavailable/failure paths., Target values are normalized and redacted consistently with existing delivery target rules., Gateway status recommended command appears when delivery fails or runtime status is missing., Existing cron delivery plan tests still pass.
- Source refs: internal/cron/delivery_plan.go, internal/cron/executor.go, internal/gateway/status.go, cmd/gormes/gateway_status.go, cmd/gormes/send.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Hermes agent runtime strict-fidelity source-pair expansion

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

## 6. Hermes plugin catalog strict-fidelity classifier

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

## 7. Hermes LSP write-time semantic diagnostics

- Phase: 5 / 5.L
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: After `write_file` or `patch`, Gormes runs a language-server diagnostic pass equivalent to Hermes' write-time LSP surface, shifts baseline ranges through edits, and returns new semantic errors to the agent without blocking unsupported languages.
- Trust class: -
- Ready when: Native write/patch tools and lint-delta rows are complete., A fake diagnostic service can be injected without launching real language servers.
- Not ready when: The slice shells out to real language servers in unit tests., Unsupported languages fail the file operation instead of returning degraded diagnostic evidence.
- Degraded mode: -
- Fixture: `internal/tools LSP diagnostic fake-server fixtures`
- Write scope: `internal/tools`, `internal/lsp`
- Test commands: `go test ./internal/tools -run 'Test.*LSP\|Test.*Diagnostic\|TestWrite\|TestPatch' -count=1`, `go test ./internal/lsp -count=1`, `go run ./cmd/progress validate`
- Done signal: File write/patch fixtures prove LSP diagnostic deltas, range shifting, and graceful unsupported-language degradation.
- Acceptance: Post-write diagnostics report only new or shifted errors relevant to the edited file., Range-shift fixtures cover insert/delete/move edits and preserve baseline diagnostic identity., Unsupported or missing LSP backends return degraded evidence without failing successful file writes.
- Source refs: ../hermes-agent/agent/lsp/manager.py, ../hermes-agent/agent/lsp/range_shift.py, ../hermes-agent/tests/agent/lsp/test_delta_key.py, ../hermes-agent/tests/agent/lsp/test_service.py, internal/tools/file_task_tools.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Long-term plan: profile fleet supervisor and single control-plane gateway

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

## 9. Hermes ui-tui strict-fidelity action matrix

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

## 10. Hermes web dashboard strict-fidelity contract map

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

<!-- PROGRESS:END -->
