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
## 1. Goncho durable recall trace IR + fused ranking pipeline

- Phase: 5 / 5.N
- Owner: `memory`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Introduce an internal Goncho recall pipeline where `RecallEngine.Run` is the only caller-facing entrypoint, package-local generate/score/select phases produce a durable `RecallTrace`, and Honcho-compatible search/context projections can only be built from that trace. `RecallCandidate` owns facts/content/provenance, `RecallScore` owns scoring components and selection evidence, and `RecallTrace` owns replay/debug/eval state including `TraceID`, `PipelineVersion`, `RecallScoringConfig`, `CreatedAt`, selected/rejected candidates, and code-first warnings for every degraded path.
- Trust class: operator, system
- Ready when: The builder restates the invariant: no projection without a RecallTrace., Existing Goncho Search/Context responses remain Honcho-compatible and do not gain new required fields., Trace fixtures can be deterministic without live embeddings, hosted Honcho, external vector databases, or provider calls.
- Not ready when: The slice rewrites Goncho persistence, adds Qdrant/Postgres/Mongo/Neo4j dependencies, or changes public honcho_* tool/API contracts., The slice exposes package-local generate/score/select phases as the caller-facing interface instead of `RecallEngine.Run`., The implementation permits semantic/graph/FTS failures, scope exclusions, or token-budget truncation without a stable RecallWarning code.
- Degraded mode: Semantic, graph, FTS, scope, stale-index, and token-budget fallbacks must be recorded in RecallTrace.Warnings instead of silently widening or dropping recall; external Honcho-compatible response shapes stay unchanged.
- Fixture: `internal/goncho/testdata/recall_trace/*.golden.json`
- Write scope: `internal/goncho/recall_ir.go`, `internal/goncho/recall_pipeline.go`, `internal/goncho/recall_projector.go`, `internal/goncho/recall_trace_test.go`, `internal/goncho/recall_pipeline_test.go`, `internal/goncho/testdata/recall_trace/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/goncho -run 'TestRecall' -count=1`, `go test ./internal/goncho ./internal/memory -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Builder reports byte-stable recall trace fixtures, stable warning codes, deterministic fused ranking/diversity tests, trace-only projection tests, and no public Honcho API break.
- Acceptance: Types define RecallQuery, RecallCandidate, RecallScore, ScoredRecallCandidate, RecallScoringConfig, RecallWarning, RecallTrace, and trace-only projectors with no exported constructor path that builds projected Honcho responses from raw candidates., TraceID is deterministic from query, scope, ordered candidate IDs, scoring config version, and pipeline version; identical inputs/config produce byte-stable JSON fixtures., RecallScoringConfig includes Version, Weights, RRFK, MMRLambda, DiversityKeys, and TokenBudget and is copied into every trace., `RecallEngine.Run` always returns a trace containing PipelineVersion, CreatedAt, ScoringConfig, selected candidates, rejected candidates, and warnings; package-local tests may exercise generate/score/select stages directly., Ranking tests cover weighted fusion, RRF-style tie behavior, MMR-style diversity penalties, deterministic tie-breaks, scope filtering, and token-budget truncation., Warning tests cover semantic_unavailable, graph_disabled, stale_embedding_index, fts_unavailable, scope_excluded_all_candidates, and token_budget_truncated as stable codes., Projection tests prove Honcho-compatible search/context output is produced only from RecallTrace and keeps existing external response fields stable.
- Source refs: https://github.com/Protocol-Lattice/go-agent/blob/6aa6e253c98afb343502e35c537d37ba4d9d17ec/src/memory/engine/engine.go, https://github.com/Protocol-Lattice/go-agent/blob/6aa6e253c98afb343502e35c537d37ba4d9d17ec/src/memory/session/spaces.go, internal/goncho/service.go, internal/goncho/importance_scorer.go, internal/memory/recall.go, internal/memory/semantic_sql.go, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#honcho-feature-map-for-goncho
- Unblocks: Goncho recall diagnostics CLI, Goncho replayable retrieval traces, Goncho retrieval benchmark corpus
- Why now: P0 handoff; needs contract proof before closeout.

## 2. ACP setup-browser bootstrap parity

- Phase: 5 / 5.H
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: `gormes acp --setup-browser` ports Hermes' ACP browser-tool bootstrap behavior with platform-specific command planning, dry-run/report output, and browser harness dependency checks while keeping actual installs explicit and operator-approved.
- Trust class: -
- Ready when: ACP server/client rows are complete and command planning can be tested without installing browser tools.
- Not ready when: The slice downloads browsers or runs package managers during tests., The slice changes ACP JSON-RPC session behavior instead of only adding bootstrap planning.
- Degraded mode: -
- Fixture: `cmd/gormes acp setup-browser dry-run fixtures`
- Write scope: `cmd/gormes/acp.go`, `internal/acp`, `internal/tools`
- Test commands: `go test ./cmd/gormes ./internal/acp -run 'ACP.*SetupBrowser\|ACP.*Bootstrap' -count=1`, `go run ./cmd/progress validate`
- Done signal: ACP setup-browser dry-run and approval fixtures prove platform planning without live downloads.
- Acceptance: Linux/macOS and Windows plans match Hermes script intent and surface missing prerequisites., Dry-run output is deterministic and secret-free., Non-dry-run execution requires explicit operator approval and reports each step outcome.
- Source refs: ../hermes-agent/acp_adapter/bootstrap/bootstrap_browser_tools.sh, ../hermes-agent/acp_adapter/bootstrap/bootstrap_browser_tools.ps1, ../hermes-agent/acp_adapter/entry.py, cmd/gormes/acp.go, internal/acp
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Hermes LSP write-time semantic diagnostics

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

## 4. Hermes x_search tool and auth surface

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

## 5. Hermes session recap command surface

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

## 6. Long-term plan: profile fleet supervisor and single control-plane gateway

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

## 7. Native TUI Terminal.app truecolor and ANSI sanitizer parity

- Phase: 5 / 5.Q
- Owner: `tui`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Port Hermes Ink TUI Terminal.app/truecolor and ANSI sanitizer behavior into the native Gormes TUI so renderer output keeps cursor/source-of-truth stability, strips malformed CSI safely, and preserves readable color behavior across modern terminals.
- Trust class: -
- Ready when: Native TUI text rendering and input fast-echo helpers can be tested without launching an interactive terminal.
- Not ready when: The slice requires a live Terminal.app session or snapshots from a developer machine., The slice changes TUI layout or slash dispatch outside text/color/input sanitizer behavior.
- Degraded mode: -
- Fixture: `internal/tui Terminal.app/ANSI sanitizer fixtures`
- Write scope: `internal/tui`, `internal/tuigateway`, `cmd/gormes`
- Test commands: `go test ./internal/tui ./internal/tuigateway ./cmd/gormes -run 'Truecolor\|ANSI\|Terminal\|TextInput\|Resume' -count=1`, `go run ./cmd/progress validate`
- Done signal: Native TUI fixtures prove truecolor environment handling, ANSI sanitizer safety, and fast-echo cursor stability without a live terminal.
- Acceptance: Malformed or dangling ANSI/CSI sequences are stripped or bounded exactly by fixture expectations., Truecolor forcing/degradation is deterministic from injected terminal environment facts., Fast-echo cursor source-of-truth does not drift after sanitized writes.
- Source refs: ../hermes-agent/ui-tui/src/lib/forceTruecolor.ts, ../hermes-agent/ui-tui/src/lib/text.ts, ../hermes-agent/ui-tui/src/components/textInput.tsx, ../hermes-agent/ui-tui/src/__tests__/forceTruecolor.test.ts, ../hermes-agent/ui-tui/src/__tests__/text.test.ts, ../hermes-agent/ui-tui/src/__tests__/textInputFastEcho.test.ts, internal/tui
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Hermes v0.14 optional skill catalog refresh

- Phase: 6 / 6.C
- Owner: `skills`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Refresh the Gormes skill catalog and metadata compatibility checks against Hermes v0.14 optional skills, including devops/pinggy-tunnel, research/darwinian-evolver, research/osint-investigation, and the updated Notion skill, without blindly copying unsupported Python scripts into runtime packages.
- Trust class: -
- Ready when: Skill metadata parser and hub registry fixtures exist.
- Not ready when: The slice vendors Hermes optional-skill scripts as trusted Go runtime code., The slice marks skills enabled by default without platform/dependency guards.
- Degraded mode: -
- Fixture: `internal/skills optional skill catalog fixtures`
- Write scope: `internal/skills`, `docs/development-skills`, `docs/content/building-gormes/architecture_plan`
- Test commands: `go test ./internal/skills -run 'Test.*Skill.*Catalog\|Test.*Optional' -count=1`, `go run ./cmd/progress validate`
- Done signal: Optional skill fixtures prove v0.14 metadata/catalog visibility and guarded unsupported-script handling.
- Acceptance: New optional skills parse with frontmatter, loaded/when metadata, references, and script/template inventories., Unsupported scripts remain catalog evidence with explicit dependency/degraded status., Skill hub/search output surfaces these skills with category and safety metadata.
- Source refs: ../hermes-agent/optional-skills/devops/pinggy-tunnel/SKILL.md, ../hermes-agent/optional-skills/research/darwinian-evolver/SKILL.md, ../hermes-agent/optional-skills/research/osint-investigation/SKILL.md, ../hermes-agent/skills/productivity/notion/SKILL.md, internal/skills
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. SimpleX Chat platform plugin parity

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

## 10. Hermes contract inventory gate

- Phase: 8 / 8.C
- Owner: `docs`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Build a report-only Hermes contract inventory gate that scans the current in-repo Hermes checkout, inventories source files, upstream docs pages, upstream tests, CLI/tool/provider/channel/session/memory/skill/learning-loop candidates, joins that evidence to progress.json rows and hermes-source-pairs.json, and emits machine-readable plus human-readable gap reports without failing CI by default. The gate must treat agent continuity as first-class: sessions, Memory/Goncho/Honcho compatibility, workspace/peer/profile identity boundaries, context retrieval and prompt budget, summaries/conclusions/search, skill templates and skills UX, skill precedence/sync/update/reset, learning-loop curator behavior, candidate memory/skill updates, feedback/outcome scoring, audit trail, mutation safety, prompt/context/memory/skill insertion ordering, and profile-scoped isolation. The gate proves whether a given Hermes SHA has every behavior/architecture contract classified as covered, partial, planned, excluded, or owned_divergence before Gormes claims full pairing.
- Trust class: operator, system
- Ready when: The in-repo Hermes checkout is present and its current SHA can be read without importing or running Hermes., hermes-source-pairs.json validates against the current Hermes SHA or reports stale evidence explicitly., The first implementation is accepted as report-only; CI-blocking strict mode is planned separately or kept behind an explicit flag., Matching is conservative and deterministic: source-pair rows, progress source_refs, upstream_tests, docs path refs, and taxonomy patterns produce confidence levels rather than proof., Agent-continuity categories are treated as first-class inventory dimensions, not incidental notes under tools or CLI., Honcho/Goncho evidence is included as compatibility evidence for memory, session, workspace, peer, message, context, search, summary, conclusion, API, and SDK-style surfaces.
- Not ready when: The slice tries to infer every contract from Python AST or py2many output instead of emitting a conservative inventory., The slice mutates progress.json rows automatically, creates hundreds of rows, or marks rows covered without planner review., The slice fails CI by default before the initial gap baseline is classified., The slice requires Hermes to be importable/runnable, live profile data, credentials, network access, or successful py2many transpilation., The report treats low-confidence matches as proof of coverage., The slice omits Memory/Goncho, skills templates/UX, learning-loop curator behavior, or prompt/context/memory/skill insertion ordering from the inventory categories., The report merges agent continuity into one vague row instead of separating memory, skills, learning, prompt insertion, and profile isolation gaps.
- Degraded mode: Without this inventory gate, Hermes pairing remains a hand-maintained release-note/source-file map and can silently drift from rolling upstream HEAD; Gormes must continue describing parity as source-backed but not exhaustive, especially for agent-continuity behavior that spans sessions, memory, skills, learning-loop updates, prompt insertion, and profile isolation.
- Fixture: `internal/repoctl/hermes_contract_inventory_test.go; webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md`
- Write scope: `cmd/repoctl`, `internal/repoctl`, `internal/progress`, `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json`, `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`, `webpages/docs/content/building-gormes/modules/docs.md`
- Test commands: `go test ./internal/repoctl -run 'TestHermesContractInventory' -count=1`, `go run ./cmd/repoctl hermes-contract-inventory --repo-root .`, `go run ./cmd/progress validate`, `go test ./webpages/docs -count=1`, `git diff --check`
- Done signal: Hermes contract inventory JSON and Markdown reports are generated for the current Hermes SHA, unclassified gaps are visible by severity, agent-continuity categories are first-class in the report, and normal validation remains report-only green unless strict mode is explicitly requested.
- Acceptance: `repoctl` can generate `webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json` and `.md` for the current Hermes SHA., The JSON records Hermes SHA, generated timestamp, source/docs/tests inventory counts, unmapped lists, extracted CLI/tool/provider/channel candidates, matched progress rows, matched source-pair rows, confidence, and gap severity., The Markdown records headline completion counts, critical blockers, per-module gap tables, release-checkpoint links, and a note that progress.json remains the only backlog., Default report mode surfaces gaps without failing progress validation or CI; a strict mode is explicit and can be promoted later., Docs state that Gormes may claim all Hermes features/architecture are paired only when every inventory gap is classified as covered, partial, planned, excluded, or owned_divergence for the current Hermes SHA., Upstream tests and docs pages with no mapped progress row or explicit exclusion appear as blockers, not as silently ignored files., The inventory has explicit sections or typed categories for sessions, Memory/Goncho/Honcho compatibility, workspace/peer/profile identity boundaries, context retrieval and prompt budget, summaries/conclusions/search, skills templates and skills UX, skill precedence/sync/update/reset, learning-loop curator behavior, candidate memory/skill updates, feedback/outcome scoring, audit trail, mutation safety, insertion ordering, and profile-scoped isolation., Agent-continuity gaps are reported separately from CLI/tool/channel gaps so planner passes can prioritize Memory/Goncho, skills UX, and learning-loop fidelity without creating a side backlog.
- Source refs: webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json, webpages/docs/content/building-gormes/architecture_plan/hermes-v0.14-module-pairings.md, webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md, webpages/docs/development-skills/gormes-planner/references/progress-row-contract.md, internal/repoctl/source_pairs.go:ValidateSourcePairs, internal/progress/progress.go:Item, cmd/repoctl/main.go, hermes-agent/RELEASE_v0.14.0.md, hermes-agent/hermes_cli/main.py, hermes-agent/tools/x_search_tool.py, hermes-agent/tests/hermes_cli/test_send_cmd.py, webpages/docs/content/building-gormes/architecture_plan/upstream-coverage-ledger.md:Hermes Source Coverage, webpages/docs/content/building-gormes/architecture_plan/upstream-coverage-ledger.md:Honcho Source Coverage, webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Prompt, Context, Compression, And Skills-In-Prompt, webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Plugins, Skills, Learning, And Specialized Modes, webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Honcho Feature Map For Goncho, hermes-agent/tools/memory_tool.py, hermes-agent/agent/memory_manager.py, hermes-agent/agent/skill_commands.py, hermes-agent/agent/skill_preprocessing.py, hermes-agent/agent/skill_utils.py, hermes-agent/tools/skills_tool.py, hermes-agent/tools/skill_manager_tool.py, hermes-agent/tools/skills_sync.py, hermes-agent/agent/curator.py, hermes-agent/hermes_cli/curator.py
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
