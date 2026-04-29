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

## 2. Environment interface + file sync contract

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

## 3. Image input mode router + native content parts

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

## 4. ACP server side

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

## 5. Backup/update opt-in and exclusion policy

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

## 6. OCI image

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

## 7. Homebrew

- Phase: 5 / 5.P
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Contract: Gormes ports Hermes Homebrew/release artifact expectations into a Go-native formula fixture with version, checksum, binary install layout, and doctor smoke contract.
- Trust class: operator, system
- Ready when: Static binary artifact naming and version output are stable enough for formula fixtures., The first slice can validate formula text and fake artifact metadata without pushing a tap.
- Not ready when: The slice publishes release artifacts, mutates a live Homebrew tap, or requires network downloads in tests., The slice mixes Homebrew with OCI, Nix, service units, or installer shell scripts.
- Degraded mode: Formula validation reports missing artifact, checksum, binary layout, or doctor smoke evidence instead of publishing an untestable tap update.
- Fixture: `docs/install/homebrew_formula_test.go`
- Write scope: `packaging/homebrew/gormes-agent.rb`, `docs/install/homebrew_formula_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./docs -run TestHomebrewFormulaContract -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Homebrew formula fixtures prove version/checksum/artifact/install-layout behavior without live tap publication or network downloads.
- Acceptance: Formula fixtures prove class name, version, URL, checksum, binary install path, and doctor smoke command are present., Release-script fixtures prove Gormes artifact names and checksums can feed the formula without Hermes Python packaging paths., Nix/flake references remain separate row-backed packaging work unless explicitly included in a later Nix row.
- Source refs: ../hermes-agent/packaging/homebrew/hermes-agent.rb, ../hermes-agent/scripts/release.py, ../hermes-agent/flake.nix, docs/content/building-gormes/architecture_plan/hermes-honcho-go-runtime-plan.md:Release packaging divergence matrix
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Yuanbao protocol envelope + markdown fixtures

- Phase: 7 / 7.E
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P4`
- Contract: Gormes parses Yuanbao websocket/protobuf-style envelopes and Markdown message fragments into gateway-neutral events using fixture data only
- Trust class: gateway, system
- Ready when: The Phase 2 shared gateway event shape and Regional + Device Adapter Backlog are available; this row does not need a live Yuanbao account., Workers can start with captured JSON/proto/markdown testdata under internal/channels/yuanbao/testdata copied or minimized from upstream fixtures., No send loop, login flow, tool registration, media download, or sticker parsing is required for this first slice.
- Not ready when: The slice opens a websocket, performs login, calls Tencent/Yuanbao endpoints, downloads media, or registers user-visible tools., The slice stores credentials or changes shared gateway session policy., The slice combines protocol parsing with send/reply runtime behavior.
- Degraded mode: Yuanbao adapter status reports protocol_unavailable or markdown_parse_failed evidence instead of starting a live session with unparsed payloads.
- Fixture: `internal/channels/yuanbao/proto_test.go`
- Write scope: `internal/channels/yuanbao/proto.go`, `internal/channels/yuanbao/proto_test.go`, `internal/channels/yuanbao/markdown.go`, `internal/channels/yuanbao/markdown_test.go`, `internal/channels/yuanbao/testdata/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels/yuanbao -run 'TestYuanbao(Proto\|Markdown)' -count=1`, `go test ./internal/channels/yuanbao -count=1`, `go run ./cmd/progress validate`
- Done signal: Yuanbao protocol/markdown fixtures prove inbound text event normalization and degraded parse evidence with no live Yuanbao network call.
- Acceptance: TestYuanbaoProto_DecodesInboundTextFixture loads a captured fixture and returns source, conversation id, message id, author role, and text content., TestYuanbaoMarkdown_RendersCodeAndLinks proves code blocks, links, mentions, and list fragments are normalized into plain prompt-safe text without losing URLs., Malformed/unknown envelope fixtures return typed degraded evidence and do not panic., No test imports a generated protobuf runtime unless a local generated fixture file is checked in under internal/channels/yuanbao.
- Source refs: ../hermes-agent/gateway/platforms/yuanbao_proto.py@ab687963, ../hermes-agent/gateway/platforms/yuanbao.py@ab687963, ../hermes-agent/tests/test_yuanbao_proto.py@ab687963, ../hermes-agent/tests/test_yuanbao_markdown.py@ab687963, ../hermes-agent/website/docs/user-guide/messaging/yuanbao.md@ab687963
- Unblocks: Yuanbao media/sticker attachment normalization, Yuanbao gateway runtime + toolset registration
- Why now: Unblocks Yuanbao media/sticker attachment normalization, Yuanbao gateway runtime + toolset registration.

<!-- PROGRESS:END -->
