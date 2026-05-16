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
## 1. Termux real-device smoke evidence

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

## 2. Termux remote execution guidance

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

## 3. Gormes setup model step uses the dynamic provider-tracked model picker

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: The `gormes setup` Inference Provider section must present, for the operator's selected/active provider, the same dynamic per-provider callable-model list the `gormes model` and gateway/native-TUI `/model` pickers already use (internal/hermes.ListPickerProviders), instead of the legacy bare free-text prompt with at most five static suggestions. Two coupled defects: (1) the model prompt provider must equal the provider just selected/active in the section — the transcript witness shows Active provider "OpenAI Codex" but the prompt reads "Model for openrouter [gpt-5.5]", because runSetupInferenceProviderSection does not carry the chosen/active provider into runSetupActiveProviderModelPicker (it resolves to the provider-catalog default, openrouter); (2) cmd/gormes/model.go:promptModelChoice is fed defaultModelCatalogSuggestions(provider) = hermes.ProviderModelCatalogSuggestions(provider, nil) — the nil disables the live/dynamic catalog, so the operator never sees the provider's actual model set. The fix wires the existing ListPickerProviders-backed picker (the port already consumed by internal/gateway/model_picker.go and internal/tui/slash_model.go) into the setup model step with provider continuity, while preserving q/cancel, Enter-keeps-current, and `gormes setup model --non-interactive` default behavior. This supersedes the deliberately-static promptModelChoice seam introduced by the completed row at progress.json:19063; it must not fork a second picker or change the already-complete gateway/`gormes model`/TUI picker behavior.
- Trust class: -
- Ready when: ListPickerProviders + model-catalog infra are complete on main (rows 2.B.5, 4.D, 5.O, 5.Q)., Tests can drive the setup model step with a fake/injected provider+catalog (no live provider network) and assert the listed models and the prompt's provider string.
- Not ready when: The slice forks a second model-picker implementation instead of reusing internal/hermes.ListPickerProviders., It changes the already-complete gateway `/model`, `gormes model`, or native-TUI picker behavior., `gormes setup model --non-interactive` stops producing defaults without prompts, or q/cancel and Enter-keeps-current regress., The model prompt still shows a provider different from the selected/active provider.
- Degraded mode: If the dynamic per-provider catalog is unavailable (offline, no creds, empty provider list), the setup model step falls back to the current static suggestions + free-text entry rather than blocking setup, and reports the degraded source instead of silently showing the wrong provider. Non-interactive setup keeps its existing default-without-prompt behavior.
- Fixture: `cmd/gormes/setup_model_picker_test.go`
- Write scope: `cmd/gormes/setup.go`, `cmd/gormes/model.go`, `cmd/gormes/setup_model_picker_test.go`, `internal/hermes/picker_providers.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes -run 'TestSetupModelPicker\|TestSetup.*Provider\|TestRunSetupActiveProviderModelPicker' -count=1`, `go test ./cmd/gormes ./internal/cli ./internal/hermes -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Setup fixtures prove the model step lists the active provider's ListPickerProviders models, the prompt provider matches the selected/active provider, q/cancel + keep-current + non-interactive defaults are preserved, and the gateway/`gormes model`/TUI pickers are unchanged; source evidence cites Hermes 6784c8079 setup_model_provider/list_picker_providers.
- Acceptance: In `gormes setup`, after selecting/confirming a provider, the model step lists that provider's ListPickerProviders model set (not the ≤5 static nil-catalog suggestions)., The model prompt's provider equals the selected/active provider (no "Model for openrouter" when the active provider is OpenAI Codex / openai-codex)., q / cancel aborts with no change; empty input keeps the current model; a chosen model persists via the existing config write path., `gormes setup model --non-interactive` still resolves a default model without prompting., Gateway `/model`, `gormes model`, and native-TUI pickers are unchanged (regression-fenced).
- Source refs: ../hermes-agent/hermes_cli/setup.py@6784c8079:789:setup_model_provider (delegates to select_provider_and_model for provider selection + model picking), ../hermes-agent/hermes_cli/main.py@6784c8079:select_provider_and_model, ../hermes-agent/hermes_cli/model_switch.py@6784c8079:1720:list_picker_providers (per-provider callable-model list; OpenRouter live-filtered; empty providers dropped), ../hermes-agent/hermes_cli/setup.py@6784c8079:3122:SETUP_SECTIONS / 3132:run_setup_wizard, cmd/gormes/setup.go:751:runSetupInferenceProviderSection (provider not carried to model step), cmd/gormes/setup.go:1020:runSetupActiveProviderModelPicker (provider := current.Provider; calls promptModelChoice with static suggestions), cmd/gormes/model.go:173:promptModelChoice / cmd/gormes/model.go:169:defaultModelCatalogSuggestions (passes nil → no live catalog), internal/hermes/picker_providers.go:24:ListPickerProviders / :7:PickerProvider (existing dynamic picker port), internal/gateway/model_picker.go, internal/tui/slash_model.go (existing consumers — reuse, do not fork), docs/content/building-gormes/architecture_plan/progress.json:19063 (the static promptModelChoice seam this supersedes), Completed prerequisites: 2.B.5 Gateway /model interactive provider/model picker; 4.D Model catalog cache + preferred-provider live merge; 5.O Gormes model interactive provider/model picker; 5.Q Native TUI /model slash command binding
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Agentic-porting-kit public repo scaffold

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
