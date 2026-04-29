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
## 1. Channel-neutral native runtime turn adapter

- Phase: 2 / 2.F.4
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Telegram, Slack, Discord, WhatsApp, BlueBubbles, and future channels enter the same native Gormes turn adapter so provider/runtime fixes preserve Hermes channel parity instead of hard-coding Telegram behavior.
- Trust class: gateway, operator, system
- Ready when: The builder restates the Hermes parity contract and confirms no dependency on hermes-agent runtime services before editing., A fake channel fixture can drive a message through the same turn adapter used by Telegram without importing Telegram SDK types., Existing per-channel identity/self-filter and delivery routing tests stay unchanged.
- Not ready when: The row adds a Telegram-only provider/runtime bypass., The row changes channel-specific identity safety, require-mention, delivery, or thread rules outside adapter boundaries., The row sends raw provider errors to external channels.
- Degraded mode: A channel with unsupported media, identity ambiguity, or provider-unavailable state emits typed channel evidence and a safe user-facing response while preserving session/admission cleanup.
- Fixture: `internal/gateway/channel_neutral_turn_adapter_test.go`
- Write scope: `internal/gateway/`, `internal/channels/`, `cmd/gormes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway ./internal/channels/... -run 'ChannelNeutral\|TurnAdapter\|Admission\|AwaitUserReply' -count=1`, `go test ./internal/gateway ./internal/channels/... -count=1`, `go run ./cmd/progress validate`
- Done signal: Gateway tests prove Telegram and a fake channel share the same native runtime adapter while preserving existing channel identity and delivery contracts.
- Acceptance: A shared channel-neutral turn request contains channel, chat/thread, sender identity, session source, media references, command/admission metadata, and reply route fields., Telegram fixture and one fake non-Telegram channel fixture both exercise the same native runtime adapter path., Provider/runtime failure is rendered through shared external-channel safe error handling and clears active turn state.
- Source refs: references/go-agent-os/GORMES-REUSE-AUDIT.md#recommended-immediate-builder-row, references/go-agent-os/trpc-agent-go/agent/await_user_reply.go, references/go-agent-os/nanobot/pkg/mcp/session.go, internal/gateway/manager.go, internal/gateway/session_context.go, internal/channels/telegram/, internal/channels/discord/, internal/channels/slack/, internal/channels/whatsapp/, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#gateway-channels-cron-api-tui-and-cli
- Unblocks: Native runtime provider gateway binding, Await-user-reply channel route, Live multi-channel parity smoke tests
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Native runtime provider gateway binding

- Phase: 4 / 4.I
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Gormes gateway constructs a native Go runtime/provider binding from Hermes-compatible config when no explicit endpoint is configured, so live Telegram and other channel turns do not default to a dead localhost backend while explicit OpenAI-compatible endpoints remain supported.
- Trust class: gateway, operator, system
- Ready when: The builder restates the Hermes parity contract and confirms no dependency on hermes-agent runtime services before editing., A failing test proves provider/model config without an explicit endpoint no longer resolves to the implicit 127.0.0.1:8642 HTTP backend for gateway turns., A separate test proves an explicit endpoint still uses the OpenAI-compatible HTTP client path.
- Not ready when: The implementation starts, shells out to, imports, or requires hermes-agent runtime services., The implementation changes Hermes config precedence or removes explicit endpoint support., The implementation solves only Telegram while bypassing the shared gateway/channel runtime path.
- Degraded mode: If native provider credentials or model routing are unavailable, gateway returns a typed provider/config admission error and clears active-turn state instead of calling hermes-agent runtime or hanging on 127.0.0.1:8642.
- Fixture: `internal/runtime/native_provider_gateway_binding_test.go`
- Write scope: `internal/runtime/`, `internal/hermes/`, `internal/config/`, `cmd/gormes/`, `internal/gateway/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/config ./internal/hermes ./internal/gateway ./cmd/gormes -run 'NativeRuntime\|ResolveInference\|Gateway.*Provider\|ExplicitEndpoint' -count=1`, `go test ./cmd/gormes ./internal/gateway -count=1`, `go run ./cmd/progress validate`
- Done signal: A source-run gateway can report native provider readiness/failure without connection-refused attempts to 127.0.0.1:8642 unless that endpoint was explicitly configured.
- Acceptance: Gateway invocation config with Hermes model.provider/model.default and no endpoint chooses a native provider path or a typed provider-config error, never the implicit localhost backend., Explicit endpoint/base-url config preserves OpenAI-compatible HTTP POST behavior for /v1/chat/completions or /v1/responses as configured., Provider/config failures clear active_agents/admission state and are visible through gateway status.
- Source refs: references/go-agent-os/GORMES-REUSE-AUDIT.md#1-native-runtime-wiring, references/go-agent-os/nanobot/pkg/runtime/runtime.go, references/go-agent-os/nanobot/pkg/tools/service.go, cmd/gormes/main.go, cmd/gormes/gateway.go, internal/config/config.go, internal/hermes/http_client.go, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#providers-models-and-credentials, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#gateway-channels-cron-api-tui-and-cli
- Unblocks: Live @gormes_bot normal turn, Channel-neutral native runtime binding, Provider-tool-memory golden transcript suite
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

## 3. ACP server side

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

## 4. Backup/update opt-in and exclusion policy

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

## 5. OCI image

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

## 6. Yuanbao protocol envelope + markdown fixtures

- Phase: 5 / 5.A
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Native tool execution bounds large tool results by persisting full output as a session artifact and returning a short text pointer to the model/channel, preserving Hermes operator readability and channel safety.
- Trust class: gateway, operator, system
- Ready when: The builder restates the Hermes parity contract and confirms no dependency on hermes-agent runtime services before editing., A small pure package can be tested without live providers, channels, or filesystem outside temp dirs., Artifact path policy uses Gormes data/session dirs, not reference repo paths.
- Not ready when: The row changes individual tool handlers before a shared result-budget helper exists., The row sends full oversized output to Telegram or provider context., The row stores artifacts outside a sanitized Gormes session/run directory.
- Degraded mode: If artifact persistence fails, the result is still bounded and includes a safe warning without exposing raw oversized payloads to external channels.
- Fixture: `internal/tools/result_budget_test.go`
- Write scope: `internal/tools/`, `internal/hermes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools ./internal/hermes -run 'Tool.*Budget\|Truncat\|Artifact' -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Tool-result budget tests prove oversized outputs become safe artifact pointers with sanitized paths and no channel/provider flooding.
- Acceptance: Text output over budget is truncated and full output is written to a sanitized artifact path., JSON/non-text output is persisted as JSON and represented by a short pointer., Callers receive evidence for truncated, persisted, and persistence_failed cases.
- Source refs: references/go-agent-os/GORMES-REUSE-AUDIT.md#2-tool-output-truncation, references/go-agent-os/nanobot/pkg/agents/truncate.go, references/go-agent-os/axe/internal/artifact/tracker.go, internal/tools/, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#tools-sandboxes-and-security
- Unblocks: 61-tool registry port, Native runtime provider gateway binding, MCP stdio transport + tool/list discovery
- Why now: Unblocks 61-tool registry port, Native runtime provider gateway binding, MCP stdio transport + tool/list discovery.

## 5. Gormes-native MCP host runtime boundary

- Phase: 5 / 5.G
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Gormes exposes a native MCP/tool host boundary with explicit tool declarations, filtering, audit evidence, and channel/runtime-safe execution without adopting a non-Hermes config surface.
- Trust class: gateway, operator, system
- Ready when: The builder restates the Hermes parity contract and confirms no dependency on hermes-agent runtime services before editing., The interface design identifies caller-facing tool declaration/call/filter types before transport implementation., Hermes toolset/config semantics remain the source of truth for what tools are enabled.
- Not ready when: The row imports Nanobot config semantics or changes Hermes config.yaml precedence., The row vendors a full MCP framework instead of creating a tested Gormes boundary., The row bypasses channel/tool trust classes.
- Degraded mode: Unavailable MCP servers/tools produce structured unavailable/unauthorized evidence while core Hermes-parity tools and channel commands continue to work.
- Fixture: `internal/tools/mcp_host_boundary_test.go`
- Write scope: `internal/tools/`, `internal/plugins/`, `internal/gateway/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools ./internal/plugins -run 'ToolDeclaration\|ToolFilter\|MCPHost\|Audit' -count=1`, `go test ./internal/tools ./internal/plugins -count=1`, `go run ./cmd/progress validate`
- Done signal: A native Gormes tool/MCP boundary exists behind Hermes-compatible toolset/config semantics with filter and audit tests.
- Acceptance: A Gormes tool declaration interface can render provider JSON schema and MCP metadata from one source., Include/exclude filters can restrict tools by channel, trust class, and configured toolset., Audit evidence records server/tool name, arguments redaction status, result status, and unavailable errors.
- Source refs: references/go-agent-os/GORMES-REUSE-AUDIT.md#3-runtime-service-wiring-via-explicit-optionsmergecomplete, references/go-agent-os/nanobot/pkg/tools/service.go, references/go-agent-os/nanobot/pkg/runtime/runtime.go, references/go-agent-os/trpc-agent-go/tool/tool.go, references/go-agent-os/trpc-agent-go/tool/filter.go, internal/tools/, internal/plugins/, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#tools-sandboxes-and-security
- Unblocks: MCP stdio transport + tool/list discovery, Managed tool gateway bridge, Tool output budget persisted artifact pointer
- Why now: Unblocks MCP stdio transport + tool/list discovery, Managed tool gateway bridge, Tool output budget persisted artifact pointer.

## 6. Goncho serialized write queue + relation candidates

- Phase: 5 / 5.N
- Owner: `memory`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Goncho serializes memory/conclusion writes and records pending relation candidates for possible conflicts or supersession without blocking the originating memory write.
- Trust class: operator, system
- Ready when: The builder restates the Hermes parity contract and confirms no dependency on hermes-agent runtime services before editing., Existing Goncho storage/tests identify the write entrypoints to serialize., The relation vocabulary is mapped to Honcho-compatible public behavior or explicitly kept internal.
- Not ready when: The row exposes Engram-specific API names as public Gormes/Honcho names., The row fails the original memory write because candidate detection is unavailable., The row adds an LLM judge before pending relation storage is deterministic.
- Degraded mode: If candidate search or relation insertion fails, the memory write still succeeds with degraded evidence; queue-full returns a retryable typed error.
- Fixture: `internal/goncho/write_queue_relation_test.go`
- Write scope: `internal/goncho/`, `internal/memory/`, `internal/gonchotools/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/goncho ./internal/memory -run 'WriteQueue\|Relation\|Conflict\|Supersede' -count=1`, `go test ./internal/goncho ./internal/memory -count=1`, `go run ./cmd/progress validate`
- Done signal: Goncho tests prove serialized writes and nonblocking pending relation candidates without changing Honcho-compatible external names.
- Acceptance: Concurrent memory writes execute in deterministic queue order under test., Queued cancellation before start does not mutate storage; started writes complete deterministically., Saving a memory can create pending relation candidates with verbs such as related, conflicts_with, supersedes, compatible, scoped, or not_conflict for later judgment.
- Source refs: references/go-agent-os/GORMES-REUSE-AUDIT.md#5-deterministic-serialized-mcpmemory-write-queue, references/go-agent-os/engram/internal/mcp/write_queue.go, references/go-agent-os/engram/internal/store/relations.go, references/go-agent-os/engram/internal/store/store.go, internal/goncho/, internal/memory/, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#honcho-feature-map-for-goncho
- Unblocks: Goncho memory integration into normal agent turn, Goncho operator diagnostics contract
- Why now: Unblocks Goncho memory integration into normal agent turn, Goncho operator diagnostics contract.

<!-- PROGRESS:END -->
