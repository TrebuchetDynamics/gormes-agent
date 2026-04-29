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
## 1. Self-monitoring telemetry

- Phase: 4 / 4.E
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Gormes bridges Hermes turn/provider/tool telemetry and Honcho telemetry/reasoning traces into local redacted telemetry, audit, and insights evidence through SelfMonitoringBridge, TelemetryEventMatrix, ReasoningTraceRecord, TelemetrySink, AuditSink, and InsightsRecorder interfaces without changing the local usage.jsonl schema until compatibility tests pass.
- Trust class: operator, system
- Ready when: Trajectory compression, Goncho webhook delivery, and Phase 3 insights rows are validated; provider usage remains the final local usage/status vocabulary dependency., Goncho memory and queue rows expose deterministic event inputs that can be fixture-tested without live providers, Prometheus, Sentry, Redis, or hosted Honcho., The slice can keep external exporters as owned/excluded divergence while preserving the event names, trace IDs, queue/dream/reconciliation evidence, and redaction semantics needed by local Gormes diagnostics.
- Not ready when: The slice changes the persisted usage.jsonl schema, starts Prometheus/Sentry/OpenTelemetry exporters, or sends telemetry over the network., The slice blocks kernel turns, provider calls, memory writes, or dream/reconciliation jobs when telemetry recording fails., The slice collapses Honcho reasoning traces into generic log lines without preserving event name, reasoning tree ID, level, parent/child relationship, and redaction evidence.
- Degraded mode: Telemetry emission failures, unavailable metrics exporters, and unsupported hosted Honcho tracing fields produce nonfatal local evidence instead of blocking turns, memory writes, or queue processing.
- Fixture: `internal/telemetry/self_monitoring_test.go; internal/goncho/telemetry_test.go`
- Write scope: `internal/telemetry/self_monitoring.go`, `internal/telemetry/self_monitoring_test.go`, `internal/audit/self_monitoring.go`, `internal/audit/self_monitoring_test.go`, `internal/insights/self_monitoring.go`, `internal/insights/self_monitoring_test.go`, `internal/goncho/telemetry.go`, `internal/goncho/telemetry_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/telemetry ./internal/audit ./internal/insights ./internal/goncho -run 'TestSelfMonitoring\|TestGonchoTelemetry\|TestReasoningTrace' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Telemetry fixtures prove Hermes usage metrics and Honcho reasoning/event traces map into deterministic redacted local evidence with nonfatal failure behavior and explicit hosted-exporter divergence.
- Acceptance: SelfMonitoringBridge accepts injected TelemetrySink, AuditSink, and InsightsRecorder implementations so telemetry, audit, and insights behavior can be tested without global state., A local event matrix maps Hermes turn/provider/tool usage and Honcho agent, dream, reconciliation, representation, and reasoning-trace events into Gormes telemetry/audit event names with owned/excluded exporter rationale., ReasoningTrace fixtures preserve trace ID, tree node ID, parent ID, level, event type, timing, and redacted payload summaries without raw prompts, secrets, or provider tokens., Provider/account usage and tool metrics bridge into the existing insights rollup as additive evidence without altering existing usage.jsonl field names., Telemetry failures return nonfatal degraded evidence and do not interrupt turns, memory writes, webhook delivery, queue processing, or dream scheduling., Tests classify hosted Prometheus/Sentry/exporter-only behavior as owned/excluded divergence and prove local event emission remains deterministic.
- Source refs: ../hermes-agent/agent/trajectory.py, ../hermes-agent/agent/usage_pricing.py:CanonicalUsage,normalize_usage,estimate_usage_cost, ../hermes-agent/tests/agent/test_usage_pricing.py, ../honcho/src/telemetry/emitter.py, ../honcho/src/telemetry/reasoning_traces.py, ../honcho/src/telemetry/events/agent.py, ../honcho/src/telemetry/events/deletion.py, ../honcho/src/telemetry/events/dream.py, ../honcho/src/telemetry/events/reconciliation.py, ../honcho/src/telemetry/events/representation.py, ../honcho/src/telemetry/metrics_collector.py, ../honcho/src/telemetry/sentry.py, ../honcho/src/telemetry/prometheus/metrics.py, ../honcho/tests/telemetry/test_events.py, ../honcho/tests/telemetry/test_emitter.py, ../honcho/tests/integration/test_telemetry.py, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Telemetry and reasoning traces, docs/content/building-gormes/architecture_plan/hermes-honcho-go-runtime-plan.md:Telemetry and reasoning traces
- Why now: P0 handoff; needs contract proof before closeout.

## 2. ContextEngine compression-boundary callback vocabulary

- Phase: 4 / 4.B
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: internal/hermes defines a compression-boundary callback vocabulary on ContextEngine with stable lineage evidence and status fields, without binding kernel compression execution yet
- Trust class: operator, system
- Ready when: ContextEngine interface + status tool contract is validated on main., Compression token-budget and single-prompt threshold fixtures are validated, so this slice only adds callback vocabulary and status evidence., The worker edits only internal/hermes context-engine files and the declared context_status JSON fixture; no kernel, transcript storage, summarizer, or Goncho/Honcho memory behavior is in scope.
- Not ready when: The slice edits internal/kernel/kernel.go, creates internal/kernel/context_engine.go, implements summarization, mutates transcript history, or binds live compression execution., The slice hides boundary failures or lets status imply a boundary callback ran before the kernel binding row is complete., The slice changes Goncho/Honcho memory extraction semantics; memory pre/post-compression observation remains a separate Phase 3/4 concern., The slice edits any testdata fixture except internal/hermes/testdata/context_status/disabled_pressure_unknown_tool.json.
- Degraded mode: Context status reports compression_boundary_unavailable or last_boundary_missing evidence until kernel compression execution binds the callback.
- Fixture: `internal/hermes/context_engine_boundary_test.go::TestContextEngineCompressionBoundaryVocabulary`
- Write scope: `internal/hermes/context_engine.go`, `internal/hermes/context_engine_test.go`, `internal/hermes/context_engine_boundary_test.go`, `internal/hermes/testdata/context_status/disabled_pressure_unknown_tool.json`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run 'Test.*ContextEngine.*Boundary\|TestDisabledContextEngine_StatusToolFixture\|TestContextStatusFixtures' -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/progress validate`
- Done signal: Hermes package fixtures and the disabled_pressure_unknown_tool.json status fixture prove boundary vocabulary, status evidence, unavailable/missing degraded modes, and no kernel or memory side effects.
- Acceptance: A new CompressionBoundary value carries old_session_id, new_session_id, reason, and compressed_at or equivalent stable lineage evidence., DisabledContextEngine or an in-package fake can record one boundary callback and expose last boundary evidence through Status without contacting a provider., Status reports compression_boundary_unavailable or last_boundary_missing when no boundary has been recorded., Existing context_status tool fixtures remain stable except for the added boundary evidence fields., The disabled_pressure_unknown_tool.json fixture is updated only as needed to include explicit missing-boundary or unavailable-boundary evidence.
- Source refs: ../hermes-agent/run_agent.py@e85b7525, ../hermes-agent/tests/run_agent/test_compression_boundary_hook.py@e85b7525, internal/hermes/context_engine.go:ContextEngine, internal/hermes/context_engine_test.go, internal/hermes/testdata/context_status/disabled_pressure_unknown_tool.json
- Unblocks: ContextEngine compression-boundary notification
- Why now: Unblocks ContextEngine compression-boundary notification.

## 3. Environment interface + file sync contract

- Phase: 5 / 5.B
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Contract: Gormes ports Hermes sandbox environment and file-sync contracts into a Go Environment interface with path mapping, upload/download, timeout, cleanup, and parser-family inventory fixtures before backend-specific Docker/SSH/Modal/Daytona/Singularity execution lands.
- Trust class: operator, child-agent, system
- Ready when: Tool execution descriptor and command-runner seams are validated enough to define an interface without starting real backends., The first slice can use fake filesystem and fake parser fixtures; no Docker daemon, SSH server, Modal account, browser, or provider credential is required.
- Not ready when: The slice implements a real Docker, SSH, Modal, Daytona, Singularity, or browser environment backend instead of the shared interface and file-sync contract., The slice executes model-generated commands or parses live LLM output instead of using hermetic parser fixtures., The slice treats broad `environments/**` coverage as complete without exact parser-family rows.
- Degraded mode: Unavailable or unsupported environment backends return environment_backend_unavailable or parser_family_row_backed evidence without shelling out, starting containers, or dropping file-sync intent.
- Fixture: `internal/tools/environment_contract_test.go; internal/hermes/tool_call_parser_manifest_test.go`
- Write scope: `internal/tools/environment_contract.go`, `internal/tools/environment_contract_test.go`, `internal/hermes/tool_call_parser_manifest.go`, `internal/hermes/tool_call_parser_manifest_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools ./internal/hermes -run 'TestEnvironmentContract\|TestToolCallParserManifest' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Environment contract fixtures prove fake backend path/file-sync/timeout/cleanup behavior and parser-family manifest coverage without real sandbox backends.
- Acceptance: Environment interface fixtures prove path mapping, upload/download intent, timeout propagation, cleanup ordering, and unsupported-backend evidence over fake backends., File-sync fixtures prove checksum/delete intent and host/container path normalization without touching real remote filesystems., Parser manifest fixtures classify hermes_parser.py, deepseek_v3_1_parser.py, and the remaining parser family as implemented, row-backed, or excluded before raw parser execution parity is claimed., No test starts Docker, SSH, Modal, Daytona, Singularity, browsers, or provider clients.
- Source refs: ../hermes-agent/tools/environments/base.py:BaseEnvironment, ../hermes-agent/tools/environments/file_sync.py, ../hermes-agent/environments/hermes_base_env.py, ../hermes-agent/environments/agentic_opd_env.py, ../hermes-agent/environments/web_research_env.py, ../hermes-agent/environments/tool_call_parsers/hermes_parser.py, ../hermes-agent/environments/tool_call_parsers/deepseek_v3_1_parser.py, docs/content/building-gormes/architecture_plan/hermes-honcho-go-runtime-plan.md:Sandbox/environments, docs/content/building-gormes/architecture_plan/subsystem-inventory.md:Per-model tool-call parsers
- Unblocks: Docker, Modal, Daytona, Singularity, Raw tool-call parser fixture matrix, Terminal snapshot source stdout suppression guard
- Why now: Unblocks Docker, Modal, Daytona, Singularity, Raw tool-call parser fixture matrix, Terminal snapshot source stdout suppression guard.

## 4. Image input mode router + native content parts

- Phase: 5 / 5.D
- Owner: `provider`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: internal/hermes exposes a pure image input routing helper that resolves agent.image_input_mode auto/native/text from model vision capability and auxiliary vision override, then builds native provider content parts with text plus data-url image_url entries without invoking a live provider
- Trust class: operator, system
- Ready when: The worker can add a pure helper under internal/hermes with injected model capability and auxiliary-vision config values; no run_agent, kernel, gateway, or config-file binding is required., Tests create temp fixture image bytes and inspect generated data URLs; no provider request, image resizing, OCR, or external binary is required., Auto mode must choose native only when the active model is known to support vision and no auxiliary vision provider/model/base_url override is configured.
- Not ready when: The slice changes provider HTTP request builders, kernel message history, TUI file-drop behavior, gateway media ingestion, or image generation tools., The slice implements text OCR/vision-tool fallback, image resizing, or shrink retry., The slice treats unknown model capability as native vision support in auto mode.
- Degraded mode: Multimodal status reports image_input_text_fallback, image_input_native_forced, image_input_native_unavailable, or image_input_auxiliary_vision_override instead of silently dropping images.
- Fixture: `internal/hermes/image_routing_test.go::TestImageInputRouting_*`
- Write scope: `internal/hermes/image_routing.go`, `internal/hermes/image_routing_test.go`, `internal/hermes/client.go`, `internal/hermes/model_registry.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run '^TestImageInputRouting_\|^TestBuildNativeImageContentParts_' -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/progress validate`
- Done signal: Image routing fixtures prove auto/native/text selection, auxiliary-vision fallback, model capability handling, native data-url content part construction, unreadable-image evidence, and default prompt behavior without live providers.
- Acceptance: TestImageInputRouting_AutoNativeWhenModelSupportsVision proves auto mode returns native for a vision-capable active model with no auxiliary vision override., TestImageInputRouting_AutoTextWhenAuxVisionConfigured proves auxiliary vision provider/model/base_url config forces text fallback even when the active model supports vision., TestImageInputRouting_AutoTextForUnknownOrNonVisionModel proves unknown and non-vision model capabilities choose text fallback., TestBuildNativeImageContentParts_TextAndImages emits one text part plus image_url data-url parts in input order, skipping unreadable paths with evidence., TestBuildNativeImageContentParts_DefaultPrompt inserts a short default prompt when user text is empty and at least one image is present.
- Source refs: ../hermes-agent/agent/image_routing.py@ec671c41, ../hermes-agent/tests/agent/test_image_routing.py@ec671c41, ../hermes-agent/run_agent.py@ec671c41:_model_supports_vision, ../hermes-agent/run_agent.py@ec671c41:vision-aware preprocessing, internal/hermes/client.go:MessageContentPart, internal/hermes/model_registry.go
- Unblocks: Image-too-large shrink retry helper
- Why now: Unblocks Image-too-large shrink retry helper.

## 5. ACP server side

- Phase: 5 / 5.H
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Contract: Gormes maps Hermes ACP adapter entry/auth/session/tools/permissions/events into a Go-native manifest and stdio/server protocol fixture before editor integrations are advertised.
- Trust class: operator, system
- Ready when: MCP schema normalization and managed gateway bridge rows are validated enough to reuse tool/permission descriptors., This slice can remain a manifest/protocol fixture package without starting an editor or spawning subprocesses.
- Not ready when: The slice starts a live ACP server, shells out to Hermes/Python, or registers editor integrations before auth/session/tool/permission event shapes are fixture-backed., The slice claims ACP parity from broad acp_adapter/** coverage without exact auth, entry, session, tool, permission, and event refs.
- Degraded mode: Unsupported ACP provider detection, missing auth, and permission prompt paths return explicit acp_row_backed evidence instead of silently registering an incomplete editor bridge.
- Fixture: `internal/acp/server_manifest_test.go`
- Write scope: `internal/acp/server_manifest.go`, `internal/acp/server_manifest_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`, `docs/upstream_coverage_test.go`
- Test commands: `go test ./internal/acp -run TestACPServerManifest -count=1`, `go test ./docs -run TestNestedUpstreamFeatureCoverage -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: ACP manifest fixtures prove exact upstream auth, entry, session, tool, permission, event, and registry surfaces are classified before live ACP server work starts.
- Acceptance: ACP manifest fixtures classify auth provider detection, stdio entry startup, session lifecycle, tool rendering, permission prompts, and event streaming as implemented, row-backed, owned, or excluded., The Go target names future internal/acp package boundaries and adapter event structs without importing Python., Tests fail when upstream acp_adapter files or acp_registry/agent.json change without manifest classification., Editor integration docs remain row-backed until protocol fixtures exist.
- Source refs: ../hermes-agent/acp_adapter/auth.py:detect_provider, ../hermes-agent/acp_adapter/entry.py:main, ../hermes-agent/acp_adapter/server.py, ../hermes-agent/acp_adapter/session.py, ../hermes-agent/acp_adapter/tools.py, ../hermes-agent/acp_adapter/permissions.py, ../hermes-agent/acp_adapter/events.py, ../hermes-agent/acp_registry/agent.json, ../hermes-agent/tests/acp/, docs/content/building-gormes/architecture_plan/hermes-honcho-go-runtime-plan.md:ACP server/session/tools/permissions matrix
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Hermes CLI command-tree parity manifest

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Contract: cmd/gormes owns a source-backed Hermes CLI compatibility manifest that enumerates every upstream top-level command, nested subcommand, global/root flag, slash command, gateway command handler, dynamic plugin-provided command, and Gormes-owned addition, then classifies each path as implemented, row-backed, owned, excluded, or not-yet-applicable before any handler work claims CLI parity.
- Trust class: operator, gateway, system
- Ready when: The Phase 5.O CLI umbrella remains inventory-only, so a manifest row can be added without reopening global planning., internal/cli CommandRegistry already locks Hermes slash-command names and active-turn policy., cmd/gormes uses Cobra and exposes root, gateway, session, memory, goncho, doctor, telegram, and version command surfaces that can be classified without executing live services.
- Not ready when: The slice implements missing command handlers, provider auth, setup wizard prompts, migration writes, backup archives, service-manager calls, or dashboard routes., The slice generates classifications solely from upstream at runtime instead of maintaining a reviewed manifest plus a drift test against upstream parser/registry surfaces., Any upstream command, subcommand, flag, alias, or slash command is left unclassified.
- Degraded mode: Unknown or unclassified Hermes command paths, including dynamic plugin command registrations, fail the parity test; unsupported-but-known commands must carry row-backed or owned evidence instead of disappearing from help, docs, or migration plans.
- Fixture: `cmd/gormes/hermes_cli_parity_test.go::TestHermesCLIParityManifest`
- Write scope: `cmd/gormes/hermes_cli_parity.go`, `cmd/gormes/hermes_cli_parity_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes -run 'TestHermesCLIParityManifest\|TestHermesCLIParityManifestNoUnknowns' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Hermes CLI parity manifest fixtures prove every top-level command, nested command, static slash command, dynamic plugin command, alias, gateway handler, and Gormes-owned divergence has a non-unknown classification with source refs and residual rows.
- Acceptance: A manifest entry exists for every top-level Hermes parser command: chat, model, fallback, gateway, setup, whatsapp, slack, login, logout, auth, status, cron, webhook, hooks, doctor, dump, debug, backup, import, config, pairing, skills, plugins, memory, tools, mcp, sessions, insights, claw, version, update, uninstall, acp, profile, completion, dashboard, and logs., Nested parser groups are covered for gateway, fallback, auth, cron, webhook, hooks, debug, config, pairing, skills including tap and snapshot, plugins, memory, tools, mcp, sessions, claw, and profile., Every slash command and alias in hermes_cli/commands.py is cross-linked to internal/cli.CommandRegistry status or a named row-backed residual., Dynamic plugin CLI and slash-command surfaces discovered through active memory plugins and PluginContext.register_command entries are classified as implemented, row-backed, owned, excluded, or not-yet-applicable., Plugin manager and plugin command surfaces from hermes_cli/plugins.py and hermes_cli/plugins_cmd.py are classified as manifest-only, implemented, row-backed, owned, or excluded before any plugin runtime execution is claimed., Gormes-owned surfaces such as goncho, --offline, --remote, and XDG/TOML config paths are classified as owned with rationale, not counted as Hermes gaps; Hermes-owned `-z/--oneshot` is classified as implemented parity, not owned divergence., Typo-like requested paths such as `gormes migrate ooenclaw` are classified explicitly: they must return a deterministic suggestion for `gormes migrate openclaw` and must not become silent import aliases without a dedicated compatibility row., Destructive or secret-bearing command paths carry explicit evidence flags for confirmation, dry-run, redaction, or credential routing.
- Source refs: ../hermes-agent/hermes_cli/main.py:subparsers.add_parser, ../hermes-agent/hermes_cli/main.py:discover_plugin_cli_commands, ../hermes-agent/hermes_cli/commands.py:COMMAND_REGISTRY,GATEWAY_KNOWN_COMMANDS, ../hermes-agent/hermes_cli/plugins.py:PluginManager, ../hermes-agent/hermes_cli/plugins_cmd.py:plugins_command, ../hermes-agent/gateway/run.py:_handle_status_command,_handle_restart_command,_handle_reset_command,_handle_help_command,_handle_model_command,_handle_profile_command,_handle_update_command,_handle_approve_command,_handle_deny_command,_handle_voice_command,_handle_usage_command, ../hermes-agent/plugins/memory/__init__.py:discover_plugin_cli_commands, ../hermes-agent/plugins/disk-cleanup/__init__.py:PluginContext.register_command, ../hermes-agent/hermes_cli/config.py:config_command,set_config_value,check_config_version,migrate_config, ../hermes-agent/hermes_cli/claw.py:claw_command,_cmd_migrate,_cmd_cleanup, cmd/gormes/main.go:newRootCommandWithRuntime, internal/cli/command_registry.go:CommandRegistry, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:CLI command tree
- Unblocks: Gormes config command surface, Gormes config edit/check/native schema-migrate closeout, Hermes config migration dry-run manifest, OpenClaw migration dry-run manifest, Gateway, platform, webhook, and cron management CLI, Diagnostics, backup, logs, and status CLI
- Why now: Unblocks Gormes config command surface, Gormes config edit/check/native schema-migrate closeout, Hermes config migration dry-run manifest, OpenClaw migration dry-run manifest, Gateway, platform, webhook, and cron management CLI, Diagnostics, backup, logs, and status CLI.

## 7. Provider endpoint/API-key root flags + runtime resolution

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Contract: cmd/gormes accepts --endpoint, --api-key, --model, and --provider as invocation-only overrides for oneshot and TUI startup; flag values win over env/config for the current process, --api-key is never persisted, and all status/error evidence redacts the secret value.
- Trust class: operator, system
- Ready when: Cobra already owns the root command and model/provider flags., internal/config already has a typed HermesCfg and precedence vocabulary for flag, env, and config sources., This slice only handles invocation-time overrides; persistent config writes remain in the Gormes config command row.
- Not ready when: The slice writes config.toml or .env., The slice introduces Viper/global config state instead of using the existing internal/config loader contract., The slice logs, formats, or stores the raw --api-key value.
- Degraded mode: If endpoint/model/provider flags are incomplete or ambiguous, startup returns an exit-code-2 operator error before opening a provider client; no config file is modified.
- Fixture: `cmd/gormes/provider_flag_resolution_test.go`
- Write scope: `cmd/gormes/main.go`, `cmd/gormes/provider_flag_resolution_test.go`, `internal/config/config.go`, `internal/config/config_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes ./internal/config -run 'Test.*Provider.*Flag\|Test.*Endpoint\|Test.*APIKey\|Test.*OneshotInference\|Test.*TUI.*Model' -count=1`, `go run ./cmd/progress validate`
- Done signal: Provider-flag fixtures prove endpoint/model/provider/api-key precedence, redaction, no config mutation, and unchanged ambiguity errors for oneshot and TUI startup.
- Acceptance: Root help exposes --endpoint and --api-key alongside the existing --model and --provider flags., A oneshot fixture proves --endpoint, --api-key, and --model build the provider client with flag values even when config/env contain stale values., A TUI startup fixture proves the same resolution path is used without mutating persisted config defaults., A redaction fixture proves raw API key bytes never appear in returned errors, status evidence, or test logs., Provider-without-model ambiguity keeps the existing explicit-model guard.
- Source refs: cmd/gormes/main.go:newRootCommandWithRuntime,resolveOneshotInvocation,resolveTUIInvocation, internal/config/config.go:Load,ResolveInference,HermesCfg, README.md:Model-Backed Turn, ../hermes-agent/hermes_cli/config.py:set_config_value,config_command, ../hermes-agent/tests/hermes_cli/test_set_config_value.py
- Unblocks: Gormes config command surface, Hermes config migration writer, OpenClaw migration writer and cleanup command
- Why now: Unblocks Gormes config command surface, Hermes config migration writer, OpenClaw migration writer and cleanup command.

## 8. Backup/update opt-in and exclusion policy

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: CLI backup/update policy defaults pre-update backups off unless explicitly requested, honors --no-backup over --backup, and excludes checkpoints plus SQLite WAL/SHM/journal sidecars from backup manifests
- Trust class: operator, system
- Ready when: Diagnostics, backup, logs, and status CLI remains an umbrella; this row is the first pure backup policy helper and does not require a real update command., Tests use synthetic flag values and temp path lists; no archive writer, git pull, network, package manager, or real Gormes home is required.
- Not ready when: The slice implements update execution, writes archives, contacts git remotes, changes installer scripts, or scans the real operator home directory., The slice includes checkpoints/, *.db-wal, *.db-shm, or *.db-journal files in a default backup manifest., The slice changes log redaction or support-upload behavior.
- Degraded mode: Update status reports backup_skipped_default, backup_forced, backup_disabled_by_flag, or backup_manifest_excluded_paths instead of silently archiving large or unsafe runtime files.
- Fixture: `internal/cli/backup_policy_test.go::TestBackupPolicy_*`
- Write scope: `internal/cli/backup_policy.go`, `internal/cli/backup_policy_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cli -run '^TestBackupPolicy_\|^TestBackupManifestExclusions_' -count=1`, `go test ./internal/cli -count=1`, `go run ./cmd/progress validate`
- Done signal: Backup policy fixtures prove pre-update backups are opt-in, --no-backup wins, and checkpoints plus SQLite WAL/SHM/journal sidecars are excluded from manifests without archive or network side effects.
- Acceptance: TestBackupPolicy_DefaultSkipsPreUpdateBackup proves no backup is requested when neither --backup nor --no-backup is set., TestBackupPolicy_ExplicitBackupEnables proves --backup requests a backup and emits backup_forced evidence., TestBackupPolicy_NoBackupWins proves --no-backup suppresses backup even when --backup is also true., TestBackupManifestExclusions_SkipsCheckpointsAndSQLiteSidecars proves checkpoints/, *.db-wal, *.db-shm, and *.db-journal are excluded while ordinary .db files remain eligible., Tests use synthetic paths/temp dirs only and do not create archives or invoke git.
- Source refs: ../hermes-agent/hermes_cli/main.py@ea3c5a14:update backup flags, ../hermes-agent/hermes_cli/backup.py@a9033c92:exclude checkpoints, ../hermes-agent/hermes_cli/backup.py@817633bc:exclude SQLite sidecars, ../hermes-agent/tests/hermes_cli/test_backup.py@817633bc, internal/cli/log_snapshot.go, cmd/gormes/doctor.go
- Unblocks: Backup manifest dry-run contract
- Why now: Unblocks Backup manifest dry-run contract.

## 9. Custom provider model-switch key_env write guard

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: internal/cli exposes a pure model-switch patch helper that accepts an in-memory custom provider ref plus a target model and returns the config patch/evidence for default_model changes while preserving original credential storage: providers that relied on key_env and had no inline api_key/api_key_ref must not gain an api_key entry, while providers that already had inline plaintext or `${VAR}` api_key may keep that existing value without writing resolved plaintext
- Trust class: operator, system
- Ready when: Custom provider model-switch credential preservation is validated on main and provides the resolver vocabulary for env-template, plaintext, key_env, unset, and missing credentials., This slice only adds a pure patch/model-switch helper under internal/cli/custom_provider_model_switch.go; no config reader, /model command handler, TUI picker, fake /v1/models server, provider routing, or cmd/gormes wiring is required., Table tests should construct input provider maps/structs in memory and assert the planned write shape; no process environment, filesystem, or network access is needed.
- Not ready when: The slice changes internal/config, internal/hermes, provider catalog probing, TUI model picker behavior, command wiring, or the existing custom_provider_secret resolver semantics., The helper writes an api_key field for a provider whose original config relied only on key_env., The helper writes resolved plaintext when the original provider used `${VAR}` or key_env references.
- Degraded mode: Model-switch planning returns credential_write_skipped_key_env, credential_ref_preserved, plaintext_preserved, or credential_missing evidence so setup/status surfaces can explain why api_key was not written. The credential-preservation prerequisite and backend health bypass are now validated, so this row should run as a pure internal/cli patch-helper fixture.
- Fixture: `internal/cli/custom_provider_model_switch_test.go::TestCustomProviderModelSwitchPatch_*`
- Write scope: `internal/cli/custom_provider_model_switch.go`, `internal/cli/custom_provider_model_switch_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cli -run 'TestCustomProviderModelSwitchPatch_\|TestResolveCustomProviderSecret_' -count=1`, `go test ./internal/cli -count=1`, `go run ./cmd/progress validate`
- Done signal: internal/cli custom-provider model-switch fixtures prove key_env-backed providers update default_model without adding api_key, existing inline references/plaintext are preserved without resolution, and resolver tests still pass.
- Acceptance: TestCustomProviderModelSwitchPatch_KeyEnvDoesNotSynthesizeAPIKey starts with {default_model:'old', key_env:'ACME_KEY'} and proves the patch sets default_model='new', preserves key_env, omits api_key, and returns credential_write_skipped_key_env evidence., TestCustomProviderModelSwitchPatch_InlineEnvRefPreserved starts with {api_key:'${ACME_KEY}'} and proves the patch keeps api_key='${ACME_KEY}' without resolving or overwriting it., TestCustomProviderModelSwitchPatch_PlaintextPreserved starts with {api_key:'sk-plain'} and proves plaintext is preserved only because it was already present., TestCustomProviderModelSwitchPatch_MissingCredentialStillUpdatesModelWithEvidence proves model changes remain possible while credential_missing evidence is returned for setup/status guidance., Existing TestResolveCustomProviderSecret_* fixtures remain green; this row does not redefine resolver semantics.
- Source refs: ../hermes-agent/hermes_cli/main.py@8258f4dc:_model_flow_named_custom, ../hermes-agent/tests/hermes_cli/test_custom_provider_model_switch.py@8258f4dc, ../hermes-agent/hermes_cli/main.py@8bbeaea6:_named_custom_provider_map, internal/cli/custom_provider_secret.go:CustomProviderRef,ResolveCustomProviderSecret, internal/cli/custom_provider_secret_test.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. OCI image

- Phase: 5 / 5.P
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Contract: Gormes ships an OCI image contract that mirrors upstream Docker entrypoint/config volume operational behavior while proving the final image contains the Go binary and no required Python runtime path.
- Trust class: operator, system
- Ready when: The Go binary build and offline doctor command are stable., The slice can test Dockerfile/entrypoint text and optional local container smoke fixtures without publishing an image.
- Not ready when: The slice requires live registry access, provider credentials, hosted Honcho Postgres/Redis, or Python package installation to prove Gormes runtime behavior., The slice changes installer policy or release artifact signing in the same pass.
- Degraded mode: Container smoke tests run offline and report missing binary, missing config volume, or Python-runtime dependency evidence without contacting registries or providers.
- Fixture: `docs/install/oci_image_test.go`
- Write scope: `Dockerfile`, `docker/entrypoint.sh`, `docs/install/oci_image_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./docs -run TestOCIImageContract -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: OCI image fixtures prove Go-binary runtime layout, offline doctor command behavior, config volume handling, and explicit hosted Honcho deploy divergence.
- Acceptance: Dockerfile fixtures prove the image builds or describes a Go-binary runtime path with no Hermes Python runtime dependency., Entrypoint fixtures preserve offline doctor/config-volume behavior and deterministic command forwarding., Honcho hosted compose/Prometheus/Grafana files are classified as owned/excluded divergence or docs-only operational examples, not required local Goncho runtime dependencies., A smoke command can run `gormes doctor --offline` with fake config volume inputs.
- Source refs: ../hermes-agent/Dockerfile, ../hermes-agent/docker/entrypoint.sh, ../hermes-agent/docker-compose.yml, ../honcho/Dockerfile, ../honcho/docker-compose.yml.example, ../honcho/docker/entrypoint.sh, ../honcho/docker/prometheus.yml, ../honcho/docker/grafana-datasource.yml, docs/content/building-gormes/architecture_plan/hermes-honcho-go-runtime-plan.md:Packaging/release/install
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
