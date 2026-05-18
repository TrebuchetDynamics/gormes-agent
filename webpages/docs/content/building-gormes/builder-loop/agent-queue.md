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
## 1. Durable operator run report for unattended jobs

- Phase: 2 / 2.D
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Add a Gormes-owned durable OperatorRunReport artifact for unattended cron/fleet jobs. The report is produced from existing cron run, provider/runtime readiness, delivery, session, and release-ledger evidence and records job_id, run_id, profile/workspace, provider/model, delivery target, start/end/status, degraded_reason, transcript/session refs, redacted error/log summary, and recommended_next_command without running a real provider, gateway, or scheduler loop.
- Trust class: operator, system
- Ready when: Cron run audit records, delivery planning evidence, and runtime binding status evidence already exist., The first slice can build reports from hermetic fixtures without live providers, gateways, schedulers, or network access., The report schema is scoped to unattended operator jobs and does not replace existing cron_runs storage.
- Not ready when: The implementation tries to run real scheduled jobs, call providers, send gateway messages, or introduce email/CRM integrations., The report is tied only to Telegram/gateway delivery and cannot represent local or suppressed cron runs., Errors are stored only as prose without stable status/degraded_reason/recommended_next_command fields.
- Degraded mode: When a run is suppressed, times out, fails provider/auth readiness, or cannot deliver, the report remains written with status=degraded or failed, a stable degraded_reason, redacted detail, and a recommended repair command rather than disappearing into logs.
- Fixture: `internal/cron/operator_run_report_test.go::TestOperatorRunReportBuildsSuccessAndDegradedArtifacts`
- Write scope: `internal/cron/operator_run_report.go`, `internal/cron/operator_run_report_test.go`, `internal/cron/run_store.go`, `internal/cron/run_store_test.go`
- Test commands: `go test ./internal/cron -run 'TestOperatorRunReport' -count=1`, `go test ./internal/cron -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Builder reports stable OperatorRunReport JSON fixtures for success and degraded unattended runs, redaction evidence, recommended repair commands, and no live provider/gateway/scheduler dependency.
- Acceptance: A pure report builder maps successful cron run evidence into a stable JSON artifact with job/run identity, provider/model, delivery target, timestamps, status, and session/transcript refs., A degraded fixture maps provider/auth missing, timeout, suppressed, and delivery-failed evidence into stable degraded_reason codes plus recommended_next_command values., Report JSON never includes raw API keys, full secret refs, provider response bodies, or unredacted filesystem home paths., The builder can write/read the artifact under a temp GORMES_HOME-style path without starting cron, gateway, kernel, or provider clients.
- Source refs: internal/cron/run_store.go, internal/cron/run_completion.go, internal/cron/executor.go, internal/cron/delivery_plan.go, internal/subagent/durable_ledger.go, internal/runtime/binding.go, cmd/gormes/status.go
- Unblocks: Scheduled briefing job emits operator run report, Morning degraded-status summary over latest run report, Gateway delivery evidence in operator run report, Provider/auth readiness preflight for unattended jobs
- Why now: Unblocks Scheduled briefing job emits operator run report, Morning degraded-status summary over latest run report, Gateway delivery evidence in operator run report, Provider/auth readiness preflight for unattended jobs.

## 2. Hermes tool tail strict-fidelity source-pair expansion

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

## 3. Strict-fidelity upstream test-suite classifier

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

## 4. Hermes gateway platform strict-fidelity source-pair expansion

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

## 8. Hermes x_search tool and auth surface

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Expose Hermes' first-class `x_search` tool in Gormes with a descriptor, OAuth/API-key auth status, query/result envelope, rate-limit/degraded errors, and registry/toolset visibility without requiring live X credentials in tests.
- Trust class: -
- Ready when: Tool registry and auth status helpers can be exercised with fake HTTP and temp config.
- Not ready when: The slice requires a live X OAuth/API-key credential., The slice hides x_search from tool descriptors while adding only a CLI helper.
- Degraded mode: -
- Fixture: `internal/tools x_search fake transport fixtures`
- Write scope: `internal/tools`, `internal/config`, `cmd/gormes/registry.go`
- Test commands: `go test ./internal/tools -run 'TestXSearch\|TestToolRegistry' -count=1`, `go test ./internal/config -run 'TestXSearch\|TestAuth' -count=1`, `go run ./cmd/progress validate`
- Done signal: x_search descriptor, auth status, fake-result normalization, and degraded errors are proven without live X credentials.
- Acceptance: `x_search` appears in the registry with Hermes-compatible schema and toolset availability., OAuth and API-key auth modes produce redacted status and missing-auth diagnostics., Fake search results normalize into a bounded model-visible result envelope; rate-limit and auth failures degrade explicitly.
- Source refs: ../hermes-agent/tools/x_search_tool.py, ../hermes-agent/tools/xai_http.py, ../hermes-agent/tests/tools/test_x_search_tool.py, ../hermes-agent/website/docs/user-guide/features/x-search.md, internal/tools, internal/config
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. Hermes session recap command surface

- Phase: 5 / 5.O
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Port Hermes' session recap command as a Gormes-native read-only session summarizer over local session/transcript storage, preserving output modes, missing-session diagnostics, and provider-free degraded behavior.
- Trust class: -
- Ready when: Gormes session list/export storage helpers are available for hermetic temp-store tests.
- Not ready when: The slice calls a live model to summarize instead of first proving the read-only recap envelope and degraded provider-free path., The slice mutates session history while generating a recap.
- Degraded mode: -
- Fixture: `cmd/gormes session recap fixtures`
- Write scope: `cmd/gormes`, `internal/session`, `internal/store`
- Test commands: `go test ./cmd/gormes -run 'TestSessionRecap\|TestSession' -count=1`, `go test ./internal/session ./internal/store -run Recap -count=1`, `go run ./cmd/progress validate`
- Done signal: Session recap fixtures prove read-only transcript loading, bounded output, missing-session diagnostics, and no live provider dependency.
- Acceptance: Known session transcripts render a deterministic recap envelope in human and JSON modes., Missing or empty sessions return explicit diagnostics without panics or live provider calls., Long transcripts are bounded with visible truncation evidence.
- Source refs: ../hermes-agent/hermes_cli/session_recap.py, ../hermes-agent/hermes_cli/main.py, internal/session, internal/store
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. Long-term plan: profile fleet supervisor and single control-plane gateway

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

<!-- PROGRESS:END -->
