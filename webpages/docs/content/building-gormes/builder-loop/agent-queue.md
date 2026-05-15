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
## 1. Bubble Tea Messaging Platforms setup: Telegram-first Hermes fidelity

- Phase: 5 / 5.O
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: `gormes setup gateway` becomes the Bubble Tea-only Messaging Platforms setup surface for TTY users and the source of channel setup truth for first-run setup. The first executable slice ships Telegram end to end: token capture and validation, allowlist/pairing/open access policy, structured home_channel config, redacted review-before-write, `--plan`, non-TTY guidance, GORMES_* plus Hermes-compatible env aliases, explicit Hermes migration mapping, channel-scoped offline doctor evidence, and one consolidated gateway lifecycle recommendation after selected flows complete. Runtime paths stay Gormes-owned and normal runtime never reads Hermes config or dotenv files.
- Trust class: operator, gateway, system
- Ready when: Existing setup gateway checklist row is complete and exposes a fakeable section seam., internal/tui/wizard Bubble Tea steps are available for Pick, Text, Password, Confirm, defaults, abort, and non-TTY refusal., Cross-agent config isolation remains the active rule: Hermes config files are read only by explicit migration commands., Tests can use temp GORMES_HOME/XDG roots, fake TTY input, fake gateway lifecycle seams, and synthetic Telegram tokens only.
- Not ready when: The slice reintroduces raw key-reading or line-menu navigation for TTY setup., The implementation reads HERMES_HOME, ~/.hermes/config.yaml, or ~/.hermes/.env during normal runtime or normal setup., The slice starts a live gateway, calls Telegram APIs, opens WhatsApp QR pairing, or contacts providers in unit tests., The slice writes raw tokens into config.toml, setup output, doctor output, capabilities JSON, migration reports, or progress evidence., The slice tries to implement Discord, Slack, WhatsApp QR, plugin channels, platform_toolsets, or full config-template parity in the same pass.
- Degraded mode: Non-TTY setup returns setup_gateway_requires_tty or renders `--plan` without raw key handling; token-only Telegram config is partial until an access policy is chosen; missing home channel is warning evidence, not unconfigured; legacy env/config aliases remain read-compatible but setup writes Gormes-owned names and redacts all secrets.
- Fixture: `cmd/gormes/setup_gateway_bubbletea_test.go; internal/gateway/channel_setup_test.go; internal/config/telegram_config_test.go; internal/migrate/hermes/telegram_mapping_test.go`
- Write scope: `cmd/gormes/setup.go`, `cmd/gormes/setup_gateway_test.go`, `cmd/gormes/setup_gateway_bubbletea_test.go`, `cmd/gormes/gateway.go`, `cmd/gormes/doctor.go`, `internal/tui/wizard/steps.go`, `internal/tui/wizard/wizard.go`, `internal/tui/wizard/wizard_test.go`, `internal/gateway/channel_setup.go`, `internal/gateway/channel_setup_test.go`, `internal/config/config.go`, `internal/config/writer.go`, `internal/config/telegram_config_test.go`, `internal/migrate/hermes/manifest.go`, `internal/migrate/hermes/writer.go`, `internal/migrate/hermes/telegram_mapping_test.go`, `webpages/docs/content/building-gormes/architecture_plan/messaging-platform-setup-fidelity.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes -run 'TestSetupGateway.*BubbleTea\|TestSetupGateway.*Telegram\|TestSetupGateway.*Plan\|TestSetupGateway.*Redact\|TestSetupGateway.*NonTTY' -count=1`, `go test ./internal/gateway -run 'TestChannelSetup' -count=1`, `go test ./internal/config -run 'TestTelegram.*HomeChannel\|TestTelegram.*EnvAlias\|TestConfigWriter.*Telegram' -count=1`, `go test ./internal/migrate/hermes -run 'TestHermes.*Telegram\|TestHermesConfigWriter.*Telegram' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Telegram-first Messaging Platforms setup fixtures prove Bubble Tea-only TTY interaction, non-TTY/plan behavior, staged redacted writes, env/migration aliases, structured home_channel config, offline doctor/capabilities redaction, and consolidated gateway lifecycle advice without live platform calls.
- Acceptance: TTY `gormes setup gateway` uses Bubble Tea for all interactive selection and data entry, and tests prove legacy raw menu markers or escape bytes are absent from the setup transcript., `gormes setup gateway --plan` prints registered channels with unconfigured/partial/configured/paired/running/failed status, required fields, redacted current values, planned writes, and gateway action without writing files or contacting live APIs., Telegram setup stages bot token, access policy, and home channel in Hermes order; token-only state is partial, allowlist is the default safe policy, open access is an explicit risky choice, and group setup renders BotFather privacy/admin/remove-readd guidance., Applying staged changes writes secrets to the Gormes dotenv or SecretRef and non-secret policy to native TOML, preferring `GORMES_TELEGRAM_BOT_TOKEN` while reading existing `GORMES_TELEGRAM_TOKEN`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ALLOWED_USERS`, `TELEGRAM_HOME_CHANNEL`, and supported `HERMES_*` aliases at lower precedence., Config supports structured `telegram.home_channel` with string chat/thread ids, reads legacy `allowed_chat_id`, imports `TELEGRAM_HOME_CHANNEL` into `telegram.home_channel.chat_id`, and imports `TELEGRAM_ALLOWED_USERS` into numeric `telegram.allowed_user_ids`., Review-before-write shows a redacted diff summary; cancel before apply writes nothing; explicit partial apply records partial channel status., Channel-scoped doctor and capabilities output are redacted and offline by default, while a live connection test is opt-in only., Gateway lifecycle advice is emitted once after all selected channel flows and never silently restarts a running gateway.
- Source refs: webpages/docs/content/building-gormes/architecture_plan/messaging-platform-setup-fidelity.md, hermes-agent@9ed751b96:hermes_cli/setup.py:setup_gateway, hermes-agent@9ed751b96:hermes_cli/gateway.py:_PLATFORMS,_setup_standard_platform,_setup_whatsapp, hermes-agent@9ed751b96:gateway/config.py:PlatformConfig,HomeChannel,_apply_env_overrides, hermes-agent@9ed751b96:cli-config.yaml.example:Gateway Platform Settings,platform_toolsets, hermes-agent@9ed751b96:website/docs/user-guide/messaging/telegram.md, hermes-agent@9ed751b96:website/docs/user-guide/messaging/whatsapp.md, cmd/gormes/setup.go, cmd/gormes/gateway.go, internal/tui/wizard, internal/config/config.go, internal/config/writer.go, internal/migrate/hermes/manifest.go, internal/migrate/hermes/writer.go
- Unblocks: WhatsApp setup handoff from Messaging Platforms, Gateway channel setup registry for plugin/manual channels, Hermes config template gateway/platform_toolsets parity
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Coding-agent delegation: Phase 1 scaffold (internal/codingagents)

- Phase: 2 / 2.A
- Owner: `tools`
- Size: `medium`
- Status: `in_progress`
- Priority: `P1`
- Contract: Shared internal/codingagents package providing the CodingAgent interface, CodingAgentRequest/Result, mode constants, binary availability detection, workspace guard with default deny list, git snapshot/diff helper, and prompt wrapper. No tools are registered in this slice; adapters and registry exposure land in later phases.
- Trust class: operator, system
- Ready when: Shared CodingAgent interface and CodingAgentRequest/Result cover workspace, prompt, mode, edit permissions, timeout, files-changed, stdout/stderr, and git diff., Availability checks detect codex, claude/claude-code, and opencode binaries and report unavailable cleanly., Workspace guard refuses empty, ambiguous, denied, and outside-allowed inputs and accepts paths under an allowed root., Git snapshot/diff helper captures HEAD/branch/dirty/files for a real repo and returns ErrNotAGitRepo on a non-git dir., Prompt wrapper restates workspace/mode/task and injects gormes-repo rules when the workspace is a gormes-agent checkout.
- Not ready when: Adapters or tool descriptors register coding_agent / codex_run / claude_code_run / opencode_run before the umbrella's later phases., Results omit files_changed or git_diff across adapters., Workspace identifiers bypass the guard via raw-path voice input.
- Degraded mode: Without the scaffold, later phases cannot compile codex/claude-code/opencode adapters against a shared contract; doctor cannot probe coding-agent binaries.
- Fixture: `internal/codingagents`
- Write scope: `internal/codingagents/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/codingagents/... -count=1`, `go vet ./internal/codingagents/...`, `go run ./cmd/progress validate`
- Done signal: go test ./internal/codingagents/... -count=1 passes locally with the scaffold, availability probe, workspace guard, git snapshot, and prompt wrap covered by unit tests.
- Acceptance: internal/codingagents compiles and tests pass on stdlib only., WorkspaceGuard returns typed sentinels (ErrWorkspaceEmpty/Ambiguous/OutsideAllowed/Denied) and refuses $HOME, /, and ~/.ssh by default., DetectAll returns availability entries for codex, claude, claude-code, and opencode., TakeSnapshot + DiffBetween capture HEAD, dirty status, and a unified diff with file list on a temp repo; non-git dirs raise ErrNotAGitRepo.
- Source refs: User design: 2026-05-13 coding-agent delegation plan, internal/codingagents/codingagents.go, internal/codingagents/workspace.go, internal/codingagents/git_snapshot.go
- Why now: Already active; contract metadata keeps execution bounded.

## 3. Agentic-porting-kit public repo scaffold

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
