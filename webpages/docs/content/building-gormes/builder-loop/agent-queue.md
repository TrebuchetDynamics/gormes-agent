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
## 1. Termux gateway foreground tmux lifecycle

- Phase: 1 / 5.X
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Gateway lifecycle commands and docs present a Termux-specific foreground/tmux model: Telegram/Discord/Slack gateways are supported from a foreground shell or tmux session, systemd/Windows service assumptions are not advertised, and doctor/status guidance names termux-wake-lock plus Android battery settings as best-effort survival aids. The implementation must preserve the same gateway command names and JSON contracts as desktop Linux.
- Trust class: operator, gateway, system
- Ready when: Termux runtime doctor check is complete., Gateway command tests can run with temp GORMES_HOME and fake runtime status stores., Termux install docs identify foreground/tmux as the supported local gateway model.
- Not ready when: The command tries to install systemd units, Android services, or Termux:Boot entries by default., Doctor/status claims guaranteed unattended background uptime on Android., Gateway command names, flags, or JSON shapes diverge from desktop Linux only for Termux.
- Degraded mode: If Termux lacks tmux or termux-wake-lock, gateway startup remains possible but doctor/status emits WARN guidance. Android process death is treated as recoverable operator environment behavior, not a Gormes crash.
- Fixture: `cmd/gormes gateway/doctor fixtures under synthetic Termux env`
- Write scope: `cmd/gormes/`, `internal/gateway/`, `internal/doctor/`, `webpages/docs/content/install/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes -run 'Test.*Termux.*Gateway\|TestDoctorCommand_JSONIncludesTermuxRuntimeWhenDetected' -count=1`, `go test ./internal/gateway -count=1`, `go run ./cmd/progress validate`
- Done signal: Gateway fixtures and docs prove Termux uses the same operator CLI with foreground/tmux lifecycle guidance and bounded Android process-survival claims.
- Acceptance: Synthetic Termux doctor/status output includes foreground/tmux and wake-lock guidance., Gateway start/status/stop command surfaces keep the same names and JSON contracts under synthetic Termux env., Termux docs explain tmux, termux-wake-lock, battery optimization, and Termux:Boot as operator-managed aids., No test starts live Telegram/Discord/Slack connections or Android services.
- Source refs: cmd/gormes/gateway.go, cmd/gormes/gateway_status.go, cmd/gormes/doctor.go, internal/gateway/status.go, internal/doctor/termux.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. Termux notification bridge via termux-api

- Phase: 1 / 5.X
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Add an optional Termux notification adapter that shells out to termux-notification only when Termux and the command are detected. Gateway/long-run status can emit Android notifications through this adapter, while non-Termux hosts and Termux hosts without Termux:API degrade to structured no-op/WARN evidence. The adapter must redact secrets and never make termux-api a hard dependency.
- Trust class: operator, gateway, system
- Ready when: Termux runtime doctor check is complete., A small notification sender interface can be injected into gateway/status paths without changing core gateway contracts., Tests can fake command lookup and command execution.
- Not ready when: The adapter invokes live termux-notification in tests., Missing Termux:API fails doctor, gateway, or long-running tasks., Notification text can include provider tokens, bot tokens, prompts containing secrets, or raw command output without redaction.
- Degraded mode: Missing termux-notification or missing Termux:API app returns optional_notification_unavailable evidence; Gormes continues normally without Android notifications.
- Fixture: `internal/gateway or internal/tools Termux notification adapter tests with fake exec runner`
- Write scope: `internal/gateway/`, `internal/tools/`, `internal/doctor/`, `cmd/gormes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway ./internal/tools -run 'Termux.*Notification\|Notification.*Termux' -count=1`, `go test ./cmd/gormes -run 'Termux\|Notification' -count=1`, `go run ./cmd/progress validate`
- Done signal: Optional termux-api notification adapter sends through fake exec under Termux and degrades cleanly everywhere else.
- Acceptance: Fake-exec tests prove Termux notification sends title/body through termux-notification with bounded arguments., Non-Termux and missing-command tests return structured no-op/WARN evidence., Doctor/status output references notification availability without requiring Termux:API., Secret redaction tests prove tokens are not passed into notification bodies.
- Source refs: internal/doctor/termux.go, internal/tools/voice_mode_env.go:termux-api detection precedent, internal/gateway/, cmd/gormes/kanban_notify_test.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Termux real-device smoke evidence

- Phase: 1 / 5.X
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Capture a dated real-device no-root Android Termux smoke record for the current release: install via repo-root install.sh release asset, run gormes version, gormes doctor --offline --json, gormes config check, initialize SQLite/Goncho state, and run a provider-backed gormes chat -q "hello from Termux" when a test credential is available. The evidence must record Android/Termux versions, device arch, install method, and any caveats without leaking credentials.
- Trust class: operator, system
- Ready when: Termux runtime doctor check is complete., Termux install and release smoke guide is complete., A real no-root Android arm64/aarch64 Termux environment is available to the operator.
- Not ready when: The evidence is only CI simulation or local Linux fake TERMUX_VERSION output., The smoke transcript includes raw provider keys, bot tokens, device-private paths beyond normal Termux paths, or personal chat IDs., The smoke uses source build as the primary install path unless the release asset is explicitly unavailable.
- Degraded mode: If no provider credential is available, record provider-backed oneshot as skipped with credential-unavailable evidence; local install/version/doctor/config/Goncho smoke remains required.
- Fixture: `webpages/docs/content/install/termux-smoke.md or release evidence note`
- Write scope: `webpages/docs/content/install/`, `docs/content/building-gormes/architecture_plan/progress.json`, `README.md`
- Test commands: -
- No test required: Manual real-device evidence row; CI simulation cannot replace the Android smoke transcript.
- Done signal: A dated redacted real-device Termux smoke record is checked in and linked from the install docs/progress row.
- Acceptance: Evidence records exact date, device arch, Android version, Termux version, and Gormes version/commit., Evidence shows install.sh release-binary path into $PREFIX/bin/gormes., Evidence includes gormes version, gormes doctor --offline --json, gormes config check, and SQLite/Goncho initialization outputs or redacted summaries., Provider-backed gormes chat -q succeeds or is explicitly skipped for missing test credential., The public compatibility claim remains bounded to the proven support matrix.
- Source refs: install.sh, cmd/gormes/version.go, cmd/gormes/doctor.go, cmd/gormes/config.go, cmd/gormes/goncho.go, internal/doctor/termux.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Termux remote execution guidance

- Phase: 1 / 5.X
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Document and, where useful, add setup/status guidance for using Termux Gormes as the mobile operator/controller while SSHing to stronger machines for heavy builds, Docker, local browser automation, and GPU/local model inference. The guidance must preserve PC-like local Gormes CLI behavior while making remote execution the credible path for workstation/server workloads.
- Trust class: operator, system
- Ready when: Termux runtime doctor check is complete., Termux install docs exist., Current shell/terminal/SSH tool behavior is documented from existing Gormes command surfaces.
- Not ready when: The docs claim local Termux can run Docker, heavy browser automation, GPU/local LLM, or large test suites like a workstation., The guidance introduces a new top-level gormes run command instead of existing gormes chat -q/gateway surfaces., The guidance requires root, privileged Android features, or a specific private server.
- Degraded mode: If no remote host is configured, Termux remains a local CLI/TUI/gateway runtime and doctor/docs explain which heavy workloads are out of local scope.
- Fixture: `webpages/docs/content/install/ Termux remote-execution docs`
- Write scope: `webpages/docs/content/install/`, `README.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./webpages/docs -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Termux docs make the mobile-control-plane plus remote-executor architecture concrete without expanding local Termux support beyond proven capability.
- Acceptance: Docs include the architecture: phone equals Gormes controller/light executor, remote host equals heavy build/browser/Docker/GPU executor., Docs give concrete SSH/tmux command examples without adding a new top-level gormes run command., Doctor or install docs point Termux users at remote execution for local browser/Docker/GPU-heavy workloads., The support matrix remains explicit about what is local, optional, and remote/degraded.
- Source refs: cmd/gormes/doctor.go, internal/doctor/termux.go, internal/tools/, webpages/docs/content/install/linux-macos.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Agentic-porting-kit public repo scaffold

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
