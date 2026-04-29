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

## 6. Homebrew

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

<!-- PROGRESS:END -->
