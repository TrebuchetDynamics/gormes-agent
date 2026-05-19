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

## 3. Scheduled briefing job emits operator run report

- Phase: 2 / 2.D
- Owner: `orchestrator`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Wire scheduled briefing cron/fleet jobs so every unattended run writes an OperatorRunReport after completion. The slice should cover local delivery, suppressed/no-agent/script-only jobs, provider-backed runs, timeout/error paths, and repeat/terminal completion evidence while preserving existing cron_runs and CRON.md mirror behavior.
- Trust class: operator, system
- Ready when: Durable operator run report for unattended jobs is complete., Existing cron executor tests can inject fake kernel, run store, sink, and clock dependencies., Briefing jobs can be represented as normal cron Job fixtures without adding email/CRM integrations.
- Not ready when: The slice changes scheduler parsing, job storage semantics, or CRON.md format before the report artifact exists., It requires live provider calls, live gateway adapters, or wall-clock sleeps., It only writes reports for successful runs and drops timeout/suppressed/error paths.
- Degraded mode: If the briefing cannot execute or cannot produce user content, the cron executor still writes a report with status=degraded or failed, run completion evidence, redacted error summary, and next repair command.
- Fixture: `internal/cron/operator_briefing_report_test.go::TestScheduledBriefingWritesOperatorRunReport`
- Write scope: `internal/cron/executor.go`, `internal/cron/operator_run_report.go`, `internal/cron/operator_briefing_report_test.go`, `internal/cron/run_store.go`
- Test commands: `go test ./internal/cron -run 'TestScheduledBriefingWritesOperatorRunReport\|TestOperatorRunReport' -count=1`, `go test ./internal/cron -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Builder reports cron executor success/degraded fixtures writing OperatorRunReport artifacts for scheduled briefings while preserving cron_runs and CRON.md evidence.
- Acceptance: Executor fixtures prove successful scheduled briefing runs write an OperatorRunReport linked to cron run identity and output preview., Timeout, kernel submit error, suppressed, and no-agent/script-only fixtures each write a report with stable degraded status/evidence., Existing cron_runs audit and CRON.md mirror tests still pass without schema-breaking changes., No live provider, scheduler goroutine, gateway adapter, or network dependency is required.
- Source refs: internal/cron/executor.go, internal/cron/run_completion.go, internal/cron/run_store.go, internal/cron/mirror.go, internal/cron/job.go, internal/subagent/durable_ledger.go
- Unblocks: Morning degraded-status summary over latest run report
- Why now: Unblocks Morning degraded-status summary over latest run report.

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

## 5. Gateway delivery evidence in operator run report

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

## 6. Hermes agent runtime strict-fidelity source-pair expansion

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

## 7. Hermes plugin catalog strict-fidelity classifier

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

## 8. Hermes LSP write-time semantic diagnostics

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

## 9. Hermes x_search tool and auth surface

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

## 10. Morning degraded-status summary over latest run report

- Phase: 5 / 5.N
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Add a read-only operator summary surface that renders the latest OperatorRunReport into text and JSON for morning review. The summary must show whether the unattended run succeeded, degraded, or failed; include job/run identity, delivery status, provider/auth readiness, redacted error details, and a recommended next command; and integrate with existing gormes status-style output without mutating cron or gateway state.
- Trust class: operator, system
- Ready when: Durable operator run report for unattended jobs is complete., The status command already has text/JSON degraded-output conventions., The first version reads only local report files and does not query live gateways/providers.
- Not ready when: The implementation starts cron, gateway, or provider clients to compute status., The surface hides failed runs because there is no successful briefing content., Raw error text leaks API keys, token-bearing URLs, or unredacted home paths.
- Degraded mode: If no report exists or the latest report cannot be decoded, the command returns status=operator_report_unavailable with the path/reason redacted and points operators to the scheduler/doctor command rather than failing with raw filesystem errors.
- Fixture: `cmd/gormes/status_operator_report_test.go::TestStatusRendersLatestOperatorRunReport`
- Write scope: `cmd/gormes/status.go`, `cmd/gormes/status_operator_report_test.go`, `internal/cli/status.go`, `internal/cron/operator_run_report.go`
- Test commands: `go test ./cmd/gormes -run 'TestStatusRendersLatestOperatorRunReport\|TestStatusJSON' -count=1`, `go test ./cmd/gormes ./internal/cron -run 'Test(Status\|OperatorRunReport)' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Builder reports status text/JSON fixtures for latest operator run report, unavailable-report degradation, redaction, and unchanged existing status output.
- Acceptance: gormes status text output includes latest unattended run status, job/run id, delivery state, degraded reason, and recommended next command when a report exists., gormes status --json includes a stable operator_run_report object with empty/absent fields normalized for automation., Missing, unreadable, or malformed reports render operator_report_unavailable evidence without non-zero exit for read-only status., Existing status progress/system output remains present.
- Source refs: cmd/gormes/status.go, internal/cli/status.go, internal/cron/operator_run_report.go, internal/doctor/durable_ledger.go, internal/gateway/status.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
