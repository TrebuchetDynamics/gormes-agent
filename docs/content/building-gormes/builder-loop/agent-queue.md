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
## 1. GONCHO local-first markdown MCP memory requirement

- Phase: 3 / 3.F
- Owner: `memory`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: GONCHO must support a local-first memory mode that answers the OpenClaw community pain point: no cloud dependency, no mandatory API key, user-readable/editable markdown memory files, MCP-compatible access from any agent framework, optional local embeddings via Ollama, and restart-persistent storage.
- Trust class: operator, system
- Ready when: Existing Goncho service/tool contracts are audited for where markdown export/import, local embedding configuration, and MCP tool catalog behavior should attach., The implementation plan preserves Honcho-compatible external tool names while branding the internal subsystem as GONCHO.
- Not ready when: The design requires mem0, Zep, a hosted vector database, a cloud API key, or an opaque-only binary store to start., Markdown files are treated only as one-way exports instead of user-editable source material that can be reloaded safely.
- Degraded mode: If Ollama or embeddings are unavailable, GONCHO still persists and serves markdown-backed lexical/SQLite recall locally without requiring network access or a hosted memory provider.
- Fixture: `internal/goncho/local_markdown_mcp_test.go; internal/gonchotools/mcp_catalog_test.go; internal/memory/markdown_store_test.go`
- Write scope: `internal/goncho/`, `internal/gonchotools/`, `internal/memory/`, `cmd/gormes/`, `docs/content/building-gormes/architecture_plan/`, `docs/superpowers/specs/`
- Test commands: `go test ./internal/goncho ./internal/gonchotools ./internal/memory -count=1`, `go test ./cmd/gormes -run Goncho -count=1`, `go run ./cmd/progress validate`
- Done signal: Docs and tests demonstrate local markdown-backed GONCHO memory over MCP with no cloud/API-key dependency and restart persistence.
- Acceptance: A fresh local Gormes install can enable GONCHO memory without cloud credentials or a paid API key., Memories can be stored, inspected, edited, and reloaded as plain markdown files with deterministic conflict handling against SQLite/source-of-truth state., GONCHO exposes MCP-compatible memory tools/catalog entries usable by non-Gormes agent frameworks while keeping Honcho-compatible tool contracts where parity requires them., Optional Ollama embeddings are configurable and never required for basic persistence/recall., Memory survives process and machine restarts with documented on-disk paths and read-only diagnostics.
- Source refs: User-provided Reddit r/openclaw post, 2026-05-03: frustration with mem0/Zep/cloud-hosted/heavy/API-key memory; requested fully local markdown MCP memory for GONCHO., internal/goncho/, internal/gonchotools/, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md, docs/content/building-gormes/architecture_plan/progress.json, Phase 5.G MCP Integration / Goncho MCP tool catalog
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Secrets Runtime Controls

- Phase: 5 / 5.J
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Port OpenClaw's secrets runtime control surface: secrets apply for deploying previously generated plans, secrets audit to detect plaintext secrets/unresolved refs/precedence drift, secrets configure for interactive provider setup with SecretRef mapping and preflight validation, and secrets reload to re-resolve secret references and atomically swap the runtime snapshot.
- Trust class: operator
- Ready when: Hermes credential/oauth store migration (5.O) is validated., SecretRef resolution layer is defined.
- Not ready when: The row ships without audit/reload atomicity., The row stores secrets in plaintext outside the secrets provider.
- Degraded mode: Unresolved SecretRef, missing provider, or reload failure reports secrets_unavailable with exact ref path rather than silently falling back or leaking plaintext secrets.
- Fixture: `internal/tools/secrets_test.go`
- Write scope: `internal/tools/secrets.go`, `internal/tools/secrets_test.go`, `cmd/gormes/secrets.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestSecrets -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes secrets ships with audit, configure, and reload commands.
- Acceptance: gormes secrets audit detects plaintext secrets, unresolved refs, and precedence drift., gormes secrets reload atomically swaps runtime snapshot without restart., SecretRef format matches OpenClaw's typed {source, provider, id} object convention.
- Source refs: openclaw secrets apply/audit/configure/reload CLI surface, docs/content/building-gormes/openclaw-platform-parity-audit.md, docs/content/building-gormes/fleet-operational-patterns.md
- Unblocks: Security audit (5.J), Provider auth parity
- Why now: P0 handoff; needs contract proof before closeout.

## 3. Security Audit Command

- Phase: 5 / 5.J
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Port OpenClaw's security audit: gormes security audit --deep --fix --json. Deep mode includes live gateway probe checks. Fix mode applies safe remediations and file-permission fixes. JSON mode produces machine-readable output. Audit categories: gateway auth status, state integrity, channel security warnings, shell blocklist coverage, filesystem scoping, credential redaction.
- Trust class: operator
- Ready when: Shell blocklist + filesystem scoping (5.J) is operational., Secrets runtime controls (5.J) is operational.
- Not ready when: The row performs destructive fixes without --fix flag., The row requires live gateway for basic audit checks.
- Degraded mode: Unauditable surfaces, missing probes, or unfixable issues report per-category status with severity level and recommended action rather than blocking the entire audit.
- Fixture: `internal/tools/security_audit_test.go`
- Write scope: `internal/tools/security_audit.go`, `internal/tools/security_audit_test.go`, `cmd/gormes/security.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestSecurityAudit -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes security audit ships with --deep, --fix, and --json modes.
- Acceptance: gormes security audit --deep checks gateway auth, state integrity, channel security, shell blocklist, filesystem scoping, and credential redaction., --fix applies safe remediations (file permissions, auth token generation)., --json produces machine-readable output with per-category pass/fail/warn.
- Source refs: openclaw security audit --deep --fix --json CLI surface, docs/content/building-gormes/openclaw-platform-parity-audit.md, docs/content/building-gormes/must-have-features.md
- Unblocks: Production security posture
- Why now: P0 handoff; needs contract proof before closeout.

## 4. CLI contextual first-touch onboarding hint renderers

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: internal/cli exposes pure constants and renderers for Hermes-compatible contextual onboarding hints: BusyInputPromptFlag = `busy_input_prompt`, ToolProgressPromptFlag = `tool_progress_prompt`, BusyInputHint(surface, mode string) string for interrupt/queue/steer modes, and ToolProgressHint(surface string) string for long-running tool progress. CLI text is plain ASCII and gateway text may use channel-friendly wording, but both preserve the operator contract: explain what just happened, name `/busy` or `/verbose` follow-up commands, and state that the tip only shows once.
- Trust class: operator, gateway, system
- Ready when: CLI onboarding seen-state map helpers are complete and preserve arbitrary onboarding.seen flags., This row only adds pure hint constants/renderers; gateway/TUI/CLI startup binding remains row-backed., Tests use string fixtures and do not require config files, TTYs, channels, gateway managers, active turns, or tool execution.
- Not ready when: The slice persists onboarding.seen flags, reads config.toml, starts a gateway, inspects active turns, or emits channel messages., The slice changes busy-input policy, tool-progress rendering, slash-command behavior, or TUI state machines., The helper text mentions Hermes product commands or upstream config paths instead of Gormes-compatible operator guidance.
- Degraded mode: Unknown busy-input modes fall back to interrupt wording; unknown surfaces return CLI/plain text. The helpers do not read or write config and do not decide whether a hint has already been seen.
- Fixture: `internal/cli/onboarding_hints_test.go`
- Write scope: `internal/cli/onboarding_hints.go`, `internal/cli/onboarding_hints_test.go`, `internal/cli/onboarding_state.go`, `internal/cli/onboarding_state_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cli -run '^TestOnboardingHint\|^TestBusyInputHint\|^TestToolProgressHint' -count=1`, `go test ./internal/cli -run '^TestOnboardingSeen\|^TestMarkOnboardingSeen' -count=1`, `go run ./cmd/progress validate`
- Done signal: internal/cli fixtures prove busy-input and tool-progress onboarding flag constants plus CLI/gateway hint text for all modes, with no config, TTY, gateway, or tool-execution side effects.
- Acceptance: TestOnboardingHintFlagsMatchHermes proves BusyInputPromptFlag is `busy_input_prompt`, ToolProgressPromptFlag is `tool_progress_prompt`, and OpenClawResidueCleanupFlag remains `openclaw_residue_cleanup`., TestBusyInputHintCLIByMode proves interrupt, queue, and steer CLI hints mention the actual behavior, `/busy` follow-up commands, and `only shows once` without markdown-only wording., TestBusyInputHintGatewayByMode proves gateway hints for interrupt, queue, and steer are channel-safe, mention `/busy status` or the relevant `/busy` mode command, and do not include raw config keys., TestToolProgressHintCLIAndGateway proves tool-progress hints mention `/verbose`, the mode cycle `all -> new -> off`, and one-time display behavior., TestOnboardingHintsUnknownInputsDegrade proves unknown mode and surface inputs return non-empty CLI-safe fallback text without panic., Existing TestOnboardingSeen and TestMarkOnboardingSeen fixtures remain green, proving renderers do not change seen-state semantics.
- Source refs: ../hermes-agent/agent/onboarding.py:BUSY_INPUT_FLAG,TOOL_PROGRESS_FLAG,busy_input_hint_gateway,busy_input_hint_cli,tool_progress_hint_gateway,tool_progress_hint_cli,is_seen,mark_seen, ../hermes-agent/tests/agent/test_onboarding.py, internal/cli/onboarding_state.go, internal/cli/onboarding_state_test.go, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#gateway-channels-cron-api-tui-and-cli
- Unblocks: Busy-input first-touch hint binding, Tool-progress first-touch hint binding
- Why now: Unblocks Busy-input first-touch hint binding, Tool-progress first-touch hint binding.

## 5. Gormes onboard interactive action runner

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: `gormes onboard --wizard` in an interactive TTY turns the existing deterministic first-run plan into an action runner. It renders the same model -> provider -> auth -> gateway -> browser/CDP -> skills -> dashboard steps, shows each step's configured/missing status and skip warning before any action, and lets the operator run, skip, or review each step. Selected actions delegate through fakeable command seams to existing setup/model/auth/gateway/browser/skills/dashboard surfaces; tests must never start live providers, gateways, browsers, dashboards, TTS downloads, or vendor probes.
- Trust class: operator, system
- Ready when: Interactive Onboarding has the deterministic `gormes onboard --wizard --non-interactive` plan builder and configured-state prefill fixtures., Gormes setup top-level chooser menu and full-wizard shell rows are complete so model/provider/setup steps have existing command targets., Tests can inject action seams, prompt answers, configured-state input, and output buffers without relying on a real TTY, provider credential, gateway process, browser, or dashboard server.
- Not ready when: The slice starts live external services, opens a browser, contacts a provider, probes Browser/CDP, launches the dashboard server, or starts a messaging gateway in tests., The slice duplicates provider/model/auth setup logic instead of delegating to existing command seams., The slice removes or changes the noninteractive wizard plan output already used for CI diagnostics., The slice treats `gormes onboard` as strict Hermes command parity; upstream has `hermes setup`, while this command is a Gormes-owned diagnostic/action wrapper.
- Degraded mode: Non-TTY or `--non-interactive` invocations keep the current deterministic plan output and do not prompt. If an action seam is unavailable, the wizard returns onboard_action_row_backed with the step id and recommended command while preserving the remaining plan.
- Fixture: `cmd/gormes/onboard_wizard_test.go; internal/cli/onboard_test.go::TestOnboardPlan*`
- Write scope: `cmd/gormes/onboard.go`, `cmd/gormes/onboard_wizard_test.go`, `cmd/gormes/skills_onboard_test.go`, `internal/cli/onboard.go`, `internal/cli/onboard_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes -run '^TestOnboardWizard' -count=1`, `go test ./internal/cli -run TestOnboard -count=1`, `go run ./cmd/progress validate`
- Done signal: Onboard wizard fixtures prove interactive action prompting, skip-warning rendering, configured-state defaults, delegated/row-backed step handlers, and noninteractive no-prompt behavior without live external services.
- Acceptance: TestOnboardWizardInteractivePromptsForStepActions proves interactive `gormes onboard --wizard` renders all seven plan steps in order and asks run/skip/review for each step through an injected prompt seam., TestOnboardWizardSkipWarningsBeforeSkip proves skipping a missing step prints that step's skip warning before continuing., TestOnboardWizardConfiguredStepsArePrefilled proves configured provider/model/auth/gateway/browser state changes the step status/detail and defaults the action to review or skip instead of blindly reconfiguring., TestOnboardWizardDelegatesSelectedActions proves selected model/provider/auth/gateway/browser/skills/dashboard steps call injected action handlers with the step id and recommended command., TestOnboardWizardRowBackedActionEvidence proves unavailable handlers return onboard_action_row_backed with the step id and recommended command without aborting the whole plan., TestOnboardWizardNonInteractiveStillDoesNotPrompt proves `--non-interactive` and non-TTY invocations keep the deterministic plan and never invoke prompt or action seams., Existing TestOnboardWizardNonInteractiveShowsOrderedPlanAndSkipWarnings and internal/cli TestOnboardPlan* fixtures remain green.
- Source refs: ../hermes-agent/hermes_cli/main.py:8369-8403:setup_parser, ../hermes-agent/hermes_cli/setup.py:217-253:prompt_choice, ../hermes-agent/hermes_cli/setup.py:2953-3245:run_setup_wizard,_run_first_time_quick_setup,_run_quick_setup, ../hermes-agent/agent/onboarding.py:busy_input_hint_cli,tool_progress_hint_cli,openclaw_residue_hint_cli, cmd/gormes/onboard.go, internal/cli/onboard.go, cmd/gormes/setup.go, cmd/gormes/auth.go, cmd/gormes/dashboard.go
- Unblocks: First-run user experience, Interactive Onboarding
- Why now: Unblocks First-run user experience, Interactive Onboarding.

## 6. ACP Client Bridge Mode

- Phase: 5 / 5.H
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Complete the ACP integration with client bridge mode: gormes acp client connects to the Go-native ACP server (5.H server side is validated) with session key/label resolution, reset-session capability, require-existing guard, provenance modes (off/meta/meta+receipt), and --no-prefix-cwd flag. Match OpenClaw's ACP bridge surface.
- Trust class: operator, system
- Ready when: ACP server side (5.H) is validated — server manifest, auth, session, tools, permissions, events., Gateway session store supports key/label resolution.
- Not ready when: The row requires a running Hermes Python process., The row exposes unauthenticated ACP endpoints.
- Degraded mode: Unsupported ACP provider, missing auth, session key not found, or permission prompt timeout returns explicit acp_client_row_backed evidence with available fallback modes.
- Fixture: `internal/acp/client_test.go`
- Write scope: `internal/acp/client.go`, `internal/acp/client_test.go`, `cmd/gormes/acp.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/acp -run TestClient -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes acp client ships with session key/label resolution and provenance modes.
- Acceptance: gormes acp client --session agent:main:main connects to local ACP server., Provenance mode meta+receipt writes signed receipts., --reset-session clears and reinitializes the session key., --require-existing fails when session does not exist.
- Source refs: openclaw acp client CLI surface, internal/acp/server_manifest_test.go, docs/content/building-gormes/openclaw-platform-parity-audit.md
- Unblocks: Multi-agent interoperability, Editor integrations
- Why now: Unblocks Multi-agent interoperability, Editor integrations.

## 7. Extension Lifecycle Hook System

- Phase: 5 / 5.I
- Owner: `tools`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: Port agent-zero extension lifecycle hook system: register extensions at 8+ lifecycle points (agent_init, monologue_start/end, message_loop_start/end, before_main_llm_call, prompt_before/after, stream_chunk, tool_before/after, context_deleted). Extension chain executes in registration order with per-extension timeout and panic isolation.
- Trust class: operator, system
- Ready when: Kernel state machine transitions are well-defined., Plugin registry supports lifecycle callback registration.
- Not ready when: The row introduces Python dependency., The row adds hooks without timeout/panic recovery per extension.
- Degraded mode: Extension load failure, timeout, or panic reports per-extension status with degraded extension skipped. No single extension failure blocks the agent turn.
- Fixture: `internal/kernel/extensions_test.go`
- Write scope: `internal/kernel/extensions.go`, `internal/kernel/extensions_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/kernel -run TestExtensions -count=1`, `go run ./cmd/progress validate`
- Done signal: Extension lifecycle hook system ships with 8+ hook points and per-extension error isolation.
- Acceptance: Extensions register for monologue_start, message_loop_start, prompt_before, prompt_after, stream_chunk, tool_before, tool_after hooks., Extension chain executes in registration order., Extension timeout or panic does not crash agent turn., gormes extensions list shows registered extensions with hook points.
- Source refs: agent-zero helpers/extension.py (@extensible decorator), agent-zero agent.py (hook points), docs/content/building-gormes/agent-zero-feature-analysis.md, internal/kernel/kernel.go
- Unblocks: Plugin ecosystem, Skill injection pipeline
- Why now: Unblocks Plugin ecosystem, Skill injection pipeline.

## 8. System Events, Heartbeat, and Presence

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Port OpenClaw's system event surface: gormes system event enqueues a system event and optionally triggers a heartbeat; gormes system heartbeat shows and controls heartbeat state; gormes system presence lists system presence entries. Events are written to the audit ledger (JSONL) and surfaced in gormes status.
- Trust class: operator, system
- Ready when: Audit JSONL ledger (internal/audit/) is operational., Session health monitoring (5.N) provides heartbeat data.
- Not ready when: The row introduces a new event bus or message queue., The row depends on external monitoring services.
- Degraded mode: Missing audit ledger, event queue full, or heartbeat disabled reports system_unavailable with ledger path/error details.
- Fixture: `internal/tools/system_events_test.go`
- Write scope: `internal/tools/system_events.go`, `internal/tools/system_events_test.go`, `cmd/gormes/system.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestSystemEvents -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes system ships with event, heartbeat, and presence subcommands.
- Acceptance: gormes system event 'gateway restart' enqueues event with timestamp., gormes system heartbeat shows on/off state and last beat time., gormes system presence lists active components with last-seen times.
- Source refs: openclaw system event/heartbeat/presence CLI surface, internal/audit/ (JSONL audit log), docs/content/building-gormes/openclaw-platform-parity-audit.md
- Unblocks: Operator observability, Gateway discover/probe diagnostics
- Why now: Unblocks Operator observability, Gateway discover/probe diagnostics.

## 9. Gateway Discover and Probe

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Port OpenClaw's gateway network discovery: gormes gateway discover finds local gateways via Bonjour/mDNS; gormes gateway probe shows gateway reachability + discovery + health + status summary; gormes gateway usage-cost fetches usage cost summary from session logs.
- Trust class: operator
- Ready when: Gateway status and health commands are operational., Session store has usage/cost data accessible.
- Not ready when: The row requires Tailscale or CoreDNS for basic discovery., The row sends unauthenticated probe requests.
- Degraded mode: No gateways discovered, probe timeout, or usage data unavailable reports per-endpoint status with failure reason.
- Fixture: `internal/tools/gateway_discover_test.go`
- Write scope: `internal/tools/gateway_discover.go`, `internal/tools/gateway_discover_test.go`, `cmd/gormes/gateway_discover.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestGatewayDiscover -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes gateway discover/probe/usage-cost ships.
- Acceptance: gormes gateway discover lists local gateways with addresses and ports., gormes gateway probe shows reachability + discovery + health + status., gormes gateway usage-cost shows per-session and aggregate token costs.
- Source refs: openclaw gateway discover/probe/usage-cost CLI surface, docs/content/building-gormes/openclaw-platform-parity-audit.md
- Unblocks: Multi-instance fleet management
- Why now: Unblocks Multi-instance fleet management.

## 10. Channels Capabilities Introspection

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Port OpenClaw's channels capabilities: gormes channels capabilities shows provider capabilities (intents/scopes + supported features) for each configured channel. Enables operators to understand what each channel adapter supports before configuring it.
- Trust class: operator
- Ready when: Channel adapters expose capability metadata., CLI channel command routing is defined.
- Not ready when: The row hardcodes per-channel capabilities., The row requires live channel connections for capability discovery.
- Degraded mode: Unconfigured channel, missing adapter, or capability query failure reports per-channel status with 'unknown' capability rather than crashing.
- Fixture: `internal/channels/capabilities_test.go`
- Write scope: `internal/channels/capabilities.go`, `internal/channels/capabilities_test.go`, `cmd/gormes/channels_capabilities.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels -run TestCapabilities -count=1`, `go run ./cmd/progress validate`
- Done signal: gormes channels capabilities ships for all configured channel adapters.
- Acceptance: gormes channels capabilities --channel telegram shows intents, scopes, and supported features., Output includes media support, command support, and format limitations per channel., Unconfigured channels show 'not configured' status.
- Source refs: openclaw channels capabilities CLI surface, internal/channels/* (adapter implementations), docs/content/building-gormes/openclaw-platform-parity-audit.md
- Unblocks: Channel configuration UX
- Why now: Unblocks Channel configuration UX.

<!-- PROGRESS:END -->
