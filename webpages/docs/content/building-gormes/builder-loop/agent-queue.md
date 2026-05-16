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
## 1. Gormes update release planner and dry-run contract

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: `gormes update` must distinguish normal release-installed operators from managed source checkouts before it mutates anything. For release installs, the command plans against trusted GitHub Releases: detect install layout, current version/build, OS/arch artifact name, channel policy (`stable` by default, `development` explicit), latest release metadata, snapshot path, components to update, blocked reasons, and whether an update is available. `gormes update --dry-run` prints the exact non-mutating plan; `gormes update --check` performs no mutation and exits 0 when current, 10 when an update is available, and nonzero for check errors. Existing source-checkout behavior from `Self-update command lifecycle safety` remains the managed/dev path and must not be silently used for release installs.
- Trust class: operator, system
- Ready when: The existing source-checkout `gormes update` command and lifecycle seams are complete and validated., Installer release-binary fetch behavior already defines trusted repo, platform artifact names, release channels, and SHA-256 sidecar expectations., Tests can inject install layout, current build, release metadata, OS/arch, channel, and clock without network or filesystem mutation.
- Not ready when: The slice downloads or swaps binaries, syncs skills/assets, restarts services, or pulls git remotes., Release-installed binaries silently fall back to managed source checkout mutation., `--check` or `--dry-run` writes update logs, snapshots, config, skills, sessions, services, or source checkout state., Exit codes for current/update-available/error are not deterministic and tested.
- Degraded mode: Unknown install layout, unsupported OS/arch, missing release metadata, channel mismatch, dirty unmanaged source checkout, and network/API lookup failures produce typed plan blockers without changing binaries, assets, skills, services, config, credentials, sessions, memory, or source checkouts.
- Fixture: `cmd/gormes/update_release_plan_test.go; internal/cli/update_release_plan_test.go`
- Write scope: `cmd/gormes/update.go`, `cmd/gormes/update_release_plan_test.go`, `internal/cli/update_release_plan.go`, `internal/cli/update_release_plan_test.go`, `internal/cli/update_lifecycle.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./cmd/gormes ./internal/cli -run 'TestUpdateReleasePlanner\|TestUpdateCommandReleaseDryRun\|TestUpdateCommandCheck' -count=1`, `go test ./cmd/gormes ./internal/cli -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Update planner fixtures prove release/source/unknown layout detection, current-vs-target metadata, stable/development channel policy, dry-run output, --check exit codes, and zero mutation before any release artifact install path exists.
- Acceptance: Release install, managed source checkout, unmanaged source checkout, and unknown install layouts are classified with typed evidence., `gormes update --dry-run` prints current version/build, target version/build, source (`github_release` or `managed_source`), channel, artifact, snapshot path, component plan, and blockers without mutation., `gormes update --check` exits 0 when current, 10 when an update is available, and nonzero for check errors, with JSON and text reports carrying the same state., Stable vs development channel policy is explicit; release installs use GitHub Releases by default, managed source/dev installs use the existing source lifecycle only when detected or explicitly requested., No binary, asset, skill, service, config, credential, session, memory, git checkout, or update ledger file changes during planner/check/dry-run tests.
- Source refs: cmd/gormes/update.go:newUpdateCommandWithSeams (current source-checkout update command and injectable seams), internal/cli/update_lifecycle.go:RunUpdateLifecycle (completed managed source-checkout updater), cmd/gormes/version.go:newVersionCommand/newBuildProvenance (current version/build evidence), install.sh:decide_install_method + fetch_release_binary (release asset naming, GitHub latest release lookup, SHA-256 sidecar evidence), scripts/install.ps1:release binary fetch block (Windows release asset naming and SHA-256 sidecar evidence), internal/installtest/install_method_test.go (release-vs-source installer behavior), Completed row: Self-update command lifecycle safety (source-checkout update path remains intact)
- Unblocks: Gormes update verified binary swap and rollback
- Why now: Unblocks Gormes update verified binary swap and rollback.

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
